import React from 'react';
import { Routes } from './router/Router';
import { QueryClient, QueryClientProvider } from 'react-query';
import { useTheme } from './themes/themes';
import { CssBaseline, ThemeProvider } from '@mui/material';

const queryClient = new QueryClient();

const MyThemeProvider: React.FC = ({ children }) => {
  const [theme] = useTheme();
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {children}
    </ThemeProvider>
  );
};

const App = () => (
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <MyThemeProvider>
        <Routes />
      </MyThemeProvider>
    </QueryClientProvider>
  </React.StrictMode>
);

export default App;
