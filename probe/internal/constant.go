package internal

const (
	BuildKitStateSaveDir = "/var/lib/buildkit"
	BuildKitStateLoadDir = "/var/lib"

	inputPrimaryKey       = "cache-key"
	inputSecondaryKeys    = "cache-restore-keys"
	inputTargetTypes      = "target-types"
	inputRewriteCache     = "rewrite-cache"
	inputResumeBuilder    = "resume-builder"
	inputCompressionLevel = "compression-level"

	outputRestoredCacheKey = "restored-cache-key"

	stateLoadedCacheKey = "loaded-cache-key"
)
