import { webApi } from './api-web';
import { wailsApi } from './api-wails';

const hasWailsBridge = typeof window !== 'undefined'
  && Boolean(window.go?.main?.App);

export const api = hasWailsBridge ? wailsApi : webApi;
