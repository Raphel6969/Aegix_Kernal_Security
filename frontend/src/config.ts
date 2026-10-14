/**
 * Central frontend configuration.
 * All API URLs are driven by VITE_API_URL when provided.
 *
 * To change the backend target, edit frontend/.env:
 *   VITE_API_URL=http://localhost:8000
 */

const envApiUrl = import.meta.env.VITE_API_URL;

// Ensure we only use VITE_API_URL if it is non-empty and non-whitespace.
// Otherwise fallback to localhost in dev (port 5173) or window.location.origin in production.
const rawBase =
  (typeof envApiUrl === 'string' && envApiUrl.trim().length > 0)
    ? envApiUrl.trim()
    : (window.location.port === '5173'
      ? 'http://localhost:8000/api'
      : window.location.origin + '/api');

// Strip trailing slash for consistency (always an absolute URL like http://... or https://...)
export const API_URL = rawBase.replace(/\/$/, '');

// WebSocket URL derived from the same base: replace http(s) with ws(s) and point to /ws
export const WS_URL = (() => {
  const baseWithoutApi = API_URL.replace(/\/api$/, '');
  if (baseWithoutApi.startsWith('http://') || baseWithoutApi.startsWith('https://')) {
    return baseWithoutApi.replace(/^http/, 'ws') + '/ws';
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/ws`;
})();
