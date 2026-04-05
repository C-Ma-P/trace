import { Call } from '@wailsio/runtime';

export {};

declare global {
  interface Window {
    wails?: {
      Call?: {
        ByName?: <T = any>(methodName: string, ...args: any[]) => Promise<T>;
      };
    };
  }
}

export async function callByName<T>(methodName: string, ...args: any[]): Promise<T> {
  if (Call?.ByName) {
    return Call.ByName(methodName, ...args) as Promise<T>;
  }

  const byName = window.wails?.Call?.ByName;
  if (typeof byName === 'function') {
    return byName<T>(methodName, ...args);
  }

  throw new Error('Wails runtime not available: Call.ByName is missing');
}
