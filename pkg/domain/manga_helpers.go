package domain

// UniqueSpeakerIDs はパネルのスライスから重複しない SpeakerID を抽出します。
func (ps Panels) UniqueSpeakerIDs() []string {
	// 重複排除用の集合 (Set) を作成
	set := make(map[string]struct{})

	for _, panel := range ps {
		// 空文字でない場合のみ追加
		if panel.SpeakerID != "" {
			set[panel.SpeakerID] = struct{}{}
		}
	}

	// 抽出されたIDをスライスに変換
	uniqueIDs := make([]string, 0, len(set))
	for id := range set {
		uniqueIDs = append(uniqueIDs, id)
	}

	return uniqueIDs
}
