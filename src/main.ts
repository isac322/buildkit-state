import * as core from '@actions/core'
import {getBinary, setDockerAPIVersionToEnv, spawn} from './common'
import {version} from '../package.json'

async function run(): Promise<void> {
  try {
    core.debug(`version: ${version}`)
    const {toolPath, binaryName} = await getBinary(version)
    core.addPath(toolPath)

    await setDockerAPIVersionToEnv()

    const code = await spawn(binaryName, ['load'], {stdio: 'inherit'})
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
