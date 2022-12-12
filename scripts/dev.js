const { join, resolve } = require('path');
const chokidar = require('chokidar');
const { spawn, exec } = require('child_process');
const colorette = require('colorette');
const opener = require('opener');

let cpServer = null;
let cpWebapp = null;

const baseDir = resolve(__dirname, '..');
const captureDir = join(baseDir, 'apps', 'client');
const serverDir = join(baseDir, 'apps', 'server');

const serveWebapp = () => {
  cpWebapp = exec(
    `webpack serve -c ${baseDir}/apps/webapp/webpack.config.dev.js`,
    {
      stdio: 'inherit',
    },
  );
  cpWebapp.stdout.pipe(process.stdout);
  cpWebapp.stderr.pipe(process.stderr);
  opener('http://localhost:3000');
};

const spawnCapture = () => {
  console.log(colorette.blueBright('Building stream server...'));
  const runtimeEnv = {
    CGO_ENABLED: 1,
    CGO_CFLAGS: `-I${captureDir}\\gstreamer\\include`,
    CGO_LDFLAGS: `-L${captureDir}\\gstreamer\\lib -L${captureDir}\\dll\\x64`,
    PKG_CONFIG_PATH: `${captureDir}\\gstreamer\\lib\\pkgconfig`,
    GST_PLUGIN_PATH_1_0: `${captureDir}\\gstreamer\\dll\\plugins`,
    PATH: `${captureDir}\\gstreamer\\dll\\dll;${process.env.PATH}`,
    CONFIG_PATH: `${captureDir}\\config.json`,
  };

  const builder = exec(
    `cd ${captureDir} && wails dev`,

    {
      stdio: 'inherit',
      env: { ...process.env, ...runtimeEnv },
    },
  );

  builder.stdout.pipe(process.stdout);
  builder.stderr.pipe(process.stderr);
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

chokidar.watch(join(serverDir, '**', '*.go')).on('change', spawnServer);

spawnCapture();
spawnServer();
serveWebapp();
