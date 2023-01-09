import * as core from '@actions/core';
import * as exec from '@actions/exec';
import * as cache from '@actions/cache';
import Docskerode from 'dockerode';
import fs from 'fs';
import {
  BUILDKIT_STATE_PATH,
  getContainerName,
  STATE_RESTORED_CACHE_KEY,
  STATE_TYPES
} from './common';

async function run() {
  try {
    core.debug('post action started');

    const cacheKey = core.getInput('cache-key');
    const restoredCacheKey = core.getState(STATE_RESTORED_CACHE_KEY);
    if (restoredCacheKey == cacheKey) {
      core.info('Cache key matched. Ignore cache saving.');
      return;
    }

    core.info('removing unwanted caches...');
    const targetTypes = core.getMultilineInput('target-types');
    await Promise.all(
      STATE_TYPES.filter(type => targetTypes.indexOf(type) == -1).map(type =>
        exec.exec('docker', [
          'buildx',
          'prune',
          '--force',
          '--filter',
          `type=${type}`
        ])
      )
    );

    await exec.getExecOutput('docker', ['buildx', 'du', '--verbose']);

    core.info('stopping buildx...');
    await exec.exec('docker', ['buildx', 'stop']);

    const buildxName = core.getInput('buildx-name');
    const buildxContainerName = core.getInput('buildx-container-name');

    const docker = new Docskerode();
    const container = docker.getContainer(
      getContainerName({buildxName, buildxContainerName})
    );
    core.debug(`found container ${container.id}`);

    core.info('fetching buildkit state from buildx container...');
    const outputStream = fs.createWriteStream(BUILDKIT_STATE_PATH, {
      encoding: 'binary'
    });
    const inputStream = await container.getArchive({path: '/var/lib/buildkit'});
    const promiseExecute = async () => {
      return new Promise(resolve => {
        inputStream.pipe(outputStream);
        outputStream.on('finish', resolve);
      });
    };
    await promiseExecute();
    outputStream.close();

    core.info('saving buildkit state into Github cache...');
    await cache.saveCache([BUILDKIT_STATE_PATH], cacheKey);
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message);
    } else {
      core.setFailed(error as any);
    }
  }
}

run();
