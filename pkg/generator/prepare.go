package generator

import (
	"context"
	"fmt"

	"github.com/shouni/go-manga-kit/pkg/domain"
	"golang.org/x/sync/errgroup"
)

// prepareCharacterResources はパネルに使用される全キャラクターの画像を File API に事前アップロードします。
func prepareCharacterResources(ctx context.Context, composer *MangaComposer, panels []domain.Panel) error {
	uniqueSpeakerIDs := domain.Panels(panels).UniqueSpeakerIDs()
	cm := composer.CharactersMap
	eg, egCtx := errgroup.WithContext(ctx)

	for _, id := range uniqueSpeakerIDs {
		speakerID := id

		eg.Go(func() error {
			char := cm.GetCharacterWithDefault(speakerID)
			if char == nil || char.ReferenceURL == "" {
				return nil
			}
			resolvedCharID := char.ID

			// singleflight を使い、同じ resolvedCharID に対する処理を集約
			_, err, _ := composer.uploadGroup.Do(resolvedCharID, func() (interface{}, error) {
				// singleflight 呼び出し前に既にマップに存在するか最終チェック
				composer.mu.RLock()
				existingURI, ok := composer.CharacterResourceMap[resolvedCharID]
				composer.mu.RUnlock()
				if ok {
					return existingURI, nil
				}

				// 重いアップロード処理（ここが同時に呼ばれるのは singleflight により resolvedCharID ごとに1回のみ）
				uploadedURI, uploadErr := composer.AssetManager.UploadFile(egCtx, char.ReferenceURL)
				if uploadErr != nil {
					return nil, uploadErr
				}

				// 書き込みのみロック
				composer.mu.Lock()
				composer.CharacterResourceMap[resolvedCharID] = uploadedURI
				composer.mu.Unlock()

				return uploadedURI, nil
			})

			if err != nil {
				return fmt.Errorf("failed to prepare asset for character %s (resolved from speaker %s): %w", resolvedCharID, speakerID, err)
			}

			return nil
		})
	}

	return eg.Wait()
}
