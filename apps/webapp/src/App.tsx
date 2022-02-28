/* eslint-disable @typescript-eslint/no-non-null-assertion */
import './logo.svg';
import { useCallback, useEffect, useRef, useState } from 'react';

import axios from 'axios';
import { forceStereoAudio, setOpusAttributes } from './sdp';
import FullscreenIcon from '@mui/icons-material/Fullscreen';
import {
  Backdrop,
  CircularProgress,
  IconButton,
  Slider,
  Stack,
} from '@mui/material';
import { blue, grey } from '@mui/material/colors';
import VolumeDown from '@mui/icons-material/VolumeDown';
import VolumeUp from '@mui/icons-material/VolumeUp';

const sdpTransform = (sdp: string) => {
  let sdp2 = sdp
    .replace(/(m=video.*\r\n)/g, `$1b=AS:${15 * 1024}\r\n`)
    .replace(/(m=audio.*\r\n)/g, `$1b=AS:${128}\r\n`);

  sdp2 = forceStereoAudio(sdp2);
  sdp2 = setOpusAttributes(sdp2, {
    stereo: 1,
    maxaveragebitrate: 128 * 1024 * 8,
    maxplaybackrate: 128 * 1024 * 8,
    maxptime: 3,
  });

  return sdp2;
};

const App = () => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamId = window.location.search.split('=')[1];
  const signalPath = `/api/signal/${streamId}`;
  const [loading, setLoading] = useState(true);
  const [logLines, setLogLines] = useState<string[]>([]);
  const [volume, setVolume] = useState(0);

  const handleVolumeChange = useCallback(
    (event: Event, value: number | number[]) => {
      setVolume(value as number);
      videoRef.current!.muted = false;
      videoRef.current!.volume = (value as number) / 100;
    },
    [],
  );

  useEffect(() => {
    axios
      .post('/api/connect')
      .then(() => axios.get('/api/ice-config'))
      .then((res) => res.data)
      .then((iceServers) => {
        const pc = new RTCPeerConnection({
          iceServers,
        });

        pc.addTransceiver('video', { direction: 'sendrecv' });
        pc.addTransceiver('audio', { direction: 'sendrecv' });

        pc.ontrack = (event) => {
          console.log(event.streams[0].getVideoTracks());
          console.log(event.streams[0].getAudioTracks());
          videoRef.current!.srcObject = event.streams[0];
        };

        pc.onicecandidate = async (e) => {
          if (e.candidate && e.candidate.candidate !== '') {
            setLogLines((prev) => [...prev, 'Sending ICE candidate...']);
            await axios.post(signalPath, {
              type: 'candidate',
              candidate: e.candidate,
            });
          }
        };

        const pollSignal = async () => {
          try {
            const res = await axios.get(signalPath, { timeout: 30000 });
            for (const signal of res.data) {
              if (signal.type === 'candidate') {
                setLogLines((prev) => [...prev, 'Received ICE candidate...']);
                pc.addIceCandidate(signal.candidate);
              } else if (signal.type === 'answer') {
                setLogLines((prev) => [...prev, 'Received answer...']);
                const patchedRemote = {
                  type: signal.type,
                  sdp: sdpTransform(signal.sdp),
                };

                pc.setRemoteDescription(patchedRemote);
              }
            }

            pollSignal();
          } catch (error) {
            console.log(error);
          }
        };
        const dc = pc.createDataChannel('data');
        pc.createOffer().then(async (offer) => {
          const patchedLocal = {
            type: offer.type,
            sdp: sdpTransform(offer.sdp!),
          };

          pc.setLocalDescription(patchedLocal);

          setLogLines((prev) => [...prev, 'Sending offer...']);
          await axios.post(signalPath, patchedLocal);
          pollSignal();
        });

        // remote control

        videoRef.current!.addEventListener(
          'contextmenu',
          (ev) => {
            ev.preventDefault();
            return false;
          },
          false,
        );

        videoRef.current!.onplay = () => {
          setLoading(false);
        };

        let isVolumeSet = false;

        videoRef.current!.addEventListener('mousedown', () => {
          if (!isVolumeSet) {
            videoRef.current!.muted = false;
            videoRef.current!.volume = 0.5;
            setVolume(50);
            isVolumeSet = true;
          }
        });

        dc.onopen = () => {
          videoRef.current!.onmousemove = (e) => {
            const width = videoRef.current!.clientWidth;
            const height = videoRef.current!.clientHeight;

            //normalize mouse position
            const normX = e.offsetX / width;
            const normY = e.offsetY / height;

            dc.send(JSON.stringify({ type: 'move', normX, normY }));
          };

          videoRef.current!.addEventListener('mousedown', (e) => {
            dc.send(JSON.stringify({ type: 'mousedown', button: e.button }));
          });

          videoRef.current!.onmouseup = (e) => {
            dc.send(JSON.stringify({ type: 'mouseup', button: e.button }));
          };

          videoRef.current!.onwheel = (e) => {
            dc.send(JSON.stringify({ type: 'wheel', delta: e.deltaY }));
          };

          document.addEventListener('keydown', (e) => {
            dc.send(JSON.stringify({ type: 'keydown', key: e.key }));
          });

          document.addEventListener('keyup', (e) => {
            dc.send(JSON.stringify({ type: 'keyup', key: e.key }));
          });
        };
      });
  }, []);

  return (
    <div className="App">
      <Backdrop
        sx={{ color: '#fff', zIndex: (theme) => theme.zIndex.drawer + 1 }}
        open={loading}
      >
        <CircularProgress
          color="inherit"
          style={{
            color: blue[500],
          }}
        />
      </Backdrop>
      <div className="video-container">
        <video className="video-height" muted autoPlay ref={videoRef}></video>
        {!loading && (
          <div className="controls">
            <IconButton
              style={{ color: grey[500] }}
              aria-label="full-screen"
              onClick={() => {
                const controls = document.querySelector(
                  '.controls',
                ) as HTMLDivElement;
                const videoContainer = document.querySelector(
                  '.video-container',
                ) as HTMLDivElement;
                videoContainer.requestFullscreen();

                videoContainer.onfullscreenchange = () => {
                  if (document.fullscreenElement) {
                    controls.style.display = 'none';
                    videoRef.current!.classList.remove('video-height');
                  } else {
                    videoRef.current!.classList.add('video-height');
                    controls.style.display = 'flex';
                  }
                };
              }}
            >
              <FullscreenIcon />
            </IconButton>
            <div className="volume-container">
              <Stack
                spacing={2}
                direction="row"
                sx={{ mb: 1 }}
                alignItems="center"
              >
                <VolumeDown fontSize="small" />
                <Slider
                  aria-label="Volume"
                  value={volume}
                  onChange={handleVolumeChange}
                  size="small"
                />
                <VolumeUp fontSize="small" />
              </Stack>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default App;
