package director

// LayoutManager は吹き出しの座標や配置ルールを管理します。
type LayoutManager struct {
	DefaultMargin string
}

func NewLayoutManager() *LayoutManager {
	return &LayoutManager{
		DefaultMargin: "10%",
	}
}

// GetPositionAttrs はパネルのインデックスに基づき、
// 右から左へ流れるような交互の配置属性を返します。
func (l *LayoutManager) GetPositionAttrs(index int) map[string]string {
	attrs := make(map[string]string)

	// 記憶した「偶数・奇数での対角配置」ロジック
	if index%2 == 0 {
		attrs["tail"] = "top"
		attrs["bottom"] = l.DefaultMargin
		attrs["left"] = l.DefaultMargin
	} else {
		attrs["tail"] = "bottom"
		attrs["top"] = l.DefaultMargin
		attrs["right"] = l.DefaultMargin
	}

	return attrs
}
