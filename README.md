## Low latency live streaming/remote control in go/react

### Languages/libraries used:

- Webapp: react
- Server: go
- Capture client: go

### Encoders

- VP8
- OpenH264
- NVENC H264

### Features

- Video - 4K120
- Audio
- Remote control (work in progress)

## How it works:

Capture client:

- this is a windows executable
- captures the desktop and audio, sends them to the server/viewers(depending on the server `DIRECT_CONNECT` mode)
- receives control commands from the server/viewers

Server:

- windows, linux executable or docker image
- serves the webapp static files
- needs to be reachable by both the capture client and the browser
- it can run in two modes:
  - `DIRECT_CONNECT=true` - peer to peer, the viewers connect directly to the capture client. In this mode, no video/audio goes through the server. The server only helps to establish the connection between the peers. The capture client(s) seperately send data to each viewer(higher upload bandwidth usage on the capture client, lower latency)
  - `DIRECT_CONNECT=false`(default) - the server acts as a forwarding unit, only one connection is maintained for each active(has viewers) capture client(lower bandwidth usage, higher latency, higher server cpu usage)

## Setting up the server:

### A. Using docker:

`docker run --pull always --network host nitedani/gstreamer-go-wrtc-remote:latest`

The server will be listening on :4000.

### B. Using the executables:

1. Edit server-linux64/config or server-win64/config
2. Run server-linux64 or server-win64.exe

Default server config:

The server respects the host environment variables over the config file.

```
PORT=4000
DIRECT_CONNECT=false
STUN_SERVER_URL=stun:stun.l.google.com:19302
STUN_SERVER_USERNAME=
STUN_SERVER_PASSWORD=
TURN_SERVER_URL=
TURN_SERVER_USERNAME=
TURN_SERVER_PASSWORD=
```

#

## Setting up the capture client:

1. Edit client-win64/config.json
2. Run client-win64.exe

Default capture client config.json:

```
{
  "settings": {
    "server_url": "http://localhost:4000/api",
    "stream_id": "stream_test",
    "remote_enabled": false,
    "direct_connect": true,
    "bitrate": 10388600,
    "resolution": "1920x1080",
    "framerate": 60,
    "encoder": "nvenc",
    "threads": 4
  }
}

```

With the configuration above, the stream is available on: http://localhost:4000?streamId=stream_test
direct_connect set on the client overrides the server setting

## Development:

Requirements: same as build requirements

1. In VS-code File->Open Workspace from File->select the included workspace file
2. open signalserver/.env, customize
3. open streamserver/.env, customize
4. `npm i`
5. `npm run start:dev`

this command:

- starts the capture client
- starts the server, listening on port 4000
- starts the webpack devserver for the webapp on port 3000, redirects /api calls to localhost:4000(server)
- opens the browser on `http://localhost:3000/?streamId=default`

The result should be similar:
![](/docs/desktop.jpg)

## Build on windows:

Requirements:

- [mingw](https://chocolatey.org/packages/mingw)
- [pkgconfig](https://chocolatey.org/packages/pkgconfiglite)
- nodejs 16+
- go 1.18

1. `npm i`
2. `npm run build` produces binaries and config files in /dist

How to install chocolatey:

```
Set-ExecutionPolicy Bypass -Scope Process -Force; `
  iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
```
