/* eslint-disable @typescript-eslint/no-non-null-assertion */
import './logo.svg';
import { useEffect, useRef } from 'react';

import axios from 'axios';
import { forceStereoAudio, setOpusAttributes } from './sdp';
import FullscreenIcon from '@mui/icons-material/Fullscreen';
import { IconButton } from '@mui/material';
import { grey } from '@mui/material/colors';

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

  useEffect(() => {
    axios
      .post('/api/connect')
      .then(() => axios.get('/api/ice-config'))
      .then((res) => res.data)
      .then((iceServers) => {
        const pc = new RTCPeerConnection({
          iceServers,
        });

        //pc.addTransceiver('audio', { direction: 'sendrecv' });
        pc.addTransceiver('video', { direction: 'sendrecv' });

        pc.ontrack = (event) => {
          videoRef.current!.srcObject = event.streams[0];
        };

        pc.onicecandidate = async (e) => {
          if (e.candidate && e.candidate.candidate !== '') {
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
                pc.addIceCandidate(signal.candidate);
              } else if (signal.type === 'answer') {
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

        dc.onopen = () => {
          videoRef.current!.onmousemove = (e) => {
            const width = videoRef.current!.clientWidth;
            const height = videoRef.current!.clientHeight;

            //normalize mouse position
            const normX = e.offsetX / width;
            const normY = e.offsetY / height;

            dc.send(JSON.stringify({ type: 'move', normX, normY }));
          };

          videoRef.current!.onmousedown = (e) => {
            dc.send(JSON.stringify({ type: 'mousedown', button: e.button }));
          };

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
      <div className="video-container">
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
                  controls.style.display = 'unset';
                }
              };
            }}
          >
            <FullscreenIcon />
          </IconButton>
        </div>
        <video className="video-height" muted autoPlay ref={videoRef}></video>
      </div>
    </div>
  );
};

export default App;
