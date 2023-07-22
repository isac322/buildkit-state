import os from 'os'
import * as core from '@actions/core'
import * as toolCache from '@actions/tool-cache'
import path from 'path'
import {exec} from '@actions/exec'

const binaryPrefix = 'buildkit-state'
const toolName = 'buildkit_state'

export async function getBinary(
  version: string
): Promise<{toolPath: string; binaryName: string}> {
  const filename = getFilename()
  const cachedPath = toolCache.find(toolName, version)
  core.debug(`cached path: ${cachedPath}`)
  if (cachedPath) {
    core.info('Restore from cache')
    return {toolPath: cachedPath, binaryName: filename}
  }

  core.debug(`filename: ${filename}`)

  core.info(`Downloading ${filename}...`)
  const downPath = await toolCache.downloadTool(
    `https://github.com/isac322/buildkit-state/releases/download/v${version}/${filename}`
  )
  await exec('ls', ['-ahl', downPath])
  await exec('file', [downPath])
  core.debug(`downloaded path: ${downPath}`)
  core.info(`Caching ${filename} for future usage...`)
  return {
    toolPath: await toolCache.cacheFile(
      path.dirname(downPath),
      filename,
      toolName,
      version
    ),
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
