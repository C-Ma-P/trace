import { callByName } from './wails';

const serviceName = 'main.WindowService';

function call<T>(method: string, ...args: any[]): Promise<T> {
  return callByName(`${serviceName}.${method}`, ...args);
}

export function openProjectWindow(projectId: string): Promise<void> {
  return call('OpenProjectWindow', projectId);
}

export function openProjectWindowKeepLauncher(projectId: string): Promise<void> {
  return call('OpenProjectWindowKeepLauncher', projectId);
}

export function listOpenProjectIDs(): Promise<string[]> {
  return call('ListOpenProjectIDs');
}

export function pickDirectory(startDir = ''): Promise<string> {
  return call('PickDirectory', startDir);
}

export function pickAssetFile(): Promise<string> {
  return call('PickAssetFile');
}

export function pickAssetDir(): Promise<string> {
  return call('PickAssetDir');
}

export function setLauncherView(view: 'launcher' | 'kicad-import'): Promise<void> {
  return call('SetLauncherView', view);
}
