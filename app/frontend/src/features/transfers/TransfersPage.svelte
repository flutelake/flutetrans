<script>
  import {onDestroy, onMount} from 'svelte'

  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'

  import {error as toastError} from '../connections/ui/feedback.js'
  import {transfersStore} from './state/transfersStore.js'

  export let sessions = []
  export let onSelectSession = (_sessionID) => {}

  const protocolTabs = [
    {id: 'all', label: 'All'},
    {id: 'ftp', label: 'FTP'},
    {id: 'sftp', label: 'SFTP'},
    {id: 's3', label: 'S3'},
    {id: 'webdav', label: 'WebDAV'},
    {id: 'smb', label: 'SMB'},
    {id: 'nfs', label: 'NFS'}
  ]

  const directionTabs = [
    {id: 'all', label: 'All'},
    {id: 'upload', label: 'Upload'},
    {id: 'download', label: 'Download'}
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

  function filteredItems(items) {
    const list = Array.isArray(items) ? items : []
    return list
      .filter(t => (protocol === 'all' ? true : t.protocol === protocol))
      .filter(t => (direction === 'all' ? true : t.direction === direction))
      .slice()
      .sort((a, b) => (Number(b.startedAt ?? 0) || 0) - (Number(a.startedAt ?? 0) || 0))
  }

  let stopListener = () => {}

  onMount(() => {
    stopListener = transfersStore.startListener()
    transfersStore.refresh().catch(err => {
      toastError('Load transfers failed', err?.message ?? 'Unknown error')
    })
    return () => {
      stopListener?.()
    }
  })

  onDestroy(() => {
    stopListener?.()
  })

  $: storeState = $transfersStore
  $: items = filteredItems(storeState.items)
</script>

<div class="flex-1 min-h-0 flex flex-col gap-4">
  <header class="flex items-center justify-between gap-4">
    <div class="text-left">
      <div class="text-xl font-semibold tracking-tight">Transfers</div>
      <div class="text-sm text-muted-foreground">集中管理所有协议的上传/下载任务。</div>
    </div>
    <Button variant="outline" on:click={() => transfersStore.refresh().catch(() => {})} disabled={storeState.loading}>
      Refresh
    </Button>
  </header>

  {#if storeState.error}
    <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
      {storeState.error.message ?? 'Failed to load transfers'}
    </div>
  {/if}

  <div class="flex items-center justify-between gap-4 flex-wrap">
    <div class="flex items-center gap-2 overflow-x-auto">
      {#each protocolTabs as t (t.id)}
        <Button size="sm" variant={protocol === t.id ? 'secondary' : 'ghost'} on:click={() => (protocol = t.id)}>
          {t.label}
        </Button>
      {/each}
    </div>
    <div class="flex items-center gap-2">
      {#each directionTabs as t (t.id)}
        <Button size="sm" variant={direction === t.id ? 'secondary' : 'ghost'} on:click={() => (direction = t.id)}>
          {t.label}
        </Button>
      {/each}
    </div>
  </div>

  <Card className="flex-1 min-h-0 flex flex-col">
    <CardContent className="flex-1 min-h-0 pt-6">
      <div class="h-full min-h-0 overflow-auto space-y-2">
        {#each items as t (t.id)}
          <div class="rounded-md border border-border px-3 py-2 text-left">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="text-sm font-medium break-all">
                  {t.protocol?.toUpperCase?.() ?? t.protocol} · {t.direction === 'upload' ? 'Upload' : 'Download'} · {t.remotePath}
                </div>
                <div class="mt-0.5 text-xs text-muted-foreground break-all">{t.localPath}</div>
                <div class="mt-0.5 text-xs text-muted-foreground">
                  Session: {t.sessionID}
                </div>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <div class="text-xs text-muted-foreground">{t.status}</div>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={!canOpenSession(t.sessionID)}
                  on:click={() => onSelectSession(t.sessionID)}
                >
                  Open
                </Button>
              </div>
            </div>

            <div class="mt-2">
              <div class="h-2 w-full rounded bg-muted">
                <div
                  class="h-2 rounded bg-emerald-500"
                  style={`width: ${t.bytesTotal > 0 ? Math.min(100, Math.floor((t.bytesTransferred / t.bytesTotal) * 100)) : 0}%`}
                ></div>
              </div>
              <div class="mt-1 flex items-center justify-between text-xs text-muted-foreground gap-3">
                <div class="shrink-0">
                  {formatSize(t.bytesTransferred)}{t.bytesTotal > 0 ? ` / ${formatSize(t.bytesTotal)}` : ''}
                </div>
                <div class="min-w-0 break-all">{t.error ? t.error : ''}</div>
              </div>
            </div>
          </div>
        {/each}

        {#if items.length === 0 && !storeState.loading}
          <div class="rounded-md border border-dashed border-border p-6 text-left text-sm text-muted-foreground">No transfers</div>
        {/if}
      </div>
    </CardContent>
  </Card>
</div>
