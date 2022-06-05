import create from 'zustand';
import { persist, subscribeWithSelector } from 'zustand/middleware';
export interface Settings {
  theme: string;
}

export const useStore = create<
  Settings,
  [
    ['zustand/subscribeWithSelector', never],
    ['zustand/persist', Partial<Settings>],
  ]
>(
  subscribeWithSelector(
    persist(
      (set) => ({
        theme: 'dark',
      }),
      { name: 'settings' },
    ),
  ),
);
