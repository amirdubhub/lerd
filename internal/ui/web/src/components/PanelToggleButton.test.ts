import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import { get } from 'svelte/store';
import PanelToggleButton from './PanelToggleButton.svelte';
import { sidePanelHidden } from '$stores/sidePanel';
import { m } from '../paraglide/messages.js';

describe('PanelToggleButton', () => {
  it('brings the hidden list panel back', async () => {
    sidePanelHidden.set(true);
    const { getByLabelText } = render(PanelToggleButton, { props: { title: m.nav_showList() } });
    await fireEvent.click(getByLabelText(m.nav_showList()));
    expect(get(sidePanelHidden)).toBe(false);
  });
});
