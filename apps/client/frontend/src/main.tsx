import React from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import './style.css';
import { MyChakraProvider } from './theme';

const container = document.getElementById('root');

// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <MyChakraProvider>
      <App />
    </MyChakraProvider>
  </React.StrictMode>,
);
