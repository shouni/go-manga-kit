package workflow

import (
	"time"
)

const (
	// defaultRateBurst は、短時間に許容される最大リクエスト数（バースト）です。
	// パネルの並列生成時などに、API のレート制限（429 Too Many Requests）に抵触しないよう制御します。
	defaultRateBurst = 1

	// defaultGeminiTemperature はモデル生成時の多様性を制御します。
	// 0.1 は低い値に設定することで、漫画の構成や指示への忠実度を安定させます。
	defaultGeminiTemperature = float32(0.1)

	// defaultInitialDelay リトライのデフォルトの遅延期間を指定します。
	defaultInitialDelay = 60 * time.Second

	// defaultCacheExpiration は、インメモリに保持されたアセット情報の有効期限です。
	// フェッチ済みの画像バイナリや、既に Gemini File API にアップロード済みの
	// URI 情報を再利用し、重複した I/O やアップロード処理を抑制します。
	defaultCacheExpiration = 5 * time.Minute

	// cacheCleanupInterval は、メモリ上の期限切れキャッシュを破棄するバックグラウンド処理の実行間隔です。
	cacheCleanupInterval = 15 * time.Minute

	// defaultTTL は、リモートリソース（GCSや外部URL）の有効期間です。
	// 短期間に同じ ReferenceURL が要求された際、ソースへの再アクセスを防ぐために使用されます。
	defaultTTL = 5 * time.Minute
)
