const { join, resolve } = require('path');
const chokidar = require('chokidar');
const { spawn, exec } = require('child_process');
const colorette = require('colorette');
const { copySync } = require('fs-extra');

let cpCapture = null;
let cpServer = null;
const baseDir = resolve(__dirname, '..');
const captureDir = join(baseDir, 'apps', 'capture');
const serverDir = join(baseDir, 'apps', 'server');

const spawnCapture = () => {
  console.log(colorette.blueBright('Building stream server...'));
  const buildEnv = {
    CGO_ENABLED: 1,
    CGO_CFLAGS: `-I${captureDir}\\gstreamer\\include`,
    CGO_LDFLAGS: `-L${captureDir}\\gstreamer\\lib -L${captureDir}\\dll\\x64`,
    PKG_CONFIG_PATH: `${captureDir}\\gstreamer\\lib\\pkgconfig`,
  };

  const runtimeEnv = {
    GST_PLUGIN_PATH_1_0: `${captureDir}\\gstreamer\\dll\\plugins`,
    PATH: `${captureDir}\\gstreamer\\dll\\dll;${process.env.PATH}`,
  };

  if (cpCapture) {
    cpCapture.kill('SIGTERM');
    cpCapture = null;
  }

  const webviewDlls = join(captureDir, 'dll', 'x64');
  const distPath = join(baseDir, 'dist');

  try {
    // copy webview dlls to dist
    copySync(webviewDlls, distPath);
  } catch (error) {}

  const binPath = join(distPath, 'stream.exe');
  const builder = exec(
    `cd ${captureDir}\\main && go build -ldflags="-H windowsgui" -v -o ${binPath}`,
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
      console.log(colorette.blueBright('Starting capture...'));
      cpCapture = spawn(binPath, [`${captureDir}\\config.json`], {
        stdio: 'inherit',
        env: { ...process.env, ...runtimeEnv },
      });
    })
    .on('error', (err) => {});
};

const spawnServer = () => {
  console.log(colorette.blueBright('Building server...'));
  const buildEnv = {};

  const runtimeEnv = {};

  if (cpServer) {
    cpServer.kill('SIGTERM');
    cpServer = null;
  }

  const binPath = join(baseDir, 'dist', 'server.exe');
  const builder = exec(`cd ${serverDir}\\main && go build -v -o ${binPath}`, {
    stdio: 'inherit',
    env: { ...process.env, ...buildEnv },
  });

  builder.stdout.pipe(process.stdout);
  builder.stderr.pipe(process.stderr);

  builder
    .on('close', (code) => {
      if (code !== 0) {
        return;
      }
      console.log(colorette.blueBright('Starting server...'));
      cpServer = spawn(binPath, [`${serverDir}\\.env`], {
        stdio: 'inherit',
        env: { ...process.env, ...runtimeEnv },
      });
    })
    .on('error', (err) => {});
};

chokidar.watch(join(captureDir, '**', '*.go')).on('change', spawnCapture);

chokidar.watch(join(serverDir, '**', '*.go')).on('change', spawnServer);

spawnCapture();
spawnServer();
