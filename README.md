## Low latency screen sharing/remote control

Only tested on windows, doesn't include build script for linux.

## Instructions:

Streamserver captures the desktop, encodes the frames and sends them to the browser through webrtc.

Signalserver is used to establish the webrtc connection and needs to be accessable by both the streamserver and the viewers browser.

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

```

The stream is available on: http://{signalserver url}?streamId={STREAM_ID}

![](/docs/1.png)

## Build on windows:

Requirements:

- [mingw](https://chocolatey.org/packages/mingw)
- [pkgconfig](https://chocolatey.org/packages/pkgconfiglite)
- nodejs

1. `npm i`
2. `npm run build` produces binaries with sample config in /dist
