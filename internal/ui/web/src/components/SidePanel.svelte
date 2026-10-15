<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    hidden: boolean;
    children: Snippet;
  }
  let { hidden, children }: Props = $props();
</script>

<!-- Width animates rather than display, so the list keeps its scroll and state;
     the inner column keeps its width so rows slide out instead of squeezing. -->
<div
  inert={hidden}
  class="hidden md:flex shrink-0 overflow-hidden transition-[width] duration-200 ease-out {hidden
    ? 'w-0'
    : 'w-56 lg:w-64'}"
>
  <div class="flex flex-col w-56 lg:w-64 shrink-0 bg-lerd-header-light dark:bg-lerd-header overflow-hidden">
    {@render children()}
  </div>
</div>
