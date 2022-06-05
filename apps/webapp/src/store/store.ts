import create from 'zustand';
import { persist, subscribeWithSelector } from 'zustand/middleware';
export interface Settings {
  theme: string;
  volume: number;
  setVolume: any;
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
        volume: 0.5,
        setVolume: (volume: number) => set({ volume }),
      }),
      { name: 'settings' },
    ),
  ),
);
