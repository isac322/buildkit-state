import fs from 'fs/promises'
import os from 'os'
import * as core from '@actions/core'
import * as exec from '@actions/exec'
import * as toolCache from '@actions/tool-cache'
import semver from 'semver'

export const STATE_RESTORED_KEY = 'restored-cache-key'
export const ARCHIVE_PATH = '/tmp/buildkit-cache/buildkit-state.tar.zst'
export const BUILDKIT_STATE_PATH = '/var/lib/buildkit'
const zstdVersionLongAdded = new semver.SemVer('v1.3.2')
const TOOL_NAME_ZSTD = 'static-zstd-slim'

async function getVersion(
  app: string,
  additionalArgs: string[] = []
): Promise<string> {
  let versionOutput = ''
  additionalArgs.push('--version')
  core.debug(`Checking ${app} ${additionalArgs.join(' ')}`)
  try {
    await exec.exec(app, additionalArgs, {
      ignoreReturnCode: true,
      silent: true,
      listeners: {
        stdout: data => (versionOutput += data.toString()),
        stderr: data => (versionOutput += data.toString())
      }
    })
  } catch (err) {
    if (err instanceof Error) {
      core.debug(err.message)
    }
  }
  versionOutput = versionOutput.trim()
  core.debug(versionOutput)
  return versionOutput
}

async function getZstdVersion(binPath: string): Promise<string | null> {
  const versionOutput = await getVersion(binPath, ['--quiet'])
  return semver.clean(versionOutput)
}

export async function downloadZstdIfNotExistAndCheckIfSupportsLong(): Promise<boolean> {
  let version = await getZstdVersion('zstd')
  core.debug(`zstd version: ${version}`)
  if (version === null) {
    if (os.platform() !== 'linux') {
      throw new Error(
        'Does not support to install zstd dynamically. Please install it on runner itself.'
      )
    }

    version = await tryInstallZstd()
  }

  return semver.gte(version, zstdVersionLongAdded)
}

async function tryInstallZstd(): Promise<string> {
  return await core.group('Install zstd', async (): Promise<string> => {
    const binaryName = getBinaryName()
    core.debug(`BinaryName: ${binaryName}`)
    if (binaryName === null) {
      throw new Error('Can not find zstd binary for the architecture.')
    }

    core.info(`Downloading ${binaryName}...`)
    const downPath = await toolCache.downloadTool(
      `https://github.com/isac322/static-bin/releases/download/zstd-slim/${binaryName}`
    )
    core.debug(`Downloaded path: ${downPath}`)
    await fs.chmod(downPath, 0o755)

    const zstdVer = await getZstdVersion(downPath)
    if (zstdVer === null) {
      throw new Error('Failed to download zstd')
    }

    core.info(`Caching zstd:${zstdVer} for future usage...`)
    const toolPath = await toolCache.cacheFile(
      downPath,
      'zstd',
      TOOL_NAME_ZSTD,
      zstdVer
    )
    core.addPath(toolPath)
    return zstdVer
  })
}

function getBinaryName(): string | null {
  switch (os.arch()) {
    case 'arm64':
      return 'arm64'
    case 'x64':
      return 'amd64'
    case 'ppc64':
      return 'ppc64el'
    case 's390x':
      return 's390x'
    case 'arm':
      return 'armv7l'
    default:
      return null
  }
}

export function getContainerName(builderName: string): string {
  return `buildx_buildkit_${builderName}0`
}
