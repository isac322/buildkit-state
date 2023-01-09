import * as core from '@actions/core';
import * as exec from '@actions/exec';
import * as cache from '@actions/cache';
import * as io from '@actions/io';
import Docskerode from 'dockerode';
import fs from 'fs';
import {
  BUILDKIT_STATE_PATH,
  getContainerName,
  STATE_RESTORED_CACHE_KEY,
  STATE_TYPES
} from './common';
import path from 'path';

async function run() {
  try {
    const cacheKey = core.getInput('cache-key');
    const restoredCacheKey = core.getState(STATE_RESTORED_CACHE_KEY);
    if (restoredCacheKey == cacheKey) {
      core.info('Cache key matched. Ignore cache saving.');
      return;
    }

    await core.group('Removing unwanted caches', async () => {
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
    });

    await core.group('Buildx dist usage', async () => {
      await exec.getExecOutput('docker', ['buildx', 'du', '--verbose']);
    });

    await core.group('Stopping buildx', async () => {
      await exec.exec('docker', ['buildx', 'stop']);
    });

    await core.group('Fetching buildx state', async () => {
      const buildxName = core.getInput('buildx-name');
      const buildxContainerName = core.getInput('buildx-container-name');

      const docker = new Docskerode();
      const container = docker.getContainer(
        getContainerName({buildxName, buildxContainerName})
      );
      core.info(`found container ${container.id}`);

      core.info('Archiving buildkit state into tar file...');
      await io.mkdirP(path.dirname(BUILDKIT_STATE_PATH));
      const outputStream = fs.createWriteStream(BUILDKIT_STATE_PATH, {
        encoding: 'binary'
      });
      const inputStream = await container.getArchive({
        path: '/var/lib/buildkit'
      });
      const promiseExecute = async () => {
        return new Promise(resolve => {
          inputStream.pipe(outputStream);
          outputStream.on('finish', resolve);
        });
      };
      await promiseExecute();
      outputStream.close();
    });

    await core.group('Upload into Github cache', async () => {
      await cache.saveCache([BUILDKIT_STATE_PATH], cacheKey);
    });
  } catch (error) {
    if (error instanceof Error) {
      core.setFailed(error.message);
    } else {
      core.setFailed(error as any);
    }
  }
}

run();
