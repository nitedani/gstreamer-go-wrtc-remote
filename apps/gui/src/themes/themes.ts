import { blue } from '@mui/material/colors';
import { createTheme, Theme } from '@mui/material/styles';
import { useStore } from 'src/store/store';

export const lightTheme = createTheme({
  typography: {
    fontFamily:
      '-apple-system,system-ui,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Fira Sans,Ubuntu,Oxygen,Oxygen Sans,Cantarell,Droid Sans,Apple Color Emoji,Segoe UI Emoji,Segoe UI Emoji,Segoe UI Symbol,Lucida Grande,Helvetica,Arial,sans-serif',
  },
  palette: {
    mode: 'light',
    primary: { main: '#f76002' },
    background: {
      default: '#f5f5f5',
      paper: '#fff',
    },
    text: {
      primary: '#030303',
    },
  },
});
export const darkTheme = createTheme({
  typography: {
    fontFamily:
      '-apple-system,system-ui,BlinkMacSystemFont,Segoe UI,Roboto,Helvetica Neue,Fira Sans,Ubuntu,Oxygen,Oxygen Sans,Cantarell,Droid Sans,Apple Color Emoji,Segoe UI Emoji,Segoe UI Emoji,Segoe UI Symbol,Lucida Grande,Helvetica,Arial,sans-serif',
  },
  palette: {
    mode: 'dark',
    primary: blue,
    background: {
      paper: '#202020',
      default: '#181818',
    },
    text: {
      primary: '#fff',
    },
  },
});

export const useTheme = (): [Theme, 'light' | 'dark'] => {
  const { theme } = useStore();
  switch (theme) {
    case 'light':
      return [lightTheme, theme];
    case 'dark':
      return [darkTheme, theme];
    default:
      return [lightTheme, theme];
  }
};
