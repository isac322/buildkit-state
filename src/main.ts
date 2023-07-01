import * as cache from '@actions/cache'
import * as core from '@actions/core'
import * as exec from '@actions/exec'
import * as io from '@actions/io'
import fsPromise from 'fs/promises'
import Docskerode from 'dockerode'
import {
  BUILDKIT_STATE_PATH,
  STATE_BUILDKIT_STATE_PATH_KEY,
  STATE_RESTORED_CACHE_KEY
} from './common'
import path from 'path'

async function run(): Promise<void> {
  const buildxName = core.getInput('buildx-name')
  const containerName = `buildx_buildkit_${buildxName}0`
  core.debug(`container name: ${containerName}`)

  try {
    await core.group('Stopping buildx', async () => {
      await exec.exec('docker', [
        'buildx',
        'inspect',
        buildxName,
        '--bootstrap'
      ])
      await exec.exec('docker', ['buildx', 'stop', buildxName])
    })

    await core.group('Locate buildkit state', async () => {
      const docker = new Docskerode()
      const container = docker.getContainer(containerName)
      core.info(`found container ${container.id}`)

      const volumeName = `${containerName}_state`
      const containerInfo = await container.inspect()
      core.debug(JSON.stringify(containerInfo))

      core.debug(`looking for volume name: ${volumeName}`)
      const stateMount = containerInfo.Mounts.find(m => m.Name === volumeName)
      if (stateMount === undefined) {
        throw new Error(`failed to find volume: ${volumeName}`)
      }
      core.info(`Found location of buildkit state: ${stateMount.Source}`)
      core.saveState(STATE_BUILDKIT_STATE_PATH_KEY, stateMount.Source)
      await io.rmRF(stateMount.Source)
    })

    await core.group('Fetching Github cache', async () => {
      const cacheRestoreKeys = core.getMultilineInput('cache-restore-keys')
      const cacheKey = core.getInput('cache-key')

      core.info(`fetching github cache using key ${cacheKey}...`)
      const restoredCacheKey = await cache.restoreCache(
        [BUILDKIT_STATE_PATH],
        cacheKey,
        cacheRestoreKeys
      )
      if (restoredCacheKey === undefined) {
        core.info(
          'Failed to fetch Github cache. Skip buildkit state restoring.'
        )
        return
      }
      core.info(`github cache restored. key: ${restoredCacheKey}`)
      core.saveState(STATE_RESTORED_CACHE_KEY, restoredCacheKey)

      const statePath = core.getState(STATE_BUILDKIT_STATE_PATH_KEY)
      await io.mv(BUILDKIT_STATE_PATH, statePath, {force: true})
    })
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message)
    }
  } finally {
    core.info('restarting buildx...')
    await exec.exec('docker', ['buildx', 'inspect', buildxName, '--bootstrap'])
    await exec.exec('docker', ['buildx', 'du', '--verbose'])
  }
}

run()
