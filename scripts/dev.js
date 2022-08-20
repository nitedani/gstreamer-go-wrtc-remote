const { join, resolve } = require('path');
const chokidar = require('chokidar');
const { spawn, exec, execSync } = require('child_process');
const colorette = require('colorette');
const { copySync } = require('fs-extra');
const opener = require('opener');
let cpCapture = null;
let cpServer = null;
let cpWebapp = null;
let cpClientGui = null;
const baseDir = resolve(__dirname, '..');
const captureDir = join(baseDir, 'apps', 'client');
const serverDir = join(baseDir, 'apps', 'server');

const serveClientGui = () => {
  cpClientGui = exec(`cd ${baseDir}/apps/client/frontend && npm run dev`, {
    stdio: 'inherit',
  });
  cpClientGui.stdout.pipe(process.stdout);
  cpClientGui.stderr.pipe(process.stderr);
};

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

  execSync(`cd ${baseDir}/apps/client/frontend && npm run build`, {
    stdio: 'inherit',
  });

  if (cpCapture) {
    console.log(colorette.blueBright('Stopping client...'));
    cpCapture.kill('SIGINT');
    cpCapture.kill('SIGTERM');
    cpCapture = null;
  }

  const distPath = join(baseDir, 'dist');

  const binPath = join(distPath, 'stream.exe');
  const builder = exec(
    `cd ${captureDir} && go build -tags dev -gcflags "all=-N -l" -v -o ${binPath}`,

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
        env: { ...process.env, ...runtimeEnv },
        cwd: captureDir,
      });
      cpCapture.stdout.pipe(process.stdout);
      cpCapture.stderr.pipe(process.stderr);
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

chokidar
  .watch([
    join(captureDir, '**', '*.go'),
    join(captureDir, 'frontend', 'src', '*.ts'),
    join(captureDir, 'frontend', 'src', '*.tsx'),
  ])
  .on('change', spawnCapture);

chokidar.watch(join(serverDir, '**', '*.go')).on('change', spawnServer);

serveClientGui();
spawnCapture();
spawnServer();
serveWebapp();
