<script lang="ts">
  import SidePanel from './SidePanel.svelte';
  let { hidden }: { hidden: boolean } = $props();
</script>

<SidePanel {hidden}><p>list</p></SidePanel>
