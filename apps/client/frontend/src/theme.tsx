import {
  ChakraProvider,
  ChakraProviderProps,
  ColorModeScript,
  extendTheme,
  ThemeConfig,
  withDefaultColorScheme,
} from '@chakra-ui/react';
import { mode } from '@chakra-ui/theme-tools';
import { useMemo } from 'react';
import { useStore } from './store';

const config: ThemeConfig = {
  initialColorMode: 'dark',
  useSystemColorMode: true,
};

const styles = {
  global: (props: any) => ({
    body: {
      color: mode('gray.800', 'whiteAlpha.900')(props),
      bg: mode('white', '#161616')(props),
    },
  }),
};

const components = {
  Drawer: {
    // setup light/dark mode component defaults
    baseStyle: (props: any) => ({
      dialog: {
        bg: mode('white', '#161616')(props),
      },
    }),
  },
  Modal: {
    // setup light/dark mode component defaults
    baseStyle: (props: any) => ({
      dialog: {
        bg: mode('white', '#161616')(props),
      },
    }),
  },
};

const theme = extendTheme({
  components,
  styles,
  config,
});

export const MyChakraProvider: React.FC<ChakraProviderProps> = ({
  children,
}) => {
  const {
    configuration: { colorScheme },
  } = useStore();

  const _theme = useMemo(
    () => extendTheme(theme, withDefaultColorScheme({ colorScheme })),
    [colorScheme],
  );

  return (
    <ChakraProvider theme={_theme}>
      <ColorModeScript initialColorMode={_theme.config.initialColorMode} />
      {children}
    </ChakraProvider>
  );
};
