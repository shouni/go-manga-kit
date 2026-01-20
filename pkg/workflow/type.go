package workflow

import (
	"time"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/generator"

	"github.com/shouni/go-gemini-client/pkg/gemini"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

const (
	defaultCacheExpiration = 5 * time.Minute
	cacheCleanupInterval   = 15 * time.Minute
	defaultTTL             = 5 * time.Minute
)

// Manager は、ワークフローの各工程を担う Runner 群を構築・管理します。
type Manager struct {
	cfg           config.Config
	httpClient    httpkit.ClientInterface
	aiClient      gemini.GenerativeModel
	reader        remoteio.InputReader
	writer        remoteio.OutputWriter
	mangaComposer *generator.MangaComposer
}
