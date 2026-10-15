<script lang="ts">
  import { untrack } from 'svelte';
  import Modal from '$components/Modal.svelte';
  import { m } from '../../paraglide/messages.js';

  // Edits the per-project Stripe webhook route and secret env key (persisted to
  // .lerd.yaml), prefilled with the current values each time it opens.
  interface Props {
    open: boolean;
    path: string;
    secretEnvKey: string;
    onclose: () => void;
    onsave: (path: string, secretEnvKey: string) => Promise<{ ok: boolean; error?: string }>;
  }
  let { open, path, secretEnvKey, onclose, onsave }: Props = $props();

  let value = $state('');
  let keyValue = $state('');
  let error = $state('');
  // Bumped on every open, so a save answered after a close and reopen is ignored.
  let opening = 0;

  // Seed only on the open transition (path read untracked), so a live sites
  // push that updates path mid-edit doesn't overwrite what the user is typing.
  $effect(() => {
    if (!open) return;
    opening++;
    error = '';
    value = untrack(() => path) || '/stripe/webhook';
    keyValue = untrack(() => secretEnvKey) || 'STRIPE_SECRET';
  });

  async function save() {
    const trimmed = value.trim();
    if (!trimmed) return onclose();
    const started = opening;
    const r = await onsave(trimmed.startsWith('/') ? trimmed : '/' + trimmed, keyValue.trim());
    if (started !== opening || !open) return;
    if (!r.ok) {
      error = r.error || m.common_requestFailed();
      return;
    }
    onclose();
  }
</script>

<Modal {open} {onclose} title={m.sites_controls_stripeConfigTitle()} size="sm">
  <div class="px-5 py-4 space-y-3">
    <label class="block text-sm text-gray-600 dark:text-gray-400" for="stripe-path">
      {m.sites_controls_stripePathLabel()}
    </label>
    <input
      id="stripe-path"
      type="text"
      bind:value
      spellcheck="false"
      autocomplete="off"
      onkeydown={(e) => e.key === 'Enter' && save()}
      class="w-full text-sm font-mono px-2.5 py-1.5 rounded-sm border border-gray-200 dark:border-lerd-border bg-white dark:bg-lerd-card text-gray-800 dark:text-gray-100 focus:outline-none focus:ring-1 focus:ring-lerd-red"
    />
    <p class="text-[11px] text-gray-500 dark:text-gray-400">
      {m.sites_controls_stripePathHint()}
    </p>
    <label class="block text-sm text-gray-600 dark:text-gray-400" for="stripe-secret-key">
      {m.sites_controls_stripeSecretKeyLabel()}
    </label>
    <input
      id="stripe-secret-key"
      type="text"
      bind:value={keyValue}
      spellcheck="false"
      autocomplete="off"
      onkeydown={(e) => e.key === 'Enter' && save()}
      class="w-full text-sm font-mono px-2.5 py-1.5 rounded-sm border border-gray-200 dark:border-lerd-border bg-white dark:bg-lerd-card text-gray-800 dark:text-gray-100 focus:outline-none focus:ring-1 focus:ring-lerd-red"
    />
    <p class="text-[11px] text-gray-500 dark:text-gray-400">
      {m.sites_controls_stripeSecretKeyHint()}
    </p>
    {#if error}
      <pre class="text-[11px] text-red-600 dark:text-red-400 whitespace-pre-wrap break-words bg-red-50 dark:bg-red-900/15 rounded-sm px-2 py-1.5 max-h-32 overflow-y-auto">{error}</pre>
    {/if}
  </div>
  {#snippet footer()}
    <button
      type="button"
      onclick={onclose}
      class="text-xs px-3 py-1.5 rounded-sm border border-gray-200 dark:border-lerd-border text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
    >{m.common_cancel()}</button>
    <button
      type="button"
      onclick={save}
      class="text-xs px-3 py-1.5 rounded-sm bg-lerd-red hover:bg-lerd-redhov text-lerd-onred transition-colors"
    >{m.common_save()}</button>
  {/snippet}
</Modal>
