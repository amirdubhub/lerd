import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

describe('sidePanelHidden', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.resetModules();
  });

  it('shows the panel with no stored value', async () => {
    const { sidePanelHidden } = await import('./sidePanel');
    expect(get(sidePanelHidden)).toBe(false);
  });

  it('restores a hidden panel after a reload', async () => {
    localStorage.setItem('lerd:sidePanelHidden', 'true');
    const { sidePanelHidden } = await import('./sidePanel');
    expect(get(sidePanelHidden)).toBe(true);
  });

  it('persists the toggle', async () => {
    const { sidePanelHidden, toggleSidePanel } = await import('./sidePanel');
    toggleSidePanel();
    expect(get(sidePanelHidden)).toBe(true);
    expect(localStorage.getItem('lerd:sidePanelHidden')).toBe('true');
  });
});
