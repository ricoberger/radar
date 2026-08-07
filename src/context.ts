import { createContext, useContext } from 'react';

import { Config } from './types.js';

export interface AppContextValue {
  config: Config;
  runExternal: (command: string, args: string[]) => void;
}

export const AppContext = createContext<AppContextValue>(
  null as unknown as AppContextValue,
);

export function useAppContext(): AppContextValue {
  return useContext(AppContext);
}
