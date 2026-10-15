<script lang="ts">
  import { onMount } from 'svelte';
  import StatusDot from '$components/StatusDot.svelte';
  import Icon from '$components/Icon.svelte';
  import DashboardHeader from '$components/DashboardHeader.svelte';
  import DashboardSection from '$components/DashboardSection.svelte';
  import InstalledServiceTile from './InstalledServiceTile.svelte';
  import PresetCard from './PresetCard.svelte';
  import UpdateAllModal from './UpdateAllModal.svelte';
  import {
    coreServices,
    servicesLoaded,
    serviceLabel,
    type Service,
    type ServiceUpdateResult
  } from '$stores/services';
  import { discoverablePresets, presetsLoaded, loadPresets } from '$stores/presets';
  import { openPresetModal } from '$stores/modals';
  import { accessMode } from '$stores/accessMode';
  import { groupByCategory, CATEGORY_LABELS } from '$lib/presetCategories';
  import { m } from '../../paraglide/messages.js';

  onMount(() => {
    loadPresets();
  });

  const running = $derived($coreServices.filter((s) => s.status === 'active').length);
  const total = $derived($coreServices.length);
  const outdated = $derived($coreServices.filter((s) => s.update_available));
  const updates = $derived(outdated.length);
  // Services waiting on an update lead the grid so they are not lost among the rest.
  const installed = $derived([...outdated, ...$coreServices.filter((s) => !s.update_available)]);

  // A snapshot of what was outdated when the dialog opened, null while it is closed.
  let updateBatch = $state<Service[] | null>(null);
  let results = $state<ServiceUpdateResult[]>([]);
  const updated = $derived(results.filter((r) => r.ok).length);
  const failed = $derived(results.filter((r) => !r.ok));

  function openUpdateAll() {
    results = [];
    updateBatch = [...outdated];
  }
  const sitesServed = $derived(
    new Set($coreServices.flatMap((s) => s.site_domains ?? [])).size
  );

  const groups = $derived(groupByCategory($discoverablePresets));
</script>

<div class="flex-1 flex flex-col min-h-0">
  <DashboardHeader title={m.nav_services()} stats={$servicesLoaded && total > 0 ? summary : undefined} />

  <div class="flex-1 min-h-0 overflow-y-auto p-4 space-y-6">
    <section class="space-y-2.5">
      <div class="flex items-center justify-between gap-3">
        <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.services_dash_installed()}</h2>
        {#if $accessMode.localControl && updates > 0}
          <button
            onclick={openUpdateAll}
            disabled={updateBatch !== null}
            class="inline-flex items-center gap-1 text-xs font-medium text-emerald-600 hover:text-emerald-700 disabled:opacity-50 dark:text-emerald-400 dark:hover:text-emerald-300"
          >
            <span>↑</span>
            {m.services_updateAll()}
          </button>
        {/if}
      </div>
      {#if results.length > 0}
        <div class="text-xs space-y-0.5">
          {#if updated > 0}
            <p class="text-emerald-600 dark:text-emerald-400">{m.services_updateAll_done({ count: updated })}</p>
          {/if}
          {#each failed as r (r.name)}
            <p class="text-red-600 dark:text-red-400">{serviceLabel(r.name)}: {r.error}</p>
          {/each}
        </div>
      {/if}
      {#if $servicesLoaded && total === 0}
        <p class="text-sm text-gray-500 dark:text-gray-400">{m.services_dash_noInstalled()}</p>
      {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-2">
          {#each installed as svc (svc.name)}
            <InstalledServiceTile {svc} />
          {/each}
        </div>
      {/if}
    </section>

    {#if $accessMode.localControl}
      <section class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.services_dash_discover()}</h2>
          <button
            onclick={openPresetModal}
            class="inline-flex items-center gap-1 text-xs font-medium text-lerd-red hover:text-lerd-redhov"
          >
            <Icon name="plus" class="w-3.5 h-3.5" />
            {m.services_addPreset()}
          </button>
        </div>
        {#if !$presetsLoaded}
          <p class="text-xs text-gray-400">{m.common_loading()}</p>
        {:else if groups.length === 0}
          <p class="text-sm text-gray-500 dark:text-gray-400">{m.services_preset_allInstalled()}</p>
        {:else}
          <div class="space-y-4">
            {#each groups as group (group.key)}
              <DashboardSection label={CATEGORY_LABELS[group.key]()} sub>
                {#each group.items as preset (preset.name)}
                  <PresetCard {preset} category={group.key} />
                {/each}
              </DashboardSection>
            {/each}
          </div>
        {/if}
      </section>
    {/if}
  </div>
</div>

{#if updateBatch}
  <UpdateAllModal
    services={updateBatch}
    onclose={() => (updateBatch = null)}
    ondone={(r) => (results = r)}
  />
{/if}

{#snippet summary()}
  <span class="inline-flex items-center gap-1.5">
    <StatusDot color={running > 0 ? 'green' : 'gray'} />
    {m.dashboard_services_summary({ running, total })}
  </span>
  {#if updates > 0}
    <span class="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
      <span>↑</span>
      {m.dashboard_services_updates({ count: updates })}
    </span>
  {/if}
  {#if sitesServed > 0}
    <span class="inline-flex items-center gap-1.5">
      <Icon name="sites" class="w-3.5 h-3.5" />
      {m.services_dash_sitesServed({ count: sitesServed })}
    </span>
  {/if}
{/snippet}
