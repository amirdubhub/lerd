<script lang="ts">
  import StripeControl from './StripeControl.svelte';

  interface Props {
    running?: boolean;
    loading?: boolean;
    webhookPath?: string;
    secretEnvKey?: string;
    onToggle?: () => void;
    onSaveConfig?: (path: string, secretEnvKey: string) => Promise<{ ok: boolean; error?: string }>;
  }
  let {
    running = false,
    loading = false,
    webhookPath = '',
    secretEnvKey = '',
    onToggle = () => {},
    onSaveConfig = async () => ({ ok: true })
  }: Props = $props();
</script>

<StripeControl {running} {loading} {webhookPath} {secretEnvKey} {onToggle} {onSaveConfig} />
