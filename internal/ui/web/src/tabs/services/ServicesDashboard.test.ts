import { render, fireEvent, within } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import ServicesDashboard from './ServicesDashboard.svelte';
import { services, type Service } from '$stores/services';
import { accessMode } from '$stores/accessMode';

const updateServices = vi.fn();
vi.mock('$stores/services', async (importOriginal) => {
  const actual = await importOriginal<typeof import('$stores/services')>();
  return { ...actual, updateServices: (...args: unknown[]) => updateServices(...args) };
});
vi.mock('$stores/presets', async (importOriginal) => {
  const actual = await importOriginal<typeof import('$stores/presets')>();
  return { ...actual, loadPresets: vi.fn() };
});

function svc(over: Partial<Service> & { name: string }): Service {
  return { status: 'active', site_count: 0, ...over };
}

describe('ServicesDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    accessMode.set({ localControl: true } as never);
    services.set([
      svc({ name: 'mysql', version: 'v8.4' }),
      svc({ name: 'mailpit', version: 'v1.27', update_available: true, latest_version: 'v1.28' }),
      svc({ name: 'redis', version: 'v7', update_available: true, latest_version: 'v7.4' }),
      svc({ name: 'rustfs', version: 'latest', update_available: true, latest_version: 'latest' })
    ]);
  });

  it('keeps the header out of the scrolling area', () => {
    const { container } = render(ServicesDashboard);
    const header = container.querySelector('.page-header')!;
    expect(header.closest('.overflow-y-auto')).toBeNull();
    expect(container.querySelector('.overflow-y-auto')).not.toBeNull();
  });

  it('offers no update all when nothing has an update', () => {
    services.set([svc({ name: 'mysql' })]);
    const { queryByText } = render(ServicesDashboard);
    expect(queryByText('Update all')).toBeNull();
  });

  it('offers no update all without local control', () => {
    accessMode.set({ localControl: false } as never);
    const { queryByText } = render(ServicesDashboard);
    expect(queryByText('Update all')).toBeNull();
  });

  // Services waiting on an update lead the grid so they are not lost among
  // the up-to-date ones.
  it('lists services with an update first', () => {
    const { container } = render(ServicesDashboard);
    const names = [...container.querySelectorAll('.grid .font-semibold')].map((n) => n.textContent);
    expect(names.slice(0, 3)).toEqual(['Mailpit', 'Redis', 'RustFS']);
  });

  it('confirms the batch, updates only the outdated services and reports each failure', async () => {
    updateServices.mockResolvedValue([
      { name: 'mailpit', ok: true },
      { name: 'redis', ok: false, error: 'pull failed' },
      { name: 'rustfs', ok: true }
    ]);
    const { getByText, getByRole, findByText } = render(ServicesDashboard);

    await fireEvent.click(getByText('Update all'));
    expect(getByText('v1.27 → v1.28')).toBeTruthy();
    expect(getByText('v7 → v7.4')).toBeTruthy();
    expect(updateServices).not.toHaveBeenCalled();

    await fireEvent.click(getByRole('button', { name: 'Update' }));
    expect(updateServices).toHaveBeenCalledWith(['mailpit', 'redis', 'rustfs'], expect.any(Function));
    expect(await findByText('2 updated')).toBeTruthy();
    expect(getByText('Redis: pull failed')).toBeTruthy();
  });

  // Every outdated service starts switched on, even when there is only one,
  // and the user can leave any of them out of the batch.
  it('updates only the services left switched on', async () => {
    updateServices.mockResolvedValue([]);
    const { getByText, getByRole, getAllByRole } = render(ServicesDashboard);

    await fireEvent.click(getByText('Update all'));
    const switches = getAllByRole('switch');
    expect(switches.map((s) => s.getAttribute('aria-label'))).toEqual(['Mailpit', 'Redis', 'RustFS']);
    expect(switches.every((s) => s.getAttribute('aria-checked') === 'true')).toBe(true);

    await fireEvent.click(getByRole('switch', { name: 'Redis' }));
    await fireEvent.click(getByRole('button', { name: 'Update' }));
    expect(updateServices).toHaveBeenCalledWith(['mailpit', 'rustfs'], expect.any(Function));
  });

  it('holds the update back while every service is switched off', async () => {
    services.set([svc({ name: 'mailpit', version: 'latest', update_available: true, latest_version: 'latest' })]);
    const { getByText, getByRole } = render(ServicesDashboard);

    await fireEvent.click(getByText('Update all'));
    expect(getByRole('switch', { name: 'Mailpit' }).getAttribute('aria-checked')).toBe('true');
    await fireEvent.click(getByRole('switch', { name: 'Mailpit' }));
    expect((getByRole('button', { name: 'Update' }) as HTMLButtonElement).disabled).toBe(true);
  });

  // A row's live progress is cleared once its update ends, so while the rest
  // of the batch is still pulling the row has to keep its own outcome.
  it('keeps each finished row on its outcome while the batch runs on', async () => {
    updateServices.mockImplementation((_names: string[], onresult: (r: unknown) => void) => {
      onresult({ name: 'mailpit', ok: true });
      onresult({ name: 'redis', ok: false, error: 'pull failed' });
      return new Promise(() => {});
    });
    const { getByText, getByRole, findByText } = render(ServicesDashboard);

    await fireEvent.click(getByText('Update all'));
    await fireEvent.click(getByRole('button', { name: 'Update' }));
    expect(await findByText('Updated')).toBeTruthy();
    expect(getByText('pull failed')).toBeTruthy();
    expect(within(getByRole('dialog')).getByText('latest')).toBeTruthy();
  });
});
