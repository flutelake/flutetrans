<script>
  import {onDestroy, onMount} from 'svelte'

  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'
  import Icon from '@iconify/svelte'
  import mdiDownload from '@iconify-icons/mdi/download'
  import mdiUpload from '@iconify-icons/mdi/upload'

  import {error as toastError} from '../connections/ui/feedback.js'
  import {transfersStore} from './state/transfersStore.js'
  import {t} from '$lib/i18n/index.js'

  export let sessions = []
  export let onSelectSession = (_sessionID) => {}

  let protocolTabs = []
  let directionTabs = []

  $: protocolTabs = [
    {id: 'all', label: $t('transfers.protocol.all')},
    {id: 'ftp', label: 'FTP'},
    {id: 'sftp', label: 'SFTP'},
    {id: 's3', label: 'S3'},
    {id: 'webdav', label: 'WebDAV'},
    {id: 'smb', label: 'SMB'},
    {id: 'nfs', label: 'NFS'}
  ]

  $: directionTabs = [
    {id: 'all', label: $t('transfers.protocol.all')},
    {id: 'upload', label: $t('transfers.direction.upload')},
    {id: 'download', label: $t('transfers.direction.download')}
  ]

  let protocol = 'all'
  let direction = 'all'

  function formatSize(bytes) {
    const n = Number(bytes ?? 0)
    if (!Number.isFinite(n) || n <= 0) return ''
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let v = n
    let i = 0
    while (v >= 1024 && i < units.length - 1) {
      v /= 1024
      i++
    }
    return `${v >= 10 || i === 0 ? v.toFixed(0) : v.toFixed(1)} ${units[i]}`
  }

  function canOpenSession(sessionID) {
    return !!sessions?.some?.(s => s.sessionID === sessionID)
  }

  function filteredItems(items, activeProtocol, activeDirection) {
    const list = Array.isArray(items) ? items : []
    return list
      .filter(t => (activeProtocol === 'all' ? true : t.protocol === activeProtocol))
      .filter(t => (activeDirection === 'all' ? true : t.direction === activeDirection))
      .slice()
      .sort((a, b) => (Number(b.startedAt ?? 0) || 0) - (Number(a.startedAt ?? 0) || 0))
  }

  let stopListener = () => {}

  onMount(() => {
    stopListener = transfersStore.startListener()
    transfersStore.refresh().catch(err => {
      toastError($t('transfers.loadFailedTitle'), err?.message ?? $t('connections.errors.unknownError'))
    })
    return () => {
      stopListener?.()
    }
  })

  onDestroy(() => {
    stopListener?.()
  })

  $: storeState = $transfersStore
  $: items = filteredItems(storeState.items, protocol, direction)
</script>

<div class="flex-1 min-h-0 flex flex-col gap-4">
  <header class="flex items-center justify-between gap-4">
    <div class="text-left">
      <div class="text-xl font-semibold tracking-tight">{$t('transfers.title')}</div>
      <div class="text-sm text-muted-foreground">{$t('transfers.description')}</div>
    </div>
    <Button variant="outline" on:click={() => transfersStore.refresh().catch(() => {})} disabled={storeState.loading}>
      {$t('common.refresh')}
    </Button>
  </header>

  {#if storeState.error}
    <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
      {storeState.error.message ?? $t('transfers.failedToLoad')}
    </div>
  {/if}

  <div class="flex items-center justify-between gap-4 flex-wrap">
    <div class="flex items-center gap-2 overflow-x-auto">
      {#each protocolTabs as tab (tab.id)}
        <Button size="sm" variant={protocol === tab.id ? 'secondary' : 'ghost'} on:click={() => (protocol = tab.id)}>
          {tab.label}
        </Button>
      {/each}
    </div>
    <div class="flex items-center gap-2">
      {#each directionTabs as tab (tab.id)}
        <Button size="sm" variant={direction === tab.id ? 'secondary' : 'ghost'} on:click={() => (direction = tab.id)}>
          {tab.label}
        </Button>
      {/each}
    </div>
  </div>

  <Card className="flex-1 min-h-0 flex flex-col">
    <CardContent className="flex-1 min-h-0 pt-6">
      <div class="h-full min-h-0 overflow-auto space-y-2">
        {#each items as tr (tr.id)}
          <div class="rounded-md border border-border px-3 py-2 text-left">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="text-sm font-medium break-all">
                  {tr.protocol?.toUpperCase?.() ?? tr.protocol} ·
                  {#if tr.direction === 'upload'}
                    <Icon icon={mdiUpload} class="inline-block h-4 w-4 align-[-0.125em] text-muted-foreground" aria-label={$t('transfers.direction.upload')} />
                  {:else}
                    <Icon icon={mdiDownload} class="inline-block h-4 w-4 align-[-0.125em] text-muted-foreground" aria-label={$t('transfers.direction.download')} />
                  {/if}
                  · {tr.remotePath}
                </div>
                <div class="mt-0.5 text-xs text-muted-foreground break-all">{tr.localPath}</div>
                <div class="mt-0.5 text-xs text-muted-foreground">
                  {$t('transfers.sessionLabel', {id: tr.sessionID})}
                </div>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <div class="text-xs text-muted-foreground">{tr.status}</div>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!canOpenSession(tr.sessionID)}
                  on:click={() => onSelectSession(tr.sessionID)}
                >
                  {$t('common.open')}
                </Button>
              </div>
            </div>

            <div class="mt-2">
              <div class="h-2 w-full rounded bg-muted">
                <div
                  class="h-2 rounded bg-emerald-500"
                  style={`width: ${tr.bytesTotal > 0 ? Math.min(100, Math.floor((tr.bytesTransferred / tr.bytesTotal) * 100)) : 0}%`}
                ></div>
              </div>
              <div class="mt-1 flex items-center justify-between text-xs text-muted-foreground gap-3">
                <div class="shrink-0">
                  {formatSize(tr.bytesTransferred)}{tr.bytesTotal > 0 ? ` / ${formatSize(tr.bytesTotal)}` : ''}
                </div>
                <div class="min-w-0 break-all">{tr.error ? tr.error : ''}</div>
              </div>
            </div>
          </div>
        {/each}

        {#if items.length === 0 && !storeState.loading}
          <div class="rounded-md border border-dashed border-border p-6 text-left text-sm text-muted-foreground">{$t('transfers.noTransfers')}</div>
        {/if}
      </div>
    </CardContent>
  </Card>
</div>
