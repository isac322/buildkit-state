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

async function run() {
  try {
    const buildxName = core.getInput('buildx-name');
    const buildxContainerName = core.getInput('buildx-container-name');

    validateInputs({buildxName, buildxContainerName});

    await core.group('Stopping buildx', async () => {
      await exec.exec('docker', ['buildx', 'stop']);
    });

    const cacheExists = await core.group('Fetching Github cache', async () => {
      const cacheRestoreKeys = core.getMultilineInput('cache-restore-key');
      const cacheKey = core.getInput('cache-key');

      core.info(`fetching github cache using key ${cacheKey}...`);
      const restoredCacheKey = await cache.restoreCache(
        [BUILDKIT_STATE_PATH],
        cacheKey,
        cacheRestoreKeys
      );
      if (restoredCacheKey === undefined) {
        core.info('Cache does not exists.');
        return false;
      }
      core.info(`github cache restored. key: ${restoredCacheKey}`);
      core.saveState(STATE_RESTORED_CACHE_KEY, restoredCacheKey);
      return true;
    });
    if (!cacheExists) {
      core.info('Failed to fetch Github cache. Skip buildkit state restoring.');
      return;
    }

    await core.group('Restoring buildkit state', async () => {
      const docker = new Docskerode();
      const container = docker.getContainer(
        getContainerName({buildxName, buildxContainerName})
      );
      core.info(`found container ${container.id}`);

      core.info('restoring buildkit state into buildx container...');
      const stateStream = fs.createReadStream(BUILDKIT_STATE_PATH, {
        encoding: 'binary'
      });
      await container.putArchive(stateStream, {path: '/var/lib/'});
      stateStream.close();
    });
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message);
    } else {
      core.setFailed(error as any);
    }
  } finally {
    core.info('restarting buildx...');
    await exec.exec('docker', ['buildx', 'inspect', '--bootstrap']);
    await exec.exec('docker', ['buildx', 'du', '--verbose']);
  }
}

function validateInputs(opts: Inputs) {
  if (opts.buildxContainerName == '' && opts.buildxName == '') {
    throw new Error('buildx-name or buildx-container-name must be set');
  }
}

run();
