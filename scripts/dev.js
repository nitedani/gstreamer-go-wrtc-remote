const { join, resolve } = require('path');
const chokidar = require('chokidar');
const { spawn, exec } = require('child_process');
const colorette = require('colorette');

let cpStreamServer = null;
let cpSignalingServer = null;
const baseDir = resolve(__dirname, '..');
const streamServerDir = join(baseDir, 'apps', 'streamserver');
const signalingServerDir = join(baseDir, 'apps', 'signalserver');

const spawnStreamServer = () => {
  console.log(colorette.blueBright('Building stream server...'));
  const buildEnv = {
    CGO_ENABLED: 1,
    CGO_CFLAGS: `-I${streamServerDir}\\gstreamer\\include`,
    CGO_LDFLAGS: `-L${streamServerDir}\\gstreamer\\lib`,
    PKG_CONFIG_PATH: `${streamServerDir}\\gstreamer\\lib\\pkgconfig`,
  };

  const runtimeEnv = {
    GST_PLUGIN_PATH_1_0: `${streamServerDir}\\gstreamer\\dll\\plugins`,
    PATH: `${streamServerDir}\\gstreamer\\dll\\dll;${process.env.PATH}`,
  };

  if (cpStreamServer) {
    cpStreamServer.kill('SIGTERM');
    cpStreamServer = null;
  }

  const binPath = join(baseDir, 'dist', 'stream.exe');
  const builder = exec(
    `cd ${streamServerDir}\\main && go build -v -o ${binPath}`,
    {
      stdio: 'inherit',
      env: { ...process.env, ...buildEnv },
    },
  );

  builder.stdout.pipe(process.stdout);
  builder.stderr.pipe(process.stderr);

  builder
    .on('close', (code) => {
      if (code !== 0) {
        return;
      }
      console.log(colorette.blueBright('Starting stream server...'));
      cpStreamServer = spawn(binPath, [`${streamServerDir}\\.env`], {
        stdio: 'inherit',
        env: { ...process.env, ...runtimeEnv },
      });
    })
    .on('error', (err) => {});
};

const spawnSignalingServer = () => {
  console.log(colorette.blueBright('Building signaling server...'));
  const buildEnv = {};

  const runtimeEnv = {};

  if (cpSignalingServer) {
    cpSignalingServer.kill('SIGTERM');
    cpSignalingServer = null;
  }

  const binPath = join(baseDir, 'dist', 'signaling.exe');
  const builder = exec(
    `cd ${signalingServerDir}\\main && go build -v -o ${binPath}`,
    {
      stdio: 'inherit',
      env: { ...process.env, ...buildEnv },
    },
  );

  builder.stdout.pipe(process.stdout);
  builder.stderr.pipe(process.stderr);

  builder
    .on('close', (code) => {
      if (code !== 0) {
        return;
      }
      console.log(colorette.blueBright('Starting signaling server...'));
      cpSignalingServer = spawn(binPath, [`${signalingServerDir}\\.env`], {
        stdio: 'inherit',
        env: { ...process.env, ...runtimeEnv },
      });
    })
    .on('error', (err) => {});
};

chokidar
  .watch(join(streamServerDir, '**', '*.go'))
  .on('change', spawnStreamServer);

chokidar
  .watch(join(signalingServerDir, '**', '*.go'))
  .on('change', spawnSignalingServer);

spawnStreamServer();
spawnSignalingServer();
