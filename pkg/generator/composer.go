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
	panelResourceMap     map[int]string    // PanelIndex (or ID) -> FileAPIURI
	mu                   sync.RWMutex
	uploadGroup          singleflight.Group
}

// PrepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
func (mc *MangaComposer) PrepareCharacterResources(ctx context.Context, panels []domain.Panel) error {
	// マップの遅延初期化
	mc.mu.Lock()
	if mc.CharacterResourceMap == nil {
		mc.CharacterResourceMap = make(map[string]string)
	}
	mc.mu.Unlock()

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

			// キャッシュを考慮したアップロードロジックを呼び出し
			_, err := mc.getOrUploadAsset(egCtx, char.ID, char.ReferenceURL)
			if err != nil {
				return fmt.Errorf("failed to prepare asset for character %s (resolved from speaker %s): %w", char.ID, speakerID, err)
			}
			return nil
		})
	}

	return eg.Wait()
}

// getOrUploadAsset は指定されたキャラクターのアセットが既にアップロード済みであればその URI を返し、
// 未アップロードであれば File API へアップロードしてマップに格納します。
// singleflight を使用して、同一キャラクターの同時リクエストを集約します。
func (mc *MangaComposer) getOrUploadAsset(ctx context.Context, charID, referenceURL string) (string, error) {
	val, err, _ := mc.uploadGroup.Do(charID, func() (interface{}, error) {
		// 【重要】singleflight.Group は「現在進行中の呼び出し」はまとめますが、
		// 過去に完了した結果を永続的にキャッシュする機能はありません。
		// そのため、コールバック内で再度マップを確認し、既に完了済みの場合は処理をスキップします。
		mc.mu.RLock()
		existingURI, ok := mc.CharacterResourceMap[charID]
		mc.mu.RUnlock()
		if ok {
			return existingURI, nil
		}

		// ネットワーク I/O を伴うアップロード実行
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

	uri, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("unexpected return type from singleflight: %T", val)
	}
	return uri, nil
}
