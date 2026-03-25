package layout

import (
	"sort"

	imagePorts "github.com/shouni/gemini-image-kit/ports"
	"github.com/shouni/go-manga-kit/ports"
)

type pageResourceCollector struct {
	composer    *MangaComposer
	resourceMap *ports.ResourceMap
	addedByURL  map[string]int
}

// newPageResourceCollector は指定されたコンポーザーからページリソースコレクターを初期化します。
func newPageResourceCollector(composer *MangaComposer) *pageResourceCollector {
	return &pageResourceCollector{
		composer: composer,
		resourceMap: &ports.ResourceMap{
			CharacterFiles: make(map[string]int),
			PanelFiles:     make(map[string]int),
		},
		addedByURL: make(map[string]int),
	}
}

// addCharacterAssets は指定されたパネルのキャラクターアセットをリソースマップに追加し、リソースインデックスをキャラクター参照URLに関連付けます。
func (c *pageResourceCollector) addCharacterAssets(panels []ports.Panel) {
	for _, speakerID := range ports.Panels(panels).UniqueSpeakerIDs() {
		char := c.composer.CharactersMap.GetCharacter(speakerID)
		if char == nil || char.ReferenceURL == "" {
			continue
		}

		fileURI := c.composer.GetCharacterResourceURI(char.ID)
		if fileURI == "" {
			continue
		}

		idx := c.addAsset(imagePorts.ImageURI{
			ReferenceURL: char.ReferenceURL,
			FileAPIURI:   fileURI,
		})
		c.resourceMap.CharacterFiles[speakerID] = idx
	}
}

// addPanelAssets はソートされたパネルアセットをリソースマップに追加し、リソースインデックスをパネル参照URLに関連付けます。
func (c *pageResourceCollector) addPanelAssets(panels []ports.Panel) {
	panelAssets := c.sortedPanelAssets(panels)
	for _, asset := range panelAssets {
		idx := c.addAsset(asset)
		c.resourceMap.PanelFiles[asset.ReferenceURL] = idx
	}

	for _, panel := range panels {
		if panel.ReferenceURL == "" {
			continue
		}
		if idx, ok := c.addedByURL[panel.ReferenceURL]; ok {
			c.resourceMap.PanelFiles[panel.ReferenceURL] = idx
		}
	}
}

// sortedPanelAssets は指定されたパネルのリソースをソートして返します。
func (c *pageResourceCollector) sortedPanelAssets(panels []ports.Panel) []imagePorts.ImageURI {
	var panelAssets []imagePorts.ImageURI

	for _, panel := range panels {
		if panel.ReferenceURL == "" {
			continue
		}
		if _, exists := c.addedByURL[panel.ReferenceURL]; exists {
			continue
		}

		fileURI := c.composer.GetPanelResourceURI(panel.ReferenceURL)
		if fileURI == "" {
			continue
		}

		panelAssets = append(panelAssets, imagePorts.ImageURI{
			ReferenceURL: panel.ReferenceURL,
			FileAPIURI:   fileURI,
		})
		c.addedByURL[panel.ReferenceURL] = -1
	}

	sort.Slice(panelAssets, func(i, j int) bool {
		return panelAssets[i].ReferenceURL < panelAssets[j].ReferenceURL
	})

	return panelAssets
}

// addAsset は指定されたアセットを追加し、そのインデックスを返します。
func (c *pageResourceCollector) addAsset(asset imagePorts.ImageURI) int {
	if idx, exists := c.addedByURL[asset.ReferenceURL]; exists && idx >= 0 {
		return idx
	}

	idx := len(c.resourceMap.OrderedAssets)
	c.resourceMap.OrderedAssets = append(c.resourceMap.OrderedAssets, asset)
	c.addedByURL[asset.ReferenceURL] = idx
	return idx
}
