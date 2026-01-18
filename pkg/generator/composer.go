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
	panelResourceMap     map[int]string    // PanelIndex -> FileAPIURI
	mu                   sync.RWMutex
	uploadGroup          singleflight.Group
}

// NewMangaComposer は MangaComposer の新しいインスタンスを、必要なマップを初期化した状態で生成します。
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
		panelResourceMap:     make(map[int]string),
	}
}

// PrepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
// コンストラクタでマップが初期化されていることを前提としているため、ここでのロック付き nil チェックは不要です。
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
				return fmt.Errorf("failed to prepare asset for character %s (resolved from speaker %s): %w", char.ID, speakerID, err)
			}
			return nil
		})
	}

	return eg.Wait()
}

// getOrUploadAsset は、内部的なキャッシュ（CharacterResourceMap）を利用し、
// 必要に応じて Gemini File API へのアップロードを singleflight で実行します。
func (mc *MangaComposer) getOrUploadAsset(ctx context.Context, charID, referenceURL string) (string, error) {
	val, err, _ := mc.uploadGroup.Do(charID, func() (interface{}, error) {
		// singleflight の実行中に他の goroutine が完了させている可能性があるため、マップを確認
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

	uri, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("unexpected return type from singleflight: %T", val)
	}
	return uri, nil
}
