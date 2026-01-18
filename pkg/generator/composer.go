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
	PanelResourceMap     map[string]string // PanelIndex -> FileAPIURI
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
			// URLそのものをキーとしてアップロード/キャッシュ
			_, err := mc.getOrUploadPanelAsset(egCtx, panel.ReferenceURL)
			return err
		})
	}
	return eg.Wait()
}

// getOrUploadAsset はキャラクター用アセットを二重チェック付き singleflight で処理します。
func (mc *MangaComposer) getOrUploadAsset(ctx context.Context, charID, referenceURL string) (string, error) {
	mc.mu.RLock()
	uri, ok := mc.CharacterResourceMap[charID]
	mc.mu.RUnlock()
	if ok {
		return uri, nil
	}

	val, err, _ := mc.uploadGroup.Do(charID, func() (interface{}, error) {
		mc.mu.RLock()
		existingURI, ok := mc.CharacterResourceMap[charID]
		mc.mu.RUnlock()
		if ok {
			return existingURI, nil
		}

		uploadedURI, uploadErr := mc.AssetManager.UploadFile(ctx, referenceURL)
		if uploadErr != nil {
			return nil, uploadErr
		}

		mc.mu.Lock()
		mc.CharacterResourceMap[charID] = uploadedURI
		mc.mu.Unlock()
		return uploadedURI, nil
	})

	if err != nil {
		return "", err
	}
	return val.(string), nil
}

// getOrUploadPanelAsset は ReferenceURL をキーにキャッシュ制御を行います。
func (mc *MangaComposer) getOrUploadPanelAsset(ctx context.Context, referenceURL string) (string, error) {
	// 1. キャッシュ確認
	mc.mu.RLock()
	uri, ok := mc.PanelResourceMap[referenceURL]
	mc.mu.RUnlock()
	if ok {
		return uri, nil
	}

	// 2. singleflight で重複抑制（キーは URL そのもの）
	val, err, _ := mc.uploadGroup.Do(referenceURL, func() (interface{}, error) {
		// ダブルチェック
		mc.mu.RLock()
		existingURI, ok := mc.PanelResourceMap[referenceURL]
		mc.mu.RUnlock()
		if ok {
			return existingURI, nil
		}

		// アップロード実行
		uploadedURI, uploadErr := mc.AssetManager.UploadFile(ctx, referenceURL)
		if uploadErr != nil {
			return nil, uploadErr
		}

		// キャッシュ保存
		mc.mu.Lock()
		mc.PanelResourceMap[referenceURL] = uploadedURI
		mc.mu.Unlock()
		return uploadedURI, nil
	})

	if err != nil {
		return "", err
	}
	return val.(string), nil
}
