import * as cache from '@actions/cache'
import * as core from '@actions/core'
import * as exec from '@actions/exec'
import child_process from 'child_process'
import * as common from './common'

const cacheTypes = [
  'internal',
  'frontend',
  'source.local',
  'source.git.checkout',
  'exec.cachemount',
  'regular'
]

async function saveCache(
  cachePath: string,
  containerName: string,
  compressionLevel: number,
  zstdWindowSize: number | null
): Promise<number | null> {
  return new Promise((resolve, reject) => {
    const cpProc = child_process.spawn(
      'docker',
      ['cp', '-a', '-', `${containerName}:${common.BUILDKIT_STATE_PATH}`],
      {stdio: ['ignore', 'pipe', 'inherit']}
    )
    cpProc.on('error', reject)

    const zstdArgs = [
      '-T0',
      `-${compressionLevel}`,
      '-cf',
      cachePath,
      '--',
      '-'
    ]
    if (zstdWindowSize !== null) {
      zstdArgs.splice(1, 0, `--long=${zstdWindowSize}`)
    }
    const zstdProc = child_process.spawn('zstd', zstdArgs, {
      stdio: ['pipe', 'inherit', 'inherit']
    })

    cpProc.stdout.on('data', chunk => zstdProc.stdin.write(chunk))
    cpProc.on('close', code => {
      zstdProc.stdin.end()
      if (code !== 0) {
        cpProc.kill('SIGTERM')
      }
    })
    zstdProc.on('error', reject)
    zstdProc.on('close', resolve)
  })
}

async function run(): Promise<void> {
  const primaryKey = core.getInput('cache-key')
  const restoredCacheKey = core.getState(common.STATE_RESTORED_KEY)
  core.info(`restoredCacheKey: ${restoredCacheKey}, cacheKey: ${primaryKey}`)
  if (
    primaryKey === restoredCacheKey &&
    !core.getBooleanInput('rewrite-cache')
  ) {
    core.info('Cache key matched. Ignore cache saving.')
    return
  }

  const builderName = core.getInput('buildx-name')
  core.debug(`Builder name: ${builderName}`)
  const containerName = common.getContainerName(builderName)
  core.debug(`Container name: ${containerName}`)

  try {
    await core.group('Remove unwanted caches', async () => {
      const toCache = core.getMultilineInput('target-types')
      const exitCodes = await Promise.all(
        cacheTypes
          .filter(value => !toCache.includes(value))
          .map(async t =>
            exec.exec('docker', [
              'buildx',
              'prune',
              '--force',
              '--builder',
              builderName,
              '--filter',
              `type=${t}`
            ])
          )
      )

      if (!exitCodes.every(v => v === 0)) {
        core.setFailed('Failed with non zero return')
        return
      }

      await exec.exec('docker', [
        'buildx',
        'du',
        '--verbose',
        '--builder',
        builderName
      ])
    })

    await core.group('Stop buildkit daemon', async () => {
      await exec.exec('docker', ['buildx', 'stop', builderName])
    })

    let zstdWindowSize: number | null = parseInt(core.getInput('window-size'))
    if (!(await common.downloadZstdIfNotExistAndCheckIfSupportsLong())) {
      zstdWindowSize = null
    }

    await core.group('Save cache from builder', async () => {
      const compressionLevel = parseInt(core.getInput('compression-level'))
      const exitCode = await saveCache(
        common.ARCHIVE_PATH,
        containerName,
        compressionLevel,
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

    await cache.saveCache([common.ARCHIVE_PATH], primaryKey)
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message)
    }
  }
}

run()
