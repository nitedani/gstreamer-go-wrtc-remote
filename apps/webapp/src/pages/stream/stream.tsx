/* eslint-disable @typescript-eslint/no-non-null-assertion */
import './stream.scss';
import { useCallback, useEffect, useRef, useState } from 'react';
import axios from 'axios';
import { forceStereoAudio, setOpusAttributes } from './sdp';
import FullscreenIcon from '@mui/icons-material/Fullscreen';
import MouseIcon from '@mui/icons-material/Mouse';
import MouseOutlinedIcon from '@mui/icons-material/MouseOutlined';
import {
  Backdrop,
  Box,
  Button,
  Checkbox,
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
import { shortcut } from 'src/utils/shortcut';
import { names } from 'src/utils/keys';
import { parseEvent } from 'src/utils/parse';
import io from 'socket.io-client';

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

  const animateClick = useCallback((pressed: boolean) => {
    if (cursorRef.current) {
      if (pressed) {
        cursorRef.current.classList.add('scaled');
      } else {
        cursorRef.current.classList.remove('scaled');
      }
    }
  }, []);

  const handleCursorToggle = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>, checked: boolean) => {
      if (cursorRef.current) {
        cursorRef.current.style.display = checked ? 'block' : 'none';
      }
    },
    [],
  );

  // eslint-disable-next-line sonarjs/cognitive-complexity
  useEffect(() => {
    let pc: RTCPeerConnection;
    const controller = new AbortController();
    const socket = io({
      path: '/api/socket',
      transports: ['polling'],
      query: {
        streamId,
      },
    });

    socket.on('conn_ev', (event: any) => {
      console.log(event);
    });

    (async () => {
      const iceServers = await axios
        .get('/api/ice-config')
        .then((res) => res.data);
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
        socket.emit(
          'signal',
          JSON.stringify(
            candidates.map((candidate) => ({
              type: 'candidate',
              candidate,
            })),
          ),
        );
      };

      const pendingCandidates: RTCIceCandidate[] = [];
      const incomingPendingCandidates: RTCIceCandidate[] = [];
      pc.onicecandidate = async (e) => {
        if (e.candidate && e.candidate.candidate !== '') {
          if (!pc.remoteDescription) {
            pendingCandidates.push(e.candidate);
            return;
          }
          await sendCandidate([e.candidate]);
        }
      };

      const dc = pc.createDataChannel('data');

      socket.on('signal', async (signal: any) => {
        if (signal.type === 'candidate') {
          setLogLines((prev) => [...prev, 'Received ICE candidate...']);
          if (!pc.remoteDescription) {
            incomingPendingCandidates.push(signal.candidate);
            return;
          }

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
          if (incomingPendingCandidates.length) {
            for (const c of incomingPendingCandidates) {
              pc.addIceCandidate(c);
            }
          }
        } else if (signal.type === 'offer') {
          setLogLines((prev) => [...prev, 'Received renegotiation offer...']);
          const patchedRemote = {
            type: signal.type,
            sdp: sdpTransform(signal.sdp),
          };
          pc.setRemoteDescription(patchedRemote);
          const answer = await pc.createAnswer();
          await pc.setLocalDescription(answer);
          setLogLines((prev) => [...prev, 'Sending renegotiation answer...']);
          socket.emit(
            'signal',
            JSON.stringify([
              {
                type: 'answer',
                sdp: answer.sdp,
              },
            ]),
          );
        }
      });

      pc.createOffer().then(async (offer) => {
        const patchedLocal = {
          type: offer.type,
          sdp: sdpTransform(offer.sdp!),
        };

        pc.setLocalDescription(patchedLocal);

        setLogLines((prev) => [...prev, 'Sending offer...']);
        socket.emit('signal', JSON.stringify([patchedLocal]));
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

      dc.onmessage = async (e) => {
        const json = await parseEvent<{
          type: 's_move' | 's_mousedown' | 's_mouseup';
          normX: number;
          normY: number;
        }>(e);

        // view

        const width = videoRef.current!.clientWidth;
        const height = videoRef.current!.clientHeight;

        switch (json.type) {
          case 's_move':
            {
              const x = json.normX * width;
              const y = json.normY * height;
              setCursorPosition({ x, y });
            }
            break;
          case 's_mousedown':
            {
              animateClick(true);
            }
            break;
          case 's_mouseup':
            {
              animateClick(false);
            }
            break;
          default:
            break;
        }
      };

      dc.onopen = () => {
        videoRef.current!.onmousemove = (e) => {
          console.log(
            (document as any).pointerLockElement === videoRef.current ||
              (document as any).mozPointerLockElement === videoRef.current,
          );
          if (
            (document as any).pointerLockElement === videoRef.current ||
            (document as any).mozPointerLockElement === videoRef.current
          ) {
            dc.send(
              JSON.stringify({
                type: 'move_raw',
                movementX: e.movementX,
                movementY: e.movementY,
              }),
            );
          } else {
            const width = videoRef.current!.clientWidth;
            const height = videoRef.current!.clientHeight;
            //normalize mouse position
            const normX = e.offsetX / width;
            const normY = e.offsetY / height;
            dc.send(
              JSON.stringify({
                type: 'move',
                normX,
                normY,
              }),
            );
          }
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

        videoRef.current!.onclick = () => {
          // works, but I disabled it for now
          // videoRef.current!.requestPointerLock();
        };

        document.addEventListener('keydown', (e) => {
          let key = e.key;
          if (e.keyCode in names) {
            key = names[e.keyCode];
          }
          console.log('keydown', e);
          e.stopPropagation();
          e.preventDefault();
          dc.send(JSON.stringify({ type: 'keydown', key: e.key }));
        });

        document.addEventListener('keyup', (e) => {
          let key = e.key;
          if (e.keyCode in names) {
            key = names[e.keyCode];
          }
          console.log('keyup', e);
          e.stopPropagation();
          e.preventDefault();
          dc.send(JSON.stringify({ type: 'keyup', key: e.key }));
        });
      };
    })();

    return () => {
      pc.close();
      controller.abort();
      socket.disconnect();
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
        <>
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
        </>
      )}

      <Box
        className="video-container"
        sx={{
          border: 0,
          maxWidth: '100vw',
          maxHeight: '100vh',
        }}
      >
        <div
          style={{
            position: 'relative',
            display: 'flex',
          }}
        >
          {!loading && (
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
          )}

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
          <Box
            className="controls"
            sx={{
              display: 'flex',
              alignItems: 'center',
            }}
          >
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
            <Checkbox
              defaultChecked={true}
              onChange={handleCursorToggle}
              icon={<MouseOutlinedIcon />}
              checkedIcon={<MouseIcon />}
            />
            <Box
              className="volume-container"
              sx={{
                width: 'fit-content',
              }}
            >
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
                  sx={{ width: 160, px: '8px' }}
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
            </Box>
          </Box>
        )}
      </Box>
    </Box>
  );
};
