package generator

import (
	"context"
	"fmt"
	"sync"

	"github.com/shouni/go-manga-kit/pkg/domain"

	"github.com/shouni/gemini-image-kit/pkg/generator"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
)

type MangaComposer struct {
	AssetManager         generator.AssetManager
	ImageGenerator       generator.ImageGenerator
	CharactersMap        domain.CharactersMap
	RateLimiter          *rate.Limiter
	CharacterResourceMap map[string]string // CharacterID -> FileAPIURI
	PanelResourceMap     map[string]string // ReferenceURL -> FileAPIURI
	mu                   sync.RWMutex
	uploadGroup          singleflight.Group
}

// NewMangaComposer は MangaComposer の新しいインスタンスを初期化済みの状態で生成します。
func NewMangaComposer(
	assetMgr generator.AssetManager,
	imgGen generator.ImageGenerator,
	cm domain.CharactersMap,
	limiter *rate.Limiter,
) *MangaComposer {
	return &MangaComposer{
		AssetManager:         assetMgr,
		ImageGenerator:       imgGen,
		CharactersMap:        cm,
		RateLimiter:          limiter,
		CharacterResourceMap: make(map[string]string),
		PanelResourceMap:     make(map[string]string),
	}
}

// PrepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
func (mc *MangaComposer) PrepareCharacterResources(ctx context.Context, panels []domain.Panel) error {
	targetIDs := make(map[string]struct{})

	// デフォルトキャラクターをアップロード対象に追加
	if def := mc.CharactersMap.GetDefault(); def != nil && def.ReferenceURL != "" {
		targetIDs[def.ID] = struct{}{}
	}

	// パネルで使用されているキャラクターをアップロード対象に追加
	for _, id := range domain.Panels(panels).UniqueSpeakerIDs() {
		targetIDs[id] = struct{}{}
	}

	eg, egCtx := errgroup.WithContext(ctx)

	for id := range targetIDs {
		charID := id // ループ変数のキャプチャ
		eg.Go(func() error {
			char := mc.CharactersMap.GetCharacterWithDefault(charID)
			if char == nil || char.ReferenceURL == "" {
				return nil
			}

			_, err := mc.getOrUploadAsset(egCtx, char.ID, char.ReferenceURL)
			if err != nil {
				return fmt.Errorf("キャラクター '%s' のリソース準備に失敗しました: %w", charID, err)
			}
			return nil
		})
	}

	return eg.Wait()
}

// PreparePanelResources は各パネル固有の ReferenceURL を事前アップロードします。
func (mc *MangaComposer) PreparePanelResources(ctx context.Context, panels []domain.Panel) error {
	eg, egCtx := errgroup.WithContext(ctx)

	for _, panel := range panels {
		if panel.ReferenceURL == "" {
			continue
		}

		eg.Go(func() error {
			_, err := mc.getOrUploadPanelAsset(egCtx, panel.ReferenceURL)
			return err
		})
	}
	return eg.Wait()
}

// getOrUploadAsset はキャラクター用アセットをキャッシュ制御しつつ取得またはアップロードします。
func (mc *MangaComposer) getOrUploadAsset(ctx context.Context, charID, referenceURL string) (string, error) {
	return mc.getOrUploadResource(ctx, charID, referenceURL, mc.CharacterResourceMap)
}

// getOrUploadPanelAsset はパネル用参照URLをキャッシュ制御しつつ取得またはアップロードします。
func (mc *MangaComposer) getOrUploadPanelAsset(ctx context.Context, referenceURL string) (string, error) {
	// パネルアセットの場合、検索キーとソースURLは同一です。
	return mc.getOrUploadResource(ctx, referenceURL, referenceURL, mc.PanelResourceMap)
}

// getOrUploadResource は二重チェックロッキングと singleflight を用いてアセットアップロードの共通ロジックを提供します。
func (mc *MangaComposer) getOrUploadResource(ctx context.Context, key, referenceURL string, resourceMap map[string]string) (string, error) {
	// gemini-image-kit 側が ReferenceURL (gs://) を直接処理するため、
	// File API へのアップロードプロセスそのものをスキップします。 {
	if mc.AssetManager != nil && mc.AssetManager.IsVertexAI() && remoteio.IsGCSURI(referenceURL) {
		mc.mu.RLock()
		_, ok := resourceMap[key]
		mc.mu.RUnlock()

		if !ok {
			mc.mu.Lock()
			resourceMap[key] = ""
			mc.mu.Unlock()
		}
		return "", nil
	}

	// 最初のチェック: ロックを最小限にするための RLock
	mc.mu.RLock()
	uri, ok := resourceMap[key]
	mc.mu.RUnlock()
	if ok {
		return uri, nil
	}

	// 同一キーに対する同時リクエストを1つに集約（HTTP URL等の場合のみ）
	val, err, _ := mc.uploadGroup.Do(key, func() (interface{}, error) {
		mc.mu.RLock()
		existingURI, ok := resourceMap[key]
		mc.mu.RUnlock()
		if ok {
			return existingURI, nil
		}

		// ここで実際に File API (Google AI Studio) へアップロードされる
		uploadedURI, uploadErr := mc.AssetManager.UploadFile(ctx, referenceURL)
		if uploadErr != nil {
			return nil, uploadErr
		}

		mc.mu.Lock()
		resourceMap[key] = uploadedURI
		mc.mu.Unlock()
		return uploadedURI, nil
	})

	if err != nil {
		return "", err
	}

	return val.(string), nil
}
