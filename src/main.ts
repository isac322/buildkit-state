import * as core from '@actions/core'
import * as exec from '@actions/exec'
import {version} from '../package.json'
import {binaryName, getBinary} from './common'

async function run(): Promise<void> {
  try {
    core.debug(`version: ${version}`)
    await core.group('Download binary', async () => {
      const toolPath = await getBinary(version)
      core.addPath(toolPath)
    })

    await exec.exec(binaryName, ['load'])
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message)
    }
  }
}

run()
