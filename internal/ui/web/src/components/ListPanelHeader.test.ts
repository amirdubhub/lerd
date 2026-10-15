import { render, screen, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { sidePanelHidden } from '$stores/sidePanel';
import { m } from '../paraglide/messages.js';
import { describe, it, expect } from 'vitest';
import Harness from './ListPanelHeader.test.svelte';

describe('ListPanelHeader', () => {
  it('renders title uppercased via class', () => {
    render(Harness, { props: { title: 'Sites' } });
    const el = screen.getByText('Sites');
    expect(el.className).toMatch(/uppercase/);
  });

  it('renders actions snippet when provided', () => {
    render(Harness, { props: { title: 'Sites', withActions: true } });
    expect(screen.getByTitle('act')).toBeInTheDocument();
  });

  it('omits actions container when no snippet', () => {
    render(Harness, { props: { title: 'Sites' } });
    expect(screen.queryByTitle('act')).not.toBeInTheDocument();
  });

  it('hides the list panel from its own header', async () => {
    sidePanelHidden.set(false);
    render(Harness, { props: { title: 'Sites' } });
    await fireEvent.click(screen.getByLabelText(m.nav_hideList()));
    expect(get(sidePanelHidden)).toBe(true);
  });
});
