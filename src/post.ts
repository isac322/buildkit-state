import * as core from '@actions/core'
import * as exec from '@actions/exec'
import {version} from '../package.json'
import {getBinary} from './common'

async function run(): Promise<void> {
  try {
    core.debug(`version: ${version}`)
    const {toolPath, binaryName} = await getBinary(version)
    core.addPath(toolPath)
    await exec.exec(binaryName, ['save'])
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message)
    }
  }
}

run()
