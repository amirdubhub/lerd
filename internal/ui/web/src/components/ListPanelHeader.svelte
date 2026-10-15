<script lang="ts">
  import type { Snippet } from 'svelte';
  import PanelToggleButton from './PanelToggleButton.svelte';
  import { m } from '../paraglide/messages.js';

  interface Props {
    title: string;
    actions?: Snippet;
  }
  let { title, actions }: Props = $props();
</script>

<div
  class="relative flex items-center justify-between px-3 py-3 border-b md:border-b-0 border-lerd-chromeborder dark:border-lerd-border shrink-0 min-h-[48px]"
>
  <!-- Desktop only: on mobile the list is a full screen of its own. -->
  <span class="hidden md:contents"><PanelToggleButton title={m.nav_hideList()} /></span>
  <!-- Beside the toggle the rail's active icon already names the section; the nav keeps it for screen readers. -->
  <span class="md:sr-only text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
    {title}
  </span>
  {#if actions}
    <div class="ml-auto flex items-center gap-1">
      {@render actions()}
    </div>
  {/if}
</div>
