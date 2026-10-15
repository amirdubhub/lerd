import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import Harness from './StripeControl.test.svelte';

describe('StripeControl', () => {
  it('renders the Stripe toggle and the config gear', () => {
    const { container } = render(Harness, { props: {} });
    expect(screen.getByRole('button', { name: 'Stripe' })).toBeInTheDocument();
    // Toggle + gear, even while stopped, so the path can be set before enabling.
    expect(container.querySelectorAll('button')).toHaveLength(2);
  });

  it('forwards the toggle click', () => {
    const onToggle = vi.fn();
    render(Harness, { props: { onToggle } });
    screen.getByRole('button', { name: 'Stripe' }).click();
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it('opens the modal prefilled with the current webhook path', async () => {
    render(Harness, { props: { webhookPath: '/webhooks/stripe' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Configure Stripe listener' }));
    const input = screen.getByLabelText('Webhook path') as HTMLInputElement;
    expect(input.value).toBe('/webhooks/stripe');
  });

  it('normalises a leading slash and forwards the saved path', async () => {
    const onSaveConfig = vi.fn().mockResolvedValue({ ok: true });
    render(Harness, { props: { onSaveConfig } });
    await fireEvent.click(screen.getByRole('button', { name: 'Configure Stripe listener' }));
    const input = screen.getByLabelText('Webhook path') as HTMLInputElement;
    await fireEvent.input(input, { target: { value: 'nest/stripe' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(onSaveConfig).toHaveBeenCalledWith('/nest/stripe', 'STRIPE_SECRET');
  });

  it('prefills the secret env key with the detected one and forwards edits', async () => {
    const onSaveConfig = vi.fn().mockResolvedValue({ ok: true });
    render(Harness, { props: { secretEnvKey: 'STRIPE_SECRET_KEY', onSaveConfig } });
    await fireEvent.click(screen.getByRole('button', { name: 'Configure Stripe listener' }));
    const input = screen.getByLabelText('Secret env key') as HTMLInputElement;
    expect(input.value).toBe('STRIPE_SECRET_KEY');
    await fireEvent.input(input, { target: { value: ' STRIPE_API_KEY ' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(onSaveConfig).toHaveBeenCalledWith('/stripe/webhook', 'STRIPE_API_KEY');
  });

  it('keeps the modal open and shows the error when the save is refused', async () => {
    const error = 'STRIPE_SECRET has no value in .env';
    const onSaveConfig = vi.fn().mockResolvedValue({ ok: false, error });
    render(Harness, { props: { onSaveConfig } });
    await fireEvent.click(screen.getByRole('button', { name: 'Configure Stripe listener' }));
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(await screen.findByText(error)).toBeInTheDocument();
    expect(screen.getByLabelText('Secret env key')).toBeInTheDocument();
  });

  it('ignores a save that resolves after the modal was closed and reopened', async () => {
    let resolve: (r: { ok: boolean }) => void = () => {};
    const onSaveConfig = vi.fn(() => new Promise<{ ok: boolean }>((r) => (resolve = r)));
    render(Harness, { props: { onSaveConfig } });
    const gear = screen.getByRole('button', { name: 'Configure Stripe listener' });
    await fireEvent.click(gear);
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }));
    await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    await fireEvent.click(gear);
    resolve({ ok: true });
    await new Promise((r) => setTimeout(r));
    expect(screen.getByLabelText('Secret env key')).toBeInTheDocument();
  });
});
