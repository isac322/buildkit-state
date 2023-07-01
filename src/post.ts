import * as cache from '@actions/cache'
import * as core from '@actions/core'
import * as exec from '@actions/exec'
import {
  BUILDKIT_STATE_PATH,
  STATE_BUILDKIT_STATE_PATH_KEY,
  STATE_RESTORED_CACHE_KEY,
  STATE_TYPES
} from './common'
import * as io from '@actions/io'

async function run(): Promise<void> {
  try {
    const rewriteCache = core.getBooleanInput('rewrite-cache')
    const cacheKey = core.getInput('cache-key')
    const restoredCacheKey = core.getState(STATE_RESTORED_CACHE_KEY)
    if (!rewriteCache && restoredCacheKey === cacheKey) {
      core.info('Cache key matched. Ignore cache saving.')
      return
    }

    await core.group('Removing unwanted caches', async () => {
      const targetTypes = core.getMultilineInput('target-types')
      await Promise.all(
        STATE_TYPES.filter(type => !targetTypes.includes(type)).map(
          async type =>
            exec.exec('docker', [
              'buildx',
              'prune',
              '--force',
              '--filter',
              `type=${type}`
            ])
        )
      )
    })

    await core.group('Buildx dist usage', async () => {
      await exec.getExecOutput('docker', ['buildx', 'du', '--verbose'])
    })

    await core.group('Stopping buildx', async () => {
      const buildxName = core.getInput('buildx-name')
      await exec.exec('docker', ['buildx', 'stop', buildxName])
    })

    await core.group('Upload into Github cache', async () => {
      const statePath = core.getState(STATE_BUILDKIT_STATE_PATH_KEY)
      if (core.isDebug()) {
        core.debug('content of buildkit state')
        await exec.exec('ls', ['-ahl', statePath])
      }
      await io.rmRF(BUILDKIT_STATE_PATH)
      await io.mkdirP(BUILDKIT_STATE_PATH)
      await io.mv(statePath, BUILDKIT_STATE_PATH)
      await cache.saveCache([BUILDKIT_STATE_PATH], cacheKey)
    })
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message)
    }
  }
}

run()
