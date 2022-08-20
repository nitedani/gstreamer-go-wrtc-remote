const { join, resolve } = require('path');
const { cwd } = require('process');
const { path7za } = require('7zip-bin');
const { spawnSync, execSync } = require('child_process');
const { mkdtempSync, mkdirSync, writeFileSync, chmodSync } = require('fs');
const { promisify } = require('util');
const exec = promisify(require('child_process').exec);
const os = require('os');
const rimraf = require('rimraf');
const webappOptions = require('../apps/webapp/webpack.config.js');
const webpack = require('webpack');

const distPath = join(cwd(), 'dist');
const sfxPath = join(cwd(), 'sfx');
const sfxFilePath = join(sfxPath, '7z.sfx');
const makeselfPath = join(sfxPath, 'makeself', 'makeself.sh');

const runWebpack = (compiler) =>
  new Promise((resolve) => {
    compiler.run(() => {
      resolve();
    });
  });

const baseDir = resolve(__dirname, '..');
const clientDir = join(baseDir, 'apps', 'client');
const serverDir = join(baseDir, 'apps', 'server');

const buildCaptureWin = async () => {
  const gstreamerDlls = join(clientDir, 'gstreamer', 'dll');
  const TMPpath = mkdtempSync(join(os.tmpdir(), 'build-win64-'));
  const TMPbuildPath = join(TMPpath, 'main.exe');
  const TMParchivePath = join(TMPpath, 'capture-win64.7z');

  const finalPath = join(distPath, 'capture-win64');
  const finalAutorunExecutablePath = join(finalPath, 'capture-win64.exe');
  const finalSfxexecutablePath = join(finalPath, 'capture-win64.sfx.exe');
  mkdirSync(finalPath);

  const buildEnv = {
    CGO_ENABLED: 1,
    CGO_CFLAGS: `-I${join(clientDir, 'gstreamer', 'include')}`,
    CGO_LDFLAGS: `-L${join(clientDir, 'gstreamer', 'lib')} -L${join(
      clientDir,
      'dll',
      'x64',
    )}`,
    PKG_CONFIG_PATH: `${join(clientDir, 'gstreamer', 'lib', 'pkgconfig')}`,
  };
  /*
  await exec(`cd ${streamServerDir}\\main && go clean -cache`, {
    stdio: 'inherit',
    env: { ...process.env, ...buildEnv },
  });
*/
  execSync(`cd ${clientDir}/frontend && npm run build`, {
    stdio: 'inherit',
  });

  await exec(
    ///////////////////////////////////////////////////// -ldflags \"-w -h -H windowsgui\"
    `cd ${clientDir} && go build -tags desktop,production -ldflags=\"-s -w\" -v -o ${TMPbuildPath}`,
    {
      stdio: 'inherit',
      env: { ...process.env, ...buildEnv },
    },
  );

  //Package capture
  //Make 7z archive
  const startupScript = `
@echo off
set PATH=%PATH%;dll
set GST_PLUGIN_PATH_1_0=plugins
set GO_ENV=release
call main.exe %1`;

  writeFileSync(join(TMPpath, 'start.bat'), startupScript);

  const sfxAutorunScript = `
;!@Install@!UTF-8!
GUIMode="2"
ExecuteFile="start.bat"
ExecuteParameters="%%S\\config.json"
;!@InstallEnd@!`;

  const sfxAutorunScriptPath = join(TMPpath, 'sfx-autorun.txt');
  writeFileSync(sfxAutorunScriptPath, sfxAutorunScript);

  const sfxScript = `
;!@Install@!UTF-8!
InstallPath="%%S/screen-capture"
GUIMode="2"
;!@InstallEnd@!`;

  const sfxScriptPath = join(TMPpath, 'sfx.txt');
  writeFileSync(sfxScriptPath, sfxScript);

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

  //Make sfx autorun exe
  await exec(
    `COPY /b "${sfxFilePath}" + "${sfxAutorunScriptPath}" + "${TMParchivePath}" "${finalAutorunExecutablePath}"`,
    {
      stdio: 'inherit',
    },
  );

  //Make sfx exe
  await exec(
    `COPY /b "${sfxFilePath}" + "${sfxScriptPath}" + "${TMParchivePath}" "${finalSfxexecutablePath}"`,
    {
      stdio: 'inherit',
    },
  );

  const exampleConfig = `\
{
  "settings": {
    "stream_id": "stream_test",
    "private": false,
    "remote_enabled": false,
    "direct_connect": true,
    "bitrate": 10485760,
    "resolution": "1920x1080",
    "framerate": 60,
    "encoder": "nvenc",
    "threads": 4,
    "server_url": "http://localhost:4000/api"
  }
}
`;

  writeFileSync(join(finalPath, 'config.json'), exampleConfig);

  rimraf.sync(TMPpath);
};

