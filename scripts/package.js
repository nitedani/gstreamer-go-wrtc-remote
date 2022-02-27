const { join, resolve } = require('path');
const { cwd } = require('process');
const { path7za } = require('7zip-bin');
const { spawnSync, execSync } = require('child_process');
const { mkdtempSync, mkdirSync, writeFileSync } = require('fs');
const os = require('os');
const rimraf = require('rimraf');
const webappOptions = require('../apps/webapp/webpack.config.js');
const webpack = require('webpack');

const distPath = join(cwd(), 'dist');
const sfxPath = join(cwd(), 'sfx');
const sfxFilePath = join(sfxPath, '7z.sfx');
const sfxConfigPath = join(sfxPath, 'sfx.txt');

const runWebpack = (compiler) =>
  new Promise((resolve) => {
    compiler.run(() => {
      resolve();
    });
  });

const baseDir = resolve(__dirname, '..');
const streamServerDir = join(baseDir, 'apps', 'server');
const signalingServerDir = join(baseDir, 'apps', 'signaling');

const buildStreamServer = async () => {
  const gstreamerDlls = join(streamServerDir, 'gstreamer', 'dll');
  const TMPpath = mkdtempSync(join(os.tmpdir(), 'build-win64-'));
  const TMPbuildPath = join(TMPpath, 'main.exe');
  const TMParchivePath = join(TMPpath, 'streamserver-win64.7z');

  const finalPath = join(distPath, 'streamserver-win64');
  const finalExecutablePath = join(finalPath, 'streamserver-win64.exe');
  mkdirSync(finalPath);

  const buildEnv = {
    CGO_ENABLED: 1,
    CGO_CFLAGS: `-I${join(streamServerDir, 'gstreamer', 'include')}`,
    CGO_LDFLAGS: `-L${join(streamServerDir, 'gstreamer', 'lib')}`,
    PKG_CONFIG_PATH: `${join(
      streamServerDir,
      'gstreamer',
      'lib',
      'pkgconfig',
    )}`,
  };

  execSync(
    `cd ${streamServerDir}\\main && go build -ldflags=\"-s -w\" -v -o ${TMPbuildPath}`,
    {
      stdio: 'inherit',
      env: { ...process.env, ...buildEnv },
    },
  );

  //Package streamserver
  //Make 7z archive
  const startupScript = `
@echo off
set PATH=%PATH%;dll
set GST_PLUGIN_PATH_1_0=plugins
set GO_ENV=release
main.exe %1`;

  writeFileSync(join(TMPpath, 'start.bat'), startupScript);

  spawnSync(
    path7za,
    [
      'a',
      //Output
      TMParchivePath,
      //Server file
      join(TMPpath, '*'),
      //gstreamer dlls
      join(gstreamerDlls, '*'),
    ],
    {
      stdio: 'inherit',
    },
  );

  //Make sfx exe
  execSync(
    `COPY /b "${sfxFilePath}" + "${sfxConfigPath}" + "${TMParchivePath}" "${finalExecutablePath}"`,
    {
      stdio: 'inherit',
    },
  );

  const exampleConfig = `\
STREAM_ID=default
REMOTE_ENABLED=true
BITRATE=15388600
RESOLUTION=1920x1080
FRAMERATE=90
THREADS=6
SIGNAL_SERVER_URL=http://localhost:4000/api
STUN_SERVER_URL=stun:stun.l.google.com:19302
STUN_SERVER_USERNAME=
STUN_SERVER_PASSWORD=
TURN_SERVER_URL=
TURN_SERVER_USERNAME=
TURN_SERVER_PASSWORD=
`;

  writeFileSync(join(finalPath, 'config'), exampleConfig);

  rimraf.sync(TMPpath);
};

const buildSignalingServer = async () => {
  const TMPpath = mkdtempSync(join(os.tmpdir(), 'build-win64-'));
  const TMPwebappPath = join(TMPpath, 'webapp');
  const TMPbuildPath = join(TMPpath, 'main.exe');
  const TMParchivePath = join(TMPpath, 'signalserver-win64.7z');

  const finalPath = join(distPath, 'signalserver-win64');
  const finalExecutablePath = join(finalPath, 'signalserver-win64.exe');
  mkdirSync(finalPath);

  const buildEnv = {};

  execSync(
    `cd ${signalingServerDir}\\main && go build -ldflags=\"-s -w\" -v -o ${TMPbuildPath}`,
    {
      stdio: 'inherit',
      env: { ...process.env, ...buildEnv },
    },
  );

  //Build webapp
  await runWebpack(
    webpack({
      ...webappOptions,
      output: { ...webappOptions.output, path: TMPwebappPath },
    }),
  );

  //Package signalserver+webapp
  //Make 7z archive
  const startupScript = `
@echo off
set GO_ENV=release
main.exe %1`;

  writeFileSync(join(TMPpath, 'start.bat'), startupScript);

  spawnSync(
    path7za,
    [
      'a',
      //Output
      TMParchivePath,
      //Server file + webapp
      join(TMPpath, '*'),
    ],
    {
      stdio: 'inherit',
    },
  );

  //Make sfx exe
  execSync(
    `COPY /b "${sfxFilePath}" + "${sfxConfigPath}" + "${TMParchivePath}" "${finalExecutablePath}"`,
    {
      stdio: 'inherit',
    },
  );

  const exampleConfig = `\
PORT=4000
STUN_SERVER_URL=stun:stun.l.google.com:19302
STUN_SERVER_USERNAME=
STUN_SERVER_PASSWORD=
TURN_SERVER_URL=
TURN_SERVER_USERNAME=
TURN_SERVER_PASSWORD=
`;

  writeFileSync(join(finalPath, 'config'), exampleConfig);

  rimraf.sync(TMPpath);
};

(async () => {
  rimraf.sync(distPath);
  mkdirSync(distPath);
  await buildStreamServer();
  await buildSignalingServer();
})();
