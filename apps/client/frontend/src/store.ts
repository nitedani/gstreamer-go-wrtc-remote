import create from 'zustand';
import { devtools } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';
import type { PartialDeep } from 'type-fest';
import _ from 'lodash';
export interface Configuration {
  colorScheme: string;
  language: string;
}

export interface StoreState {
  configuration: Configuration;
}

export const useStore = create<StoreState>()(
  immer(
    devtools((set) => ({
      configuration: {
        colorScheme: 'blue',
        language: 'en',
      },
    })),
  ),
);
export const setState = (
  newState:
    | PartialDeep<StoreState>
    | ((state: StoreState) => PartialDeep<StoreState>),
) =>
  useStore.setState((state) => {
    if (typeof newState === 'function') {
      Object.assign(state, newState(state));
    } else {
      Object.assign(state, _.mergeWith({}, useStore.getState(), newState));
    }
  });
