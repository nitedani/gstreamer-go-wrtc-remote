## High performance, low latency screen sharing/remote control

### Libraries used:

- https://github.com/tinyzimmer/go-gst
- https://github.com/pion/webrtc
- https://github.com/go-vgo/robotgo
- https://github.com/labstack/echo

### OS checklist

- [x] Windows
- [ ] Linux
- [ ] Mac

### Encoder checklist

- [x] VP8
- [x] OpenH264
- [x] NVENC H264
- [ ] X264
- [ ] QuickSync
- [ ] AMF

### Feature checklist

- [x] Video
- [x] Audio
- [x] Remote mouse
- [x] Remote keyboard
- [x] Collaborative control
- [ ] Remote clipboard
- [ ] Drag-n-drop file transfer
- [ ] Centralized, deployable SFU service

## How it works:

Streamserver:

- captures the desktop and audio, sends them to the viewers
- receives control commands from the viewers

Signalserver:

- used to establish the webrtc connection between streamserver and the viewers
- needs to be accessable by both the streamserver and the browser

## Setting up the signalserver:

1. Edit signalserver-win64/config
2. Run signalserver-win64.exe

Example signalserver config:

```
PORT=4000
STUN_SERVER_URL=stun:stun.l.google.com:19302
STUN_SERVER_USERNAME=
STUN_SERVER_PASSWORD=
TURN_SERVER_URL=
TURN_SERVER_USERNAME=
TURN_SERVER_PASSWORD=
```

## Setting up the streamserver:

1. Edit streamserver-win64/config
2. Run streamserver-win64.exe

Example streamserver config:

```
SIGNAL_SERVER_URL=http://localhost:4000/api
STREAM_ID=default
REMOTE_ENABLED=true
BITRATE=15388600
RESOLUTION=1920x1080
FRAMERATE=90
THREADS=4
#ENCODER=h264
#ENCODER=vp8
ENCODER=nvenc
```

The stream is available on: http://{signalserver url}?streamId={STREAM_ID}

![](/docs/1.png)

## Development:

Requirements: same as build requirements

1. In VS-code File->Open Workspace from File->select the included workspace file
2. open signalserver/.env, customize
3. open streamserver/.env, customize
4. `npm i`
5. `npm run start:dev`

this command:

- starts the streamserver
- starts the signalserver, listening on port 4000
- starts the webpack devserver for the webapp on port 3000, redirects /api calls to localhost:4000(signalserver)
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
2. `npm run build` produces binaries with sample config in /dist

How to install chocolatey:

```
Set-ExecutionPolicy Bypass -Scope Process -Force; `
  iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
```
