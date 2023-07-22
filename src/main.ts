import * as core from '@actions/core'
import {version} from '../package.json'
import {getBinary} from './common'
import child_process from 'child_process'
import util from 'util'

async function run(): Promise<void> {
  try {
    core.debug(`version: ${version}`)
    const {toolPath, binaryName} = await getBinary(version)
    core.addPath(toolPath)
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
