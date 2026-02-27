<script>
  import {Button} from '$lib/components/ui/button/index.js'
  import {testConnection} from '../../../lib/wails/connectionService.js'
  import ErrorDetailsDialog from './ErrorDetailsDialog.svelte'
  import {error as toastError, success} from '../ui/feedback.js'
  import {t} from '$lib/i18n/index.js'

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
        success($t('testConnection.toastSuccessTitle'), result?.latencyMs != null ? `${result.latencyMs}ms` : '')
      } else {
        toastError($t('testConnection.toastFailedTitle'), result?.message ?? $t('connections.errors.unknownError'))
      }
    } catch (err) {
      if (token !== runToken) return
      error = err
      toastError($t('testConnection.toastFailedTitle'), err?.message ?? $t('connections.errors.unknownError'))
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
  <Button variant="outline" size="sm" on:click={run} disabled={disabled || testing}>{$t('connections.form.test')}</Button>
  {#if testing}
    <Button variant="ghost" size="sm" on:click={cancel} disabled={disabled}>{$t('common.cancel')}</Button>
  {:else if error || (last && !last.success)}
    <Button variant="ghost" size="sm" on:click={run} disabled={disabled}>{$t('common.retry')}</Button>
  {/if}

  {#if error}
    <Button variant="ghost" size="sm" on:click={() => (showDetails = true)} disabled={disabled}>{$t('common.details')}</Button>
  {/if}

  {#if testing}
    <div class="text-xs text-muted-foreground">{$t('testConnection.testing')}</div>
  {:else if last}
    <div class="text-xs text-muted-foreground">{last.success ? $t('testConnection.statusSuccess') : $t('testConnection.statusFailed')}{last.latencyMs != null ? ` · ${last.latencyMs}ms` : ''}</div>
  {:else if error}
    <div class="text-xs text-destructive">{error.message ?? $t('testConnection.failedFallback')}</div>
  {/if}
</div>

<ErrorDetailsDialog
  open={showDetails}
  title={$t('testConnection.dialogTitle')}
  error={error}
  onRetry={run}
  onClose={() => (showDetails = false)}
/>
