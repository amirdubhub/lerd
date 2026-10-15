import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';

const { loadGitChanges, loadGitPatch, closeModal } = vi.hoisted(() => ({
  loadGitChanges: vi.fn(),
  loadGitPatch: vi.fn(),
  closeModal: vi.fn()
}));
vi.mock('$lib/gitDiff', async (orig) => ({ ...(await orig<object>()), loadGitChanges, loadGitPatch }));
vi.mock('$stores/modals', () => ({ closeModal }));

import GitDiffModal from './GitDiffModal.svelte';

const props = { domain: 'acme.test', path: '/srv/acme', branch: 'main' };

describe('GitDiffModal', () => {
  beforeEach(() => {
    loadGitChanges.mockReset();
    loadGitPatch.mockReset();
  });

  it('opens the first changed file and loads another on click', async () => {
    loadGitChanges.mockResolvedValue({
      staged: [],
      unstaged: [{ path: 'app.php', status: 'M' }],
      untracked: [{ path: 'notes.md', status: '?' }]
    });
    loadGitPatch.mockImplementation(async (_d: string, _p: string, section: string, file: string) => ({
      patch: `diff --git a/${file} b/${file}\n@@ -1 +1 @@\n-old ${section}\n+new ${file}\n`,
      truncated: false
    }));
    render(GitDiffModal, { props });

    expect(await screen.findByText('new app.php')).toBeTruthy();
    expect(loadGitPatch).toHaveBeenCalledWith('acme.test', '/srv/acme', 'unstaged', 'app.php', false);
    expect(screen.queryByText('Staged · 0')).toBeNull();

    await fireEvent.click(screen.getByText('notes.md'));
    expect(await screen.findByText('new notes.md')).toBeTruthy();
    expect(loadGitPatch).toHaveBeenLastCalledWith('acme.test', '/srv/acme', 'untracked', 'notes.md', false);
  });

  it('reloads the open file without whitespace changes and says when nothing else is left', async () => {
    loadGitChanges.mockResolvedValue({ staged: [], unstaged: [{ path: 'c.yaml', status: 'M' }], untracked: [] });
    loadGitPatch.mockImplementation(async (...args: unknown[]) => ({
      patch: args[4] ? '' : '@@ -1 +1 @@\n-    - x\n+  - x\n',
      truncated: false
    }));
    render(GitDiffModal, { props });
    // The marker sits in its own column, so the code keeps its exact indentation.
    const code = await screen.findAllByText('- x');
    expect(code.map((el) => el.textContent)).toEqual(['    - x', '  - x']);

    await fireEvent.click(screen.getByRole('switch', { name: 'Hide whitespace' }));
    expect(await screen.findByText('Only whitespace changed in this file.')).toBeTruthy();
    expect(loadGitPatch).toHaveBeenLastCalledWith('acme.test', '/srv/acme', 'unstaged', 'c.yaml', true);
  });

  it('says so when the tree is clean', async () => {
    loadGitChanges.mockResolvedValue({ staged: [], unstaged: [], untracked: [] });
    render(GitDiffModal, { props });
    await waitFor(() => expect(screen.getByText('No uncommitted changes.')).toBeTruthy());
    expect(loadGitPatch).not.toHaveBeenCalled();
  });
});
