export const BUILDKIT_STATE_PATH = '/tmp/buildkit-cache'
export const STATE_RESTORED_CACHE_KEY = 'restored-cache-key'
export const STATE_BUILDKIT_STATE_PATH_KEY = 'buildkit-state-path-key'
export const STATE_TYPES = [
  'regular',
  'source.local',
  'source.git.checkout',
  'exec.cachemount',
  'frontend',
  'internal'
]
