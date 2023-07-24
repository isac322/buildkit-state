import * as core from '@actions/core'
import {getBinary, getDockerEndpoint, spawn} from './common'
import {version} from '../package.json'

async function run(): Promise<void> {
  try {
    core.debug(`version: ${version}`)
    const {toolPath, binaryName} = await getBinary(version)
    core.addPath(toolPath)

    const context = core.getInput('docker-context')
    const dockerEndpoint = await getDockerEndpoint(context)

    const args = ['load']
    if (dockerEndpoint !== '') {
      args.push('--docker-endpoint', dockerEndpoint)
    }
    const code = await spawn(binaryName, args, {stdio: 'inherit'})
    if (code !== null && code !== 0) {
      core.setFailed(`non zero return: ${code}`)
    }
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message)
    }
  }
}

run()
