package workflow

import (
	"time"
)

const (
	defaultGeminiTemperature = float32(0.1)
	defaultCacheExpiration   = 5 * time.Minute
	cacheCleanupInterval     = 15 * time.Minute
	defaultTTL               = 5 * time.Minute
)
