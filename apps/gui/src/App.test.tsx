import { act, render } from '@testing-library/react';

import { unmountComponentAtNode } from 'react-dom';
import App from './App';

let container: any = null;
beforeEach(() => {
  // setup a DOM element as a render target
  container = document.createElement('div');
  document.body.appendChild(container);
});

afterEach(() => {
  // cleanup on exiting
  unmountComponentAtNode(container);
  container.remove();
  container = null;
});

it('renders', () => {
  act(() => {
    render(<App />, container);
  });
  expect(container).toBeVisible();
});
