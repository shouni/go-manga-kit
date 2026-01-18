package generator

import (
	"context"
	"fmt"
	"sync"

	"github.com/shouni/go-manga-kit/pkg/domain"

	"github.com/shouni/gemini-image-kit/pkg/generator"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
)

type MangaComposer struct {
	AssetManager         generator.AssetManager
	ImageGenerator       generator.ImageGenerator
	PromptBuilder        ImagePromptBuilder
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
	pb ImagePromptBuilder,
	cm domain.CharactersMap,
	limiter *rate.Limiter,
) *MangaComposer {
	return &MangaComposer{
		AssetManager:         assetMgr,
		ImageGenerator:       imgGen,
		PromptBuilder:        pb,
		CharactersMap:        cm,
		RateLimiter:          limiter,
		CharacterResourceMap: make(map[string]string),
		PanelResourceMap:     make(map[string]string),
	}
}

// PrepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
func (mc *MangaComposer) PrepareCharacterResources(ctx context.Context, panels []domain.Panel) error {
	uniqueSpeakerIDs := domain.Panels(panels).UniqueSpeakerIDs()
	cm := mc.CharactersMap
	eg, egCtx := errgroup.WithContext(ctx)

	for _, id := range uniqueSpeakerIDs {
		speakerID := id
		eg.Go(func() error {
			char := cm.GetCharacterWithDefault(speakerID)
			if char == nil || char.ReferenceURL == "" {
				return nil
			}

			_, err := mc.getOrUploadAsset(egCtx, char.ID, char.ReferenceURL)
			if err != nil {
				return fmt.Errorf("failed to prepare character asset %s: %w", char.ID, err)
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
		panel := panel
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
	// 最初のチェック: ロックを最小限にするための RLock
	mc.mu.RLock()
	uri, ok := resourceMap[key]
	mc.mu.RUnlock()
	if ok {
		return uri, nil
	}

	// 同一キーに対する同時リクエストを1つに集約
	val, err, _ := mc.uploadGroup.Do(key, func() (interface{}, error) {
		// ダブルチェック: singleflight 待機中に他で完了している可能性があるため
		mc.mu.RLock()
		existingURI, ok := resourceMap[key]
		mc.mu.RUnlock()
		if ok {
			return existingURI, nil
		}

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

	res, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("unexpected type from upload group for key %q: expected string, got %T", key, val)
	}
	return res, nil
}
