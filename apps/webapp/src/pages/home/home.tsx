import { Stack, Typography } from '@mui/material';
import { Box } from '@mui/system';
import { useQuery } from 'react-query';
import { useNavigate } from 'react-router-dom';
import { getStreams } from 'src/api/api';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import { useEffect, useState } from 'react';

const Placeholder = ({ streamId }: { streamId: string }) => {
  const [state, setState] = useState(Date.now());
  useEffect(() => {
    setInterval(() => {
      setState(Date.now());
    }, 5000);
  }, []);

  return (
    <img
      src={`/api/snapshot/${streamId}?${state}`}
      alt=""
      style={{
        position: 'absolute',
        inset: 0,
        width: '100%',
        height: '100%',
      }}
    />
  );
};

export const Home = () => {
  const navigate = useNavigate();
  const { data } = useQuery('streams', () => getStreams(), {
    refetchInterval: 1000,
  });
  if (!data) {
    return null;
  }
  return (
    <Box p={2}>
      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        spacing={{ xs: 1, sm: 2, md: 4 }}
      >
        {data.map((stream) => (
          <Box
            key={stream.streamId}
            sx={{
              padding: '12px',
              cursor: 'pointer',
              border: '1px solid transparent',
              transition: 'all 0.15s ease-in-out',
              '&:hover': {
                border: '1px solid #ffab95',
                borderRadius: '5px',
              },
            }}
            onClick={() => {
              navigate(`/stream/${stream.streamId}`);
            }}
          >
            <Box
              sx={{
                position: 'relative',
                width: '16rem',
                height: '9rem',
                background: 'black',
              }}
            >
              <Placeholder streamId={stream.streamId} />
              <Box
                display="flex"
                alignItems="center"
                justifyContent="center"
                gap={0.5}
                sx={{
                  position: 'absolute',
                  bottom: 0,
                  left: 0,
                  p: 1,
                  width: 80,
                  backdropFilter: 'blur(5px)',
                  background: 'rgba(0,0,0,0.2)',
                }}
              >
                <Typography>{(stream.uptime / 60).toFixed(0)} mins</Typography>
              </Box>

              <Box
                display="flex"
                alignItems="center"
                justifyContent="center"
                gap={0.5}
                sx={{
                  position: 'absolute',
                  width: 50,
                  bottom: 0,
                  right: 0,
                  p: 1,
                  backdropFilter: 'blur(5px)',
                  background: 'rgba(0,0,0,0.2)',
                }}
              >
                <Typography>{stream.viewers}</Typography>
                <VisibilityOutlinedIcon
                  sx={{ marginTop: '2px', fontSize: '1.1em' }}
                />
              </Box>
            </Box>
            <Typography fontSize={18} textTransform="capitalize">
              {stream.streamId}
            </Typography>
          </Box>
        ))}
      </Stack>
    </Box>
  );
};
