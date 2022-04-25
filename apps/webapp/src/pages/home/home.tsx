import { Stack, Typography } from '@mui/material';
import { Box } from '@mui/system';
import { useQuery } from 'react-query';
import { useNavigate } from 'react-router-dom';
import { getStreams } from 'src/api/api';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';

export const Home = () => {
  const navigate = useNavigate();
  const { data } = useQuery('streams', () => getStreams());
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
              cursor: 'pointer',
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
              <img
                src={`/api/snapshot/${stream.streamId}`}
                alt=""
                style={{
                  position: 'absolute',
                  inset: 0,
                  width: '100%',
                  height: '100%',
                }}
              />
              <Box
                display="flex"
                alignItems="center"
                justifyContent="center"
                gap={0.5}
                sx={{
                  position: 'absolute',
                  bottom: 0,
                  right: 0,
                  p: 1,
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
