/* eslint-disable @typescript-eslint/no-non-null-assertion */
import './stream.scss';
import { useCallback, useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { forceStereoAudio, setOpusAttributes } from './sdp';
import FullscreenIcon from '@mui/icons-material/Fullscreen';
import {
  Backdrop,
  Box,
  Button,
  CircularProgress,
  IconButton,
  Slider,
  Stack,
} from '@mui/material';
import { blue, grey } from '@mui/material/colors';
import VolumeDown from '@mui/icons-material/VolumeDown';
import VolumeUp from '@mui/icons-material/VolumeUp';
import { useParams } from 'react-router-dom';
import { useStore } from 'src/store/store';

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

export const Stream = () => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const { streamId } = useParams<{ streamId: string }>();
  const signalPath = `/api/signal/${streamId}`;
  const [loading, setLoading] = useState(true);
  const [logLines, setLogLines] = useState<string[]>([]);
  const { volume, setVolume } = useStore();

  const handleVolumeChange = useCallback(
    (event: Event, value: number | number[]) => {
      setVolume((value as number) / 100);
      videoRef.current!.muted = false;
      videoRef.current!.volume = (value as number) / 100;
    },
    [],
  );
  const cursorRef = useRef<HTMLDivElement>(null);

  const setCursorPosition = useCallback(({ x, y }) => {
    if (cursorRef.current) {
      cursorRef.current.style.left = `${x}px`;
      cursorRef.current.style.top = `${y}px`;
    }
  }, []);

  const animateClick = useCallback(() => {
    if (cursorRef.current) {
      cursorRef.current.classList.add('scaled');
      setTimeout(() => {
        if (cursorRef.current) {
          cursorRef.current.classList.remove('scaled');
        }
      }, 200);
    }
  }, []);

  // eslint-disable-next-line sonarjs/cognitive-complexity
  useEffect(() => {
    let pc: RTCPeerConnection;
    const controller = new AbortController();
    axios
      .post('/api/connect')
      .then(() => axios.get('/api/ice-config'))
      .then((res) => res.data)
      .then((iceServers) => {
        pc = new RTCPeerConnection({
          iceServers,
        });

        pc.addTransceiver('video', { direction: 'sendrecv' });
        pc.addTransceiver('audio', { direction: 'sendrecv' });

        pc.ontrack = (event) => {
          console.log(event.streams[0].getVideoTracks());
          console.log(event.streams[0].getAudioTracks());
          videoRef.current!.srcObject = event.streams[0];
        };

        const sendCandidate = async (candidates: RTCIceCandidate[]) => {
          setLogLines((prev) => [...prev, 'Sending pending ICE candidates...']);
          await axios.post(
            signalPath,
            candidates.map((candidate) => ({
              type: 'candidate',
              candidate,
            })),
          );
        };

        const pendingCandidates: RTCIceCandidate[] = [];
        pc.onicecandidate = async (e) => {
          if (e.candidate && e.candidate.candidate !== '') {
            if (!pc.remoteDescription) {
              pendingCandidates.push(e.candidate);
              return;
            }
            await sendCandidate([e.candidate]);
          }
        };

        const pollSignal = async () => {
          try {
            const res = await axios.get(signalPath, {
              timeout: 30000,
              signal: controller.signal,
            });
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
                if (pendingCandidates.length) {
                  await sendCandidate(pendingCandidates);
                }
              } else if (signal.type === 'offer') {
                setLogLines((prev) => [
                  ...prev,
                  'Received renegotiation offer...',
                ]);
                const patchedRemote = {
                  type: signal.type,
                  sdp: sdpTransform(signal.sdp),
                };
                pc.setRemoteDescription(patchedRemote);
                const answer = await pc.createAnswer();
                await pc.setLocalDescription(answer);
                setLogLines((prev) => [
                  ...prev,
                  'Sending renegotiation answer...',
                ]);
                await axios.post(signalPath, [
                  {
                    type: 'answer',
                    sdp: answer.sdp,
                  },
                ]);
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
          await axios.post(signalPath, [patchedLocal]);
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
          // videoRef.current!.muted = false;
          videoRef.current!.volume = volume;
        };

        dc.onmessage = (e) => {
          const data = e.data as ArrayBuffer;
          const width = videoRef.current!.clientWidth;
          const height = videoRef.current!.clientHeight;

          const json = JSON.parse(new TextDecoder().decode(data)) as {
            type: 'move' | 'click';
            normX: number;
            normY: number;
          };

          switch (json.type) {
            case 'move':
              {
                const x = json.normX * width;
                const y = json.normY * height;
                setCursorPosition({ x, y });
              }
              break;
            case 'click': {
              //animateClick();
            }
            break;
            default:
              break;
          }
        };

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
    return () => {
      pc.close();
      controller.abort();
    };
  }, []);

  return (
    <Box
      className="stream-page"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        width: '100vw',
        height: '100vh',
        position: 'relative',
      }}
    >
      {loading && (
        <Box
          sx={{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '300px',
            height: 'fit-content',
            background: '#00000099',
            backdropFilter: 'blur(5px)',
            color: 'white',
            padding: '20px',
            margin: '8px',
            lineHeight: '1.2em',
            border: '1px solid gray',
          }}
        >
          {logLines.map((line, index) => (
            <div key={index}>{line}</div>
          ))}
        </Box>
      )}

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
        <div
          style={{
            position: 'relative',
          }}
        >
          <Box
            ref={cursorRef}
            id="cursor"
            sx={{
              width: 12,
              height: 12,
              borderRadius: 20,
              transform: 'translate(-50%, -50%)',
              backgroundColor: '#eee',
              position: 'absolute',
            }}
          ></Box>
          <video
            className="video-height"
            muted
            autoPlay
            playsInline
            loop
            ref={videoRef}
          ></video>
        </div>

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
              {videoRef.current!.muted && volume !== 0 ? (
                <Button
                  onClick={() => {
                    videoRef.current!.muted = false;
                    useStore.setState((s) => ({ ...s }));
                  }}
                >
                  Unmute
                </Button>
              ) : (
                <Stack
                  spacing={2}
                  direction="row"
                  sx={{ mb: 1 }}
                  alignItems="center"
                >
                  <VolumeDown fontSize="small" />
                  <Slider
                    aria-label="Volume"
                    value={volume * 100}
                    onChange={handleVolumeChange}
                    size="small"
                  />
                  <VolumeUp fontSize="small" />
                </Stack>
              )}
            </div>
          </div>
        )}
      </div>
    </Box>
  );
};
