import { useState } from 'react';
import { Window } from '@wailsio/runtime';

export interface WindowControlActions {
  alwaysOnTop: boolean;
  toggleAlwaysOnTop: () => Promise<void>;
  minimise: () => Promise<void>;
  toggleMaximise: () => Promise<void>;
  close: () => Promise<void>;
}

export const useWindowControls = (): WindowControlActions => {
  const [alwaysOnTop, setAlwaysOnTop] = useState(false);

  return {
    alwaysOnTop,
    toggleAlwaysOnTop: async () => {
      const next = !alwaysOnTop;
      await Window.SetAlwaysOnTop(next);
      setAlwaysOnTop(next);
    },
    minimise: async () => {
      await Window.Minimise();
    },
    toggleMaximise: async () => {
      const isMax = await Window.IsMaximised();
      if (isMax) {
        await Window.UnMaximise();
      } else {
        await Window.Maximise();
      }
    },
    close: async () => {
      await Window.Close();
    },
  };
};
