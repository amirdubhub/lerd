import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import SidePanel from './SidePanel.test.svelte';

describe('SidePanel', () => {
  it('slides shut and takes itself out of the tab order while hidden', async () => {
    const { getByText, rerender } = render(SidePanel, { props: { hidden: false } });
    const panel = getByText('list').parentElement!.parentElement!;
    expect(panel.className).toContain('w-56');
    expect(panel.inert).toBe(false);
    await rerender({ hidden: true });
    expect(panel.className).toContain('w-0');
    // Slid shut but still in the page, so keyboard focus must not reach it.
    expect(panel.inert).toBe(true);
  });
});
