import * as cache from '@actions/cache'
import * as core from '@actions/core'
import * as exec from '@actions/exec'
import child_process from 'child_process'
import path from 'path'
import * as common from './common'

async function loadCache(
  cachePath: string,
  containerName: string,
  zstdWindowSize: number | null
): Promise<number | null> {
  let save_on_failure = core.getInput('save-on-failure').toLowerCase()
  if (save_on_failure !== 'true') {
    save_on_failure = 'false'
  }
  core.exportVariable('SAVE_ON_FAILURE', save_on_failure)

  return new Promise((resolve, reject) => {
    const zstdArgs = ['-T0', '-d', '--stdout', '--force', '--', cachePath]
    if (zstdWindowSize !== null) {
      zstdArgs.splice(1, 0, `--long=${zstdWindowSize}`)
    }
    const zstdProc = child_process.spawn('zstd', zstdArgs, {
      stdio: ['ignore', 'pipe', 'inherit']
    })
    zstdProc.on('error', reject)

    const cpProc = child_process.spawn(
      'docker',
      [
        'cp',
        '-',
        `${containerName}:${path.dirname(common.BUILDKIT_STATE_PATH)}`
      ],
      {stdio: ['pipe', 'inherit', 'inherit']}
    )

    zstdProc.stdout.on('data', chunk => cpProc.stdin.write(chunk))
    zstdProc.on('close', code => {
      cpProc.stdin.end()
      if (code !== 0) {
        cpProc.kill('SIGTERM')
      }
    })
    cpProc.on('error', reject)
    cpProc.on('close', resolve)
  })
}

async function run(): Promise<void> {
  try {
    const restoredKey = await core.group('Download cache', async () => {
      const primaryKey = core.getInput('cache-key')
      const secondaryKeys = core.getMultilineInput('cache-restore-keys')

      return await cache.restoreCache(
        [common.ARCHIVE_PATH],
        primaryKey,
        secondaryKeys
      )
    })
    if (restoredKey === undefined) {
      core.info('Cache not found. Skip loading.')
      return
    }
    core.saveState(common.STATE_RESTORED_KEY, restoredKey)
    core.setOutput('restored-cache-key', restoredKey)
  } catch (e) {
    if (e instanceof Error) {
      core.setFailed(e.message)
    }
  }

  const builderName = core.getInput('buildx-name')
  core.debug(`Builder name: ${builderName}`)
  const containerName = common.getContainerName(builderName)
  core.debug(`Container name: ${containerName}`)

  try {
    let zstdWindowSize: number | null = parseInt(core.getInput('window-size'))
    if (!(await common.downloadZstdIfNotExistAndCheckIfSupportsLong())) {
      zstdWindowSize = null
    }

    await core.group('Stop buildkit daemon', async () => {
      await exec.exec('docker', ['buildx', 'stop', builderName])
    })

    await core.group('Load cache into builder', async () => {
      const exitCode = await loadCache(
        common.ARCHIVE_PATH,
        containerName,
        zstdWindowSize
      )
      if (exitCode !== 0) {
        core.setFailed(`Failed with non zero return: ${exitCode}`)
        return
      }
    })

    if (!core.getBooleanInput('resume-builder')) {
      core.debug('Skip buildx resuming')
      return
    }

    await exec.exec('docker', ['buildx', 'inspect', '--bootstrap', builderName])
    await exec.exec('docker', [
      'buildx',
      'du',
      '--verbose',
      '--builder',
      builderName
    ])
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message)
    }
  }
}

run()
