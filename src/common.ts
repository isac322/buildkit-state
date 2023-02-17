export const BUILDKIT_STATE_PATH = '/tmp/buildkit-cache/buildkit-state.tar'
export const STATE_RESTORED_CACHE_KEY = 'restored-cache-key'
export const STATE_TYPES = [
  'regular',
  'source.local',
  'exec.cachemount',
  'frontend',
  'internal'
]

export interface Inputs {
  buildxName: string
  buildxContainerName: string
}

export function getContainerName(opts: Inputs): string {
  if (opts.buildxContainerName !== '') {
    return opts.buildxContainerName
  } else {
    return `buildx_buildkit_${opts.buildxName}0`
  }
}
