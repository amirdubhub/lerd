<script lang="ts">
  import { onMount } from 'svelte';
  import Modal from '$components/Modal.svelte';
  import Toggle from '$components/Toggle.svelte';
  import { closeModal } from '$stores/modals';
  import {
    SECTIONS,
    diffLines,
    diffStats,
    loadGitChanges,
    loadGitPatch,
    type GitChanges,
    type GitDiffSection,
    type GitPatch,
    type DiffLineKind
  } from '$lib/gitDiff';
  import { m } from '../paraglide/messages.js';

  interface Props {
    domain: string;
    path: string;
    branch: string;
  }
  let { domain, path, branch }: Props = $props();

  let changes = $state<GitChanges | null>(null);
  let selected = $state<{ section: GitDiffSection; file: string } | null>(null);
  let patch = $state<GitPatch | null>(null);
  let error = $state('');
  let ignoreSpace = $state(false);

  const sectionLabel: Record<GitDiffSection, () => string> = {
    staged: m.gitDiff_staged,
    unstaged: m.gitDiff_unstaged,
    untracked: m.gitDiff_untracked
  };
  const lineClass: Record<DiffLineKind, string> = {
    add: 'bg-green-500/10 text-green-700 dark:text-green-300',
    del: 'bg-red-500/10 text-red-700 dark:text-red-300',
    hunk: 'text-sky-600 dark:text-sky-400',
    meta: 'text-gray-500 dark:text-gray-400 italic',
    context: 'text-gray-700 dark:text-gray-300'
  };
  const statusClass: Record<string, string> = {
    A: 'text-green-600 dark:text-green-400',
    '?': 'text-green-600 dark:text-green-400',
    D: 'text-red-600 dark:text-red-400',
    U: 'text-red-600 dark:text-red-400'
  };

  const empty = $derived(!!changes && SECTIONS.every((s) => changes![s].length === 0));
  const lines = $derived(patch ? diffLines(patch.patch) : []);
  const stats = $derived(diffStats(lines));

  onMount(async () => {
    try {
      changes = await loadGitChanges(domain, path);
      const first = SECTIONS.find((s) => changes![s].length > 0);
      if (first) select(first, changes[first][0].path);
    } catch (e) {
      error = e instanceof Error ? e.message : m.common_failed();
    }
  });

  async function select(section: GitDiffSection, file: string) {
    selected = { section, file };
    patch = null;
    error = '';
    try {
      const ignoring = ignoreSpace;
      const p = await loadGitPatch(domain, path, section, file, ignoring);
      // A slower answer for an earlier click must not replace this one.
      if (selected?.section === section && selected.file === file && ignoreSpace === ignoring) patch = p;
    } catch (e) {
      error = e instanceof Error ? e.message : m.common_failed();
    }
  }

  function toggleSpace() {
    ignoreSpace = !ignoreSpace;
    if (selected) select(selected.section, selected.file);
  }
</script>

<Modal open size="2xl" title={m.gitDiff_title({ branch })} onclose={closeModal}>
  {#if empty}
    <p class="px-5 py-6 text-sm text-gray-500 dark:text-gray-400">{m.gitDiff_empty()}</p>
  {:else}
    <div class="flex flex-col md:flex-row h-[75vh] min-h-0">
      <nav class="md:w-72 shrink-0 max-h-48 md:max-h-none overflow-y-auto border-b md:border-b-0 md:border-r border-gray-100 dark:border-lerd-border py-2">
        {#each SECTIONS as section (section)}
          {#if changes && changes[section].length > 0}
            <h4 class="px-4 pt-2 pb-1 text-[11px] font-semibold uppercase tracking-wide text-gray-400">
              {sectionLabel[section]()} · {changes[section].length}
            </h4>
            {#each changes[section] as f (f.path)}
              {@const active = selected?.section === section && selected.file === f.path}
              <button
                type="button"
                onclick={() => select(section, f.path)}
                aria-current={active ? 'true' : undefined}
                class="w-full flex items-center gap-2 px-4 py-1 text-left font-mono text-xs {active
                  ? 'bg-gray-100 dark:bg-white/10 text-gray-900 dark:text-white'
                  : 'text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-white/5'}"
              >
                <span class="w-3 shrink-0 font-semibold {statusClass[f.status] ?? 'text-amber-600 dark:text-amber-400'}">{f.status}</span>
                <span class="truncate" title={f.path}>{f.path}</span>
              </button>
            {/each}
          {/if}
        {/each}
      </nav>
      <div class="flex-1 min-w-0 min-h-0 overflow-auto">
        {#if error}
          <p class="px-5 py-4 text-sm text-red-600 dark:text-red-400">{error}</p>
        {:else if patch && selected}
          <div class="sticky top-0 z-10 flex items-center gap-3 px-4 py-2 border-b border-gray-100 dark:border-lerd-border bg-white dark:bg-lerd-card font-mono text-xs">
            <span class="truncate text-gray-800 dark:text-gray-100" title={selected.file}>{selected.file}</span>
            <span class="ml-auto shrink-0 tabular-nums text-green-600 dark:text-green-400">+{stats.added}</span>
            <span class="shrink-0 tabular-nums text-red-600 dark:text-red-400">−{stats.removed}</span>
            <span class="shrink-0 flex items-center gap-2 pl-3 border-l border-gray-200 dark:border-lerd-border font-sans text-gray-500 dark:text-gray-400">
              {m.gitDiff_ignoreSpace()}
              <Toggle on={ignoreSpace} title={m.gitDiff_ignoreSpace()} onclick={toggleSpace} />
            </span>
          </div>
          {#if patch.truncated}
            <p class="px-4 py-2 text-xs text-amber-700 dark:text-amber-300 bg-amber-500/10">{m.gitDiff_truncated()}</p>
          {/if}
          {#if lines.length === 0 && ignoreSpace}
            <p class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">{m.gitDiff_onlyWhitespace()}</p>
          {/if}
          <div class="py-1 font-mono text-xs leading-5 min-w-max">
            {#each lines as l, i (i)}
              {@const code = l.kind === 'add' || l.kind === 'del' || l.kind === 'context'}
              <div class="flex {lineClass[l.kind]}">
                <span class="w-12 shrink-0 pr-2 text-right select-none tabular-nums text-gray-400 dark:text-gray-500">{l.old ?? ''}</span>
                <span class="w-12 shrink-0 pr-2 text-right select-none tabular-nums text-gray-400 dark:text-gray-500">{l.new ?? ''}</span>
                <!-- The +/- gets its own column so changed indentation stays visible as indentation. -->
                <span class="w-5 shrink-0 text-center select-none">{code ? l.text.slice(0, 1).trim() : ''}</span>
                <span class="pr-4 whitespace-pre">{(code ? l.text.slice(1) : l.text) || ' '}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</Modal>
