package internal

const (
	BuildKitStateDir = "/var/lib/buildkit"

	inputPrimaryKey       = "cache-key"
	inputSecondaryKeys    = "cache-restore-keys"
	inputTargetTypes      = "target-types"
	inputRewriteCache     = "rewrite-cache"
	inputResumeBuilder    = "resume-builder"
	inputCompressionLevel = "compression-level"

	outputRestoredCacheKey = "restored-cache-key"

	stateLoadedCacheKey = "loaded-cache-key"
)
