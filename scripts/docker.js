const { join, resolve } = require('path');
const { mkdtempSync, writeFileSync, chmodSync } = require('fs');
const { promisify } = require('util');
const exec = promisify(require('child_process').exec);
const os = require('os');
const rimraf = require('rimraf');
const webappOptions = require('../apps/webapp/webpack.config.js');
const webpack = require('webpack');

const runWebpack = (compiler) =>
  new Promise((resolve) => {
    compiler.run(() => {
      resolve();
    });
  });

const baseDir = resolve(__dirname, '..');
const serverDir = join(baseDir, 'apps', 'server');

const buildAndPushServerDocker = async () => {
  const TMPpath = mkdtempSync(join(os.tmpdir(), 'build-docker-'));
  const TMPwebappPath = join(TMPpath, 'webapp');
  const TMPbuildPath = join(TMPpath, 'main');

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

  const dockerfile = `\
FROM golang:1.18
ENV GO_ENV=release
WORKDIR /app
ADD . ./  
EXPOSE 4000  
CMD [ "./main", "./config" ]
`;

  const dockerfilePath = join(TMPpath, 'Dockerfile');
  writeFileSync(dockerfilePath, dockerfile);
  chmodSync(TMPpath, '0777');
  chmodSync(dockerfilePath, '0777');

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

  writeFileSync(join(TMPpath, 'config'), exampleConfig);

  console.log('Building docker image...');
  await exec(
    [
      'docker',
      'build',
      TMPpath,
      '-t',
      'nitedani/gstreamer-go-wrtc-remote:latest',
    ].join(' '),
    {
      stdio: 'inherit',
    },
  );

  console.log('Pushing docker image...');
  await exec(
    ['docker', 'push', 'nitedani/gstreamer-go-wrtc-remote:latest'].join(' '),
    {
      stdio: 'inherit',
    },
  );

  rimraf.sync(TMPpath);
};

(async () => {
  await buildAndPushServerDocker();
})();
