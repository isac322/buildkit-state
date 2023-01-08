import * as core from '@actions/core';
import * as cache from '@actions/cache';
import * as exec from '@actions/exec';
import * as fs from 'fs';
import Docskerode from 'dockerode';
import {
  BUILDKIT_STATE_PATH,
  getContainerName,
  Inputs,
  STATE_RESTORED_CACHE_KEY
} from './common';

export async function run() {
  try {
    const buildxName = core.getInput('buildx-name');
    const buildxContainerName = core.getInput('buildx-container-name');

    validateInputs({buildxName, buildxContainerName});

    await exec.exec('docker', ['buildx', 'stop']);

    const cacheRestoreKeys = core.getMultilineInput('cache-restore-key');
    const cacheKey = core.getInput('cache-key');

    const restoredCacheKey = await cache.restoreCache(
      [BUILDKIT_STATE_PATH],
      cacheKey,
      cacheRestoreKeys
    );
    if (restoredCacheKey === undefined) {
      core.info('Failed to fetch cache.');
      return;
    }
    core.saveState(STATE_RESTORED_CACHE_KEY, restoredCacheKey);

    const docker = new Docskerode();
    const container = docker.getContainer(
      getContainerName({buildxName, buildxContainerName})
    );
    const stateStream = fs.createReadStream(BUILDKIT_STATE_PATH, {
      encoding: 'binary'
    });
    await container.putArchive(stateStream, {path: '/var/lib/'});
    stateStream.close();
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message);
    } else {
      core.setFailed(error as any);
    }
  } finally {
    await exec.exec('docker', ['buildx', 'inspect', '--bootstrap']);
    await exec.exec('docker', ['buildx', 'du', '--verbose']);
  }
}

function validateInputs(opts: Inputs) {
  if (opts.buildxContainerName == '' && opts.buildxName == '') {
    throw new Error('buildx-name or buildx-container-name must be set');
  }
}
