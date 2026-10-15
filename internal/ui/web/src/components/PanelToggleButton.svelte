<script lang="ts">
  import { tooltip } from '$lib/tooltip';
  import { toggleSidePanel } from '$stores/sidePanel';
  import Icon from './Icon.svelte';

  interface Props {
    title: string;
  }
  let { title }: Props = $props();
</script>

<!-- Pinned to the same corner of the list header and of the page header, so the
     button stays under the pointer whichever way it toggles. Sized like the git
     fetch/pull/push buttons; pinned so it never makes either header taller. -->
<button
  type="button"
  onclick={toggleSidePanel}
  use:tooltip={title}
  aria-label={title}
  class="absolute top-2 left-1.5 z-20 h-8 min-w-8 px-1.5 flex items-center justify-center rounded-md transition-colors text-gray-500 dark:text-gray-400 hover:text-lerd-red hover:bg-gray-100 dark:hover:bg-white/5"
>
  <Icon name="sidebar" class="w-4 h-4" />
</button>
