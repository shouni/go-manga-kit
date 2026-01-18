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
	PanelResourceMap     map[int]string    // PanelIndex -> FileAPIURI
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
		PanelResourceMap:     make(map[int]string),
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
				return fmt.Errorf("failed to prepare asset for character %s (resolved from speaker %s): %w", char.ID, speakerID, err)
			}
			return nil
		})
	}

	return eg.Wait()
}

// getOrUploadAsset は内部的なキャッシュを利用し、必要に応じてアップロードを実行します（非公開メソッド）。
func (mc *MangaComposer) getOrUploadAsset(ctx context.Context, charID, referenceURL string) (string, error) {
	// RLock でキャッシュ（マップ）を素早く確認
	mc.mu.RLock()
	uri, ok := mc.CharacterResourceMap[charID]
	mc.mu.RUnlock()
	if ok {
		return uri, nil
	}

	val, err, _ := mc.uploadGroup.Do(charID, func() (interface{}, error) {
		// ingleflight で待機中に他のゴルーチンがアップロードを完了させている可能性があるため、コールバック内で再度マップを確認
		mc.mu.RLock()
		existingURI, ok := mc.CharacterResourceMap[charID]
		mc.mu.RUnlock()
		if ok {
			return existingURI, nil
		}

		// 本当に未アップロードの場合のみ、重い I/O 処理を実行
		uploadedURI, uploadErr := mc.AssetManager.UploadFile(ctx, referenceURL)
		if uploadErr != nil {
			return nil, uploadErr
		}

		// 結果を永続的なキャッシュマップに保存
		mc.mu.Lock()
		mc.CharacterResourceMap[charID] = uploadedURI
		mc.mu.Unlock()

		return uploadedURI, nil
	})

	if err != nil {
		return "", err
	}

	uri, ok = val.(string)
	if !ok {
		return "", fmt.Errorf("unexpected return type from singleflight: %T", val)
	}
	return uri, nil
}
