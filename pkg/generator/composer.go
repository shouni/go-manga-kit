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
			resolvedCharID := char.ID

			_, err, _ := mc.uploadGroup.Do(resolvedCharID, func() (interface{}, error) {
				mc.mu.RLock()
				existingURI, ok := mc.CharacterResourceMap[resolvedCharID]
				mc.mu.RUnlock()
				if ok {
					return existingURI, nil
				}

				uploadedURI, uploadErr := mc.AssetManager.UploadFile(egCtx, char.ReferenceURL)
				if uploadErr != nil {
					return nil, uploadErr
				}

				mc.mu.Lock()
				mc.CharacterResourceMap[resolvedCharID] = uploadedURI
				mc.mu.Unlock()

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
