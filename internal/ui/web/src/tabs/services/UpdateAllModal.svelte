<script lang="ts">
  import ConfirmModal from '$components/ConfirmModal.svelte';
  import ServiceIcon from '$components/ServiceIcon.svelte';
  import Toggle from '$components/Toggle.svelte';
  import {
    serviceLabel,
    updateServices,
    updateProgress,
    type Service,
    type ServiceUpdateResult
  } from '$stores/services';
  import { m } from '../../paraglide/messages.js';

  interface Props {
    // The services that had an update when the dialog opened. Kept fixed so a
    // row does not vanish the moment its own update clears the badge.
    services: Service[];
    onclose: () => void;
    ondone: (results: ServiceUpdateResult[]) => void;
  }
  let { services, onclose, ondone }: Props = $props();

  let on = $state<Record<string, boolean>>(Object.fromEntries(services.map((s) => [s.name, true])));
  let updating = $state(false);
  // Live progress is cleared once a service's update ends, so its outcome is
  // kept here for the row to show while the rest of the batch runs.
  let outcome = $state<Record<string, ServiceUpdateResult>>({});
  const selected = $derived(services.filter((s) => on[s.name]).map((s) => s.name));

  function target(s: Service): string {
    if (!s.latest_version || s.latest_version === s.version) return s.version ?? '';
    return s.version ? s.version + ' → ' + s.latest_version : '→ ' + s.latest_version;
  }

  async function run() {
    updating = true;
    try {
      ondone(await updateServices(selected, (r) => (outcome[r.name] = r)));
    } finally {
      updating = false;
      onclose();
    }
  }
</script>

<ConfirmModal
  open
  title={m.services_updateAll()}
  body={m.services_updateAll_body()}
  confirmLabel={m.services_update()}
  loading={updating}
  disabled={selected.length === 0}
  onconfirm={run}
  {onclose}
>
  {#snippet extra()}
    <div class="max-h-[50vh] overflow-y-auto rounded-lg border border-gray-200 dark:border-lerd-border divide-y divide-gray-100 dark:divide-lerd-border">
      {#each services as s (s.name)}
        {@const progress = $updateProgress[s.name]}
        {@const result = outcome[s.name]}
        <div class="flex items-center gap-3 px-3 py-2.5">
          <ServiceIcon name={s.name} category={s.category} icon={s.icon} color={s.color} preset={s.preset} compact />
          <div class="flex-1 min-w-0">
            <div class="text-sm text-gray-800 dark:text-gray-100 truncate">{serviceLabel(s.name)}</div>
            {#if progress}
              <div class="truncate text-xs {progress.error ? 'text-red-500' : 'text-gray-500 dark:text-gray-400'}">{progress.message}</div>
            {:else if result}
              <div class="truncate text-xs {result.ok ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-500'}">
                {result.ok ? m.services_updateAll_rowDone() : result.error}
              </div>
            {:else}
              <div class="truncate font-mono text-xs text-emerald-600 dark:text-emerald-400">{target(s)}</div>
            {/if}
          </div>
          <Toggle on={on[s.name]} title={serviceLabel(s.name)} disabled={updating} onclick={() => (on[s.name] = !on[s.name])} />
        </div>
      {/each}
    </div>
  {/snippet}
</ConfirmModal>
