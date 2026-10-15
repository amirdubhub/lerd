import { writable } from 'svelte/store';

// Whether the list panel beside the rail is hidden, so a half-width window can
// give its room to the detail. A view preference, kept in the browser.
const KEY = 'lerd:sidePanelHidden';

function initial(): boolean {
  try {
    return localStorage.getItem(KEY) === 'true';
  } catch {
    return false;
  }
}

export const sidePanelHidden = writable<boolean>(initial());

sidePanelHidden.subscribe((v) => {
  try {
    localStorage.setItem(KEY, String(v));
  } catch {
    // private mode / storage disabled — fall back to in-memory only.
  }
});

export function toggleSidePanel() {
  sidePanelHidden.update((v) => !v);
}
