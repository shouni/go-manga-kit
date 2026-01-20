package workflow

import (
	"time"

	"github.com/shouni/go-manga-kit/pkg/config"
	"github.com/shouni/go-manga-kit/pkg/domain"
	"github.com/shouni/go-manga-kit/pkg/prompts"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

const (
	defaultGeminiTemperature = float32(0.1)
	defaultCacheExpiration   = 5 * time.Minute
	cacheCleanupInterval     = 15 * time.Minute
	defaultTTL               = 5 * time.Minute
)

type ManagerArgs struct {
	Config        config.Config
	HTTPClient    httpkit.ClientInterface
	IOFactory     remoteio.IOFactory
	CharactersMap domain.CharactersMap
	ScriptPrompt  prompts.ScriptPrompt
	ImagePrompt   prompts.ImagePrompt
}
