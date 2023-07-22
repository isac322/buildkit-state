import fs from 'fs/promises'
import os from 'os'
import * as core from '@actions/core'
import * as exec from '@actions/exec'
import * as toolCache from '@actions/tool-cache'

const toolName = 'buildkit_state'

export async function getBinary(
  version: string
): Promise<{toolPath: string; binaryName: string}> {
  const filename = getFilename()
  version = 'fix-tool-cache'
  const cachedPath = toolCache.find(toolName, version)
  core.debug(`cached path: ${cachedPath}`)
  if (cachedPath) {
    core.info('Restore from cache')
    return {toolPath: cachedPath, binaryName: filename}
  }

  core.debug(`filename: ${filename}`)

  core.info(`Downloading ${filename}...`)
  const downPath = await toolCache.downloadTool(
    `https://github.com/isac322/buildkit-state/releases/download/${version}/${filename}`
  )
  await fs.chmod(downPath, 0o755)
  core.debug(`downloaded path: ${downPath}`)
  core.info(`Caching ${filename} for future usage...`)
  const toolPath = await toolCache.cacheFile(
    downPath,
    filename,
    toolName,
    version
  )
  core.debug(`toolPath: ${toolPath}`)
  return {
    toolPath,
    binaryName: filename
  }
}

function getFilename(): string {
  const platform = os.platform()
  const arch = os.arch()

  switch (platform) {
    case 'darwin':
      switch (arch) {
        case 'arm64':
          return `${platform}-arm64`
        case 'x64':
          return `${platform}-amd64`
      }
      break
    case 'linux':
      switch (arch) {
        case 'arm':
          return `${platform}-arm`
        case 'arm64':
          return `${platform}-arm64`
        case 'x64':
          return `${platform}-amd64`
      }
      break
    case 'win32':
      switch (arch) {
        case 'x64':
          return `windows-amd64.exe`
      }
  }
  throw new Error(
    `Unsupported platform (${platform}) and architecture (${arch})`
  )
}

export async function setDockerAPIVersionToEnv(): Promise<void> {
  const dockerServerVersion = await exec.getExecOutput('docker', [
    'version',
    '-f',
    '{{.Server.APIVersion}}'
  ])
  if (dockerServerVersion.exitCode !== 0) {
    throw new Error(
      `Failed to get docker api version: ${dockerServerVersion.stderr}`
    )
  }
  if (process.env.DOCKER_API_VERSION === undefined) {
    process.env.DOCKER_API_VERSION = dockerServerVersion.stdout
  }
}