const buildServerWin = async () => {
  const TMPpath = mkdtempSync(join(os.tmpdir(), 'build-win64-'));
  const TMPwebappPath = join(TMPpath, 'webapp');
  const TMPbuildPath = join(TMPpath, 'main.exe');
  const TMParchivePath = join(TMPpath, 'server-win64.7z');

  const finalPath = join(distPath, 'server-win64');
  const finalExecutablePath = join(finalPath, 'server-win64.exe');
  mkdirSync(finalPath);

  const buildEnv = {
    GOOS: 'windows',
    GOARCH: 'amd64',
  };

  await exec(
    `cd ${serverDir}\\main && go build -ldflags=\"-s -w\" -v -o ${TMPbuildPath}`,
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

  //Package server+webapp
  //Make 7z archive
  const startupScript = `
@echo off
set GO_ENV=release
call main.exe %1`;

  writeFileSync(join(TMPpath, 'start.bat'), startupScript);
  const sfxScript = `
;!@Install@!UTF-8!
GUIMode="2"
ExecuteFile="start.bat"
ExecuteParameters="%%S\\config"
;!@InstallEnd@!`;

  const sfxScriptPath = join(TMPpath, 'sfx.txt');
  writeFileSync(sfxScriptPath, sfxScript);

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
  await exec(
    `COPY /b "${sfxFilePath}" + "${sfxScriptPath}" + "${TMParchivePath}" "${finalExecutablePath}"`,
    {
      stdio: 'inherit',
    },
  );

  const exampleConfig = `\
PORT=4000
DIRECT_CONNECT=false
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

const buildServerLinux = async () => {
  const TMPpath = mkdtempSync(join(os.tmpdir(), 'build-linux64-'));
  const TMPwebappPath = join(TMPpath, 'webapp');
  const TMPbuildPath = join(TMPpath, 'main');

  const finalPath = join(distPath, 'server-linux64');
  const finalExecutablePath = join(finalPath, 'server-linux64');
  mkdirSync(finalPath);

  const buildEnv = {
    GOOS: 'linux',
    GOARCH: 'amd64',
  };

  await exec(
    `cd ${serverDir}\\main && go build -ldflags=\"-s -w\" -v -o ${TMPbuildPath}`,
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

  //Package server+webapp
  //Make 7z archive
  const startupScript = `
export GO_ENV=release
./main $1`;

  const scriptPath = join(TMPpath, 'start.sh');
  writeFileSync(scriptPath, startupScript);
  chmodSync(TMPpath, '0777');
  chmodSync(scriptPath, '0777');
  await exec(
    [
      'chmod',
      '755',
      scriptPath,

      '&&',

      'chmod',
      '-R',
      '755',
      TMPpath,

      '&&',

      makeselfPath,
      '--notemp',
      //Server file + webapp
      TMPpath,
      //Output
      finalExecutablePath,
      '""',
      './start.sh',
      '$(pwd)/config',
    ].join(' '),
    {
      stdio: 'inherit',
    },
  );

  const exampleConfig = `\
PORT=4000
DIRECT_CONNECT=false
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
  await Promise.all([buildCaptureWin(), buildServerWin(), buildServerLinux()]);
})();
