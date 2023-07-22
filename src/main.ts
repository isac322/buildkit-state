import child_process from 'child_process'
import util from 'util'
import * as core from '@actions/core'
import {getBinary, setDockerAPIVersionToEnv} from './common'
import {version} from '../package.json'

async function run(): Promise<void> {
  try {
    core.debug(`version: ${version}`)
    const {toolPath, binaryName} = await getBinary(version)
    core.addPath(toolPath)

    await setDockerAPIVersionToEnv()

    await util.promisify(child_process.spawn)(binaryName, ['load'], {
      stdio: 'inherit'
    })
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message)
    }
  }
}

run()
