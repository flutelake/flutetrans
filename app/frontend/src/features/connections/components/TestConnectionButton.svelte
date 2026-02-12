<script>
  import {Button} from '$lib/components/ui/button/index.js'
  import {testConnection} from '../../../lib/wails/connectionService.js'
  import ErrorDetailsDialog from './ErrorDetailsDialog.svelte'
  import {error as toastError, success} from '../ui/feedback.js'

  export let getProfile = () => null
  export let disabled = false

  let testing = false
  let last = null
  let error = null
  let showDetails = false
  let runToken = 0

  async function run() {
    error = null
    last = null
    testing = true
    const token = ++runToken
    try {
      const profile = getProfile()
      const result = await testConnection(profile)
      if (token !== runToken) return
      last = result
      if (result?.success) {
        success('Test succeeded', result?.latencyMs != null ? `${result.latencyMs}ms` : '')
      } else {
        toastError('Test failed', result?.message ?? 'Unknown error')
      }
    } catch (err) {
      if (token !== runToken) return
      error = err
      toastError('Test failed', err?.message ?? 'Unknown error')
    } finally {
      if (token === runToken) testing = false
    }
  }

  function cancel() {
    runToken++
    testing = false
  }
</script>

<div class="flex items-center gap-2">
  <Button variant="outline" size="sm" on:click={run} disabled={disabled || testing}>Test</Button>
  {#if testing}
    <Button variant="ghost" size="sm" on:click={cancel} disabled={disabled}>Cancel</Button>
  {:else if error || (last && !last.success)}
    <Button variant="ghost" size="sm" on:click={run} disabled={disabled}>Retry</Button>
  {/if}

  {#if error}
    <Button variant="ghost" size="sm" on:click={() => (showDetails = true)} disabled={disabled}>Details</Button>
  {/if}

  {#if testing}
    <div class="text-xs text-muted-foreground">Testing…</div>
  {:else if last}
    <div class="text-xs text-muted-foreground">{last.success ? 'Success' : 'Failed'}{last.latencyMs != null ? ` · ${last.latencyMs}ms` : ''}</div>
  {:else if error}
    <div class="text-xs text-destructive">{error.message ?? 'Test failed'}</div>
  {/if}
</div>

<ErrorDetailsDialog
  open={showDetails}
  title="Test connection"
  error={error}
  onRetry={run}
  onClose={() => (showDetails = false)}
/>
