/// <reference types="svelte" />
/// <reference types="vite/client" />

declare module 'occt-import-js' {
  interface OcctInitOptions {
    locateFile?: (name: string) => string;
  }
  function occtimportjs(options?: OcctInitOptions): Promise<any>;
  export default occtimportjs;
}