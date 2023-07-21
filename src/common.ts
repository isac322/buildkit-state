import os from 'os'
import * as core from '@actions/core'
import * as toolCache from '@actions/tool-cache'

const binaryPrefix = 'buildkit-state'
export const binaryName = 'buildkit_state'

export async function getBinary(version: string): Promise<string> {
  const cachedPath = toolCache.find(binaryName, version)
  if (cachedPath) {
    core.info('Restore from cache')
    return cachedPath
  }

  const filename = getFilename()
  core.debug(`filename: ${filename}`)

  core.info(`Downloading ${filename}...`)
  const downPath = await toolCache.downloadTool(
    `https://github.com/isac322/buildkit-state/releases/download/v${version}/${filename}`
  )
  core.info(`Caching ${filename} for future usage...`)
  return await toolCache.cacheFile(downPath, filename, binaryName, version)
}

function getFilename(): string {
  const platform = os.platform()
  const arch = os.arch()

  switch (platform) {
    case 'darwin':
      switch (arch) {
        case 'arm64':
          return `${binaryPrefix}-${platform}-arm64`
        case 'x64':
          return `${binaryPrefix}-${platform}-amd64`
      }
      break
    case 'linux':
      switch (arch) {
        case 'arm':
          return `${binaryPrefix}-${platform}-arm-5`
        case 'arm64':
          return `${binaryPrefix}-${platform}-arm64`
        case 'x64':
          return `${binaryPrefix}-${platform}-amd64`
      }
      break
    case 'win32':
      switch (arch) {
        case 'x64':
          return `${binaryPrefix}-windows-amd64.exe`
      }
  }
  throw new Error(
    `Unsupported platform (${platform}) and architecture (${arch})`
  )
}
