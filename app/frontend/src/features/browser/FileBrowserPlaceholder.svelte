<script>
  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'

  import Icon from '@iconify/svelte'

  import mdiFolder from '@iconify-icons/mdi/folder'
  import mdiFileOutline from '@iconify-icons/mdi/file-outline'
  import mdiFileDocumentOutline from '@iconify-icons/mdi/file-document-outline'
  import mdiFileImageOutline from '@iconify-icons/mdi/file-image-outline'
  import mdiFileVideoOutline from '@iconify-icons/mdi/file-video-outline'
  import mdiFileMusicOutline from '@iconify-icons/mdi/file-music-outline'
  import mdiFolderZipOutline from '@iconify-icons/mdi/folder-zip-outline'
  import mdiFileCodeOutline from '@iconify-icons/mdi/file-code-outline'
  import mdiFileCsvOutline from '@iconify-icons/mdi/file-csv-outline'
  import mdiFilePdfBox from '@iconify-icons/mdi/file-pdf-box'

  import {error as toastError, success} from '../connections/ui/feedback.js'
  import {listFiles, pickUploadFiles, startDownload, startUpload} from '../../lib/wails/connectionService.js'

  export let session
  export let onDisconnect = (_sessionID) => {}
  export let onOpenTransfers = () => {}

  let currentPath = ''
  let entries = []
  let loading = false
  let loadError = null

  $: sessionID = session?.sessionID
  $: connected = session?.status === 'connected'
  $: profileName = session?.profileName || session?.connectionName || session?.name
  $: sessionDisplayName = profileName || (sessionID ? sessionID.slice(0, 8) : '')

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

  function formatTime(ms) {
    const n = Number(ms ?? 0)
    if (!Number.isFinite(n) || n <= 0) return ''
    const d = new Date(n)
    return d.toLocaleString()
  }

  function parentPath(p) {
    if (!p || p === '.' || p === '/') return p
    const trimmed = p.endsWith('/') ? p.slice(0, -1) : p
    const idx = trimmed.lastIndexOf('/')
    if (idx === -1) return '.'
    if (idx === 0) return '/'
    return trimmed.slice(0, idx)
  }

  function normalizePath(p) {
    const v = String(p ?? '').trim()
    if (!v) return ''
    if (v === '.') return '.'
    if (v === '/') return '/'
    return v
  }

  function splitPath(p) {
    const v = normalizePath(p)
    if (!v || v === '.') {
      return [{label: '.', path: '.'}]
    }
    if (v === '/') {
      return [{label: '/', path: '/'}]
    }

    const isAbs = v.startsWith('/')
    const cleaned = isAbs ? v.slice(1) : v
    const parts = cleaned.split('/').filter(Boolean)

    const out = []
    if (isAbs) {
      out.push({label: '/', path: '/'})
      let acc = ''
      for (const part of parts) {
        acc = acc ? `${acc}/${part}` : part
        out.push({label: part, path: `/${acc}`})
      }
      return out
    }

    out.push({label: '.', path: '.'})
    let acc = ''
    for (const part of parts) {
      acc = acc ? `${acc}/${part}` : part
      out.push({label: part, path: acc})
    }
    return out
  }

  function fileExt(name) {
    const n = String(name ?? '')
    const idx = n.lastIndexOf('.')
    if (idx <= 0 || idx === n.length - 1) return ''
    return n.slice(idx + 1).toLowerCase()
  }

  function entryKind(item) {
    if (item?.isDir) return 'folder'

    const ext = fileExt(item?.name)
    if (!ext) return 'file'

    if (
      ext === 'png' ||
      ext === 'jpg' ||
      ext === 'jpeg' ||
      ext === 'gif' ||
      ext === 'bmp' ||
      ext === 'webp' ||
      ext === 'svg' ||
      ext === 'ico' ||
      ext === 'tiff'
    ) {
      return 'image'
    }

    if (
      ext === 'mp4' ||
      ext === 'mkv' ||
      ext === 'avi' ||
      ext === 'mov' ||
      ext === 'wmv' ||
      ext === 'webm' ||
      ext === 'm4v'
    ) {
      return 'video'
    }

    if (ext === 'mp3' || ext === 'wav' || ext === 'flac' || ext === 'aac' || ext === 'ogg' || ext === 'm4a') {
      return 'audio'
    }

    if (
      ext === 'zip' ||
      ext === 'rar' ||
      ext === '7z' ||
      ext === 'tar' ||
      ext === 'gz' ||
      ext === 'bz2' ||
      ext === 'xz' ||
      ext === 'tgz'
    ) {
      return 'archive'
    }

    if (ext === 'pdf') return 'pdf'
    if (ext === 'csv') return 'csv'

    if (
      ext === 'js' ||
      ext === 'ts' ||
      ext === 'jsx' ||
      ext === 'tsx' ||
      ext === 'go' ||
      ext === 'py' ||
      ext === 'java' ||
      ext === 'kt' ||
      ext === 'rs' ||
      ext === 'c' ||
      ext === 'h' ||
      ext === 'cpp' ||
      ext === 'hpp' ||
      ext === 'cs' ||
      ext === 'swift' ||
      ext === 'php' ||
      ext === 'rb' ||
      ext === 'sh' ||
      ext === 'yml' ||
      ext === 'yaml' ||
      ext === 'json' ||
      ext === 'toml' ||
      ext === 'ini' ||
      ext === 'md'
    ) {
      return 'code'
    }

    if (ext === 'txt' || ext === 'log') return 'text'

    return 'file'
  }

  function iconForEntry(item) {
    const kind = entryKind(item)
    if (kind === 'folder') return mdiFolder
    if (kind === 'image') return mdiFileImageOutline
    if (kind === 'video') return mdiFileVideoOutline
    if (kind === 'audio') return mdiFileMusicOutline
    if (kind === 'archive') return mdiFolderZipOutline
    if (kind === 'pdf') return mdiFilePdfBox
    if (kind === 'csv') return mdiFileCsvOutline
    if (kind === 'code') return mdiFileCodeOutline
    if (kind === 'text') return mdiFileDocumentOutline
    return mdiFileOutline
  }

  function iconClassForEntry(item) {
    const kind = entryKind(item)
    if (kind === 'folder') return 'text-amber-500'
    if (kind === 'image') return 'text-fuchsia-500'
    if (kind === 'video') return 'text-rose-500'
    if (kind === 'audio') return 'text-emerald-500'
    if (kind === 'archive') return 'text-orange-500'
    if (kind === 'pdf') return 'text-red-500'
    if (kind === 'csv') return 'text-lime-600'
    if (kind === 'code') return 'text-sky-500'
    if (kind === 'text') return 'text-muted-foreground'
    return 'text-muted-foreground'
  }

  async function load(targetPath = '') {
    if (!sessionID) return
    loadError = null
    loading = true
    try {
      const result = await listFiles(sessionID, targetPath)
      currentPath = result?.path ?? ''
      entries = Array.isArray(result?.entries) ? result.entries : []
    } catch (err) {
      loadError = err
      toastError('Load files failed', err?.message ?? 'Unknown error')
    } finally {
      loading = false
    }
  }

  async function goUp() {
    await load(parentPath(currentPath))
  }

  async function openEntry(item) {
    if (!item) return
    if (item.isDir) {
      await load(item.path)
      return
    }
  }

  function sortEntries(items) {
    const list = Array.isArray(items) ? items : []
    return list
      .slice()
      .sort((a, b) => {
        const ad = a?.isDir ? 0 : 1
        const bd = b?.isDir ? 0 : 1
        if (ad !== bd) return ad - bd
        return String(a?.name ?? '').localeCompare(String(b?.name ?? ''), undefined, {numeric: true, sensitivity: 'base'})
      })
  }

  async function download(item) {
    if (!sessionID || !item || item.isDir) return
    try {
      await startDownload(sessionID, item.path)
      success('Download started', item.name)
    } catch (err) {
      if (String(err?.message ?? '').toLowerCase().includes('canceled')) return
      toastError('Download failed', err?.message ?? 'Unknown error')
    }
  }

  async function uploadViaDialog() {
    if (!sessionID) return
    try {
      const paths = await pickUploadFiles()
      if (!paths || paths.length === 0) return
      await startUpload(sessionID, paths, currentPath)
      success('Upload started', `${paths.length} item(s)`) 
    } catch (err) {
      toastError('Upload failed', err?.message ?? 'Unknown error')
    }
  }

  $: if (sessionID && connected) {
    load('')
  }

  $: breadcrumbs = splitPath(currentPath)
  $: sortedEntries = sortEntries(entries)
</script>

<div class="flex-1 min-h-0 flex flex-col">
  <div class="flex-1 min-h-0 flex flex-col gap-4">
  <header class="flex items-center justify-between gap-4">
    <div class="min-w-0 text-left">
      <div class="text-xl font-semibold tracking-tight">FileBrowser</div>
      <div class="text-sm text-muted-foreground">
        {#if sessionID}
          来自连接：{sessionDisplayName}
        {:else}
          请选择一个连接以浏览文件。
        {/if}
      </div>
    </div>

    <div class="flex items-center gap-2 shrink-0">
      {#if sessionID && connected}
        <Button
          size="sm"
          variant="outline"
          on:click={goUp}
          disabled={loading || currentPath === '' || currentPath === '.' || currentPath === '/'}
        >
          Up
        </Button>
        <Button size="sm" variant="outline" on:click={() => load(currentPath)} disabled={loading}>Refresh</Button>
        <Button size="sm" on:click={uploadViaDialog} disabled={loading}>Upload</Button>
      {/if}
      {#if sessionID}
        <Button variant="outline" size="sm" on:click={onOpenTransfers}>传输</Button>
        <Button variant="secondary" size="sm" on:click={() => onDisconnect(sessionID)}>断开</Button>
      {/if}
    </div>
  </header>

  {#if !sessionID}
    <Card className="flex-1 min-h-0 flex flex-col">
      <CardHeader className="space-y-1 text-left">
        <CardTitle className="text-base">No active session</CardTitle>
        <CardDescription>Connect to a server to browse files.</CardDescription>
      </CardHeader>
      <CardContent className="flex-1 min-h-0"></CardContent>
    </Card>
  {:else if !connected}
    <Card className="flex-1 min-h-0 flex flex-col">
      <CardHeader className="space-y-1 text-left">
        <CardTitle className="text-base">连接中</CardTitle>
        <CardDescription>{session?.message || 'Please wait...'}</CardDescription>
      </CardHeader>
      <CardContent className="flex-1 min-h-0"></CardContent>
    </Card>
  {:else}
    <div class="flex-1 min-h-0">
      <Card className="h-full min-h-0 flex flex-col overflow-hidden">
        {#if sessionID}
          <div class="shrink-0 border-b border-border bg-muted/20 px-3 py-2">
            <div class="flex items-center gap-1 overflow-x-auto text-left text-sm">
              {#each breadcrumbs as c (c.path)}
                <button
                  type="button"
                  class={
                    c === breadcrumbs[breadcrumbs.length - 1]
                      ? 'rounded px-1.5 py-1 font-medium text-foreground'
                      : 'rounded px-1.5 py-1 text-muted-foreground hover:bg-accent/60 hover:text-foreground'
                  }
                  on:click={() => load(c.path)}
                  disabled={loading || c === breadcrumbs[breadcrumbs.length - 1]}
                >
                  {c.label}
                </button>
                {#if c !== breadcrumbs[breadcrumbs.length - 1]}
                  <span class="px-0.5 text-muted-foreground">/</span>
                {/if}
              {/each}
            </div>
          </div>
        {/if}

        <CardContent className="flex-1 min-h-0 p-0 flex flex-col">
          {#if loadError}
            <div class="p-4">
              <div class="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
                {loadError.message ?? 'Load failed'}
              </div>
            </div>
          {/if}

          <div class="flex-1 min-h-0 overflow-auto">
            <table class="w-full text-left text-sm">
              <thead class="sticky top-0 bg-card">
                <tr class="border-b border-border">
                  <th class="px-4 py-2.5 font-medium">Name</th>
                  <th class="px-4 py-2.5 font-medium w-[140px]">Size</th>
                  <th class="px-4 py-2.5 font-medium w-[200px]">Modified</th>
                  <th class="px-4 py-2.5 font-medium w-[120px]"></th>
                </tr>
              </thead>
              <tbody>
                {#each sortedEntries as item (item.path)}
                  <tr class="border-b border-border hover:bg-accent/50 cursor-pointer" on:click={() => openEntry(item)}>
                    <td class="px-4 py-2.5">
                      <div class="flex items-center gap-2">
                        <Icon
                          icon={iconForEntry(item)}
                          width={18}
                          height={18}
                          class={`shrink-0 ${iconClassForEntry(item)}`}
                        />
                        <span class={item.isDir ? 'font-medium break-all' : 'break-all'}>{item.name}</span>
                      </div>
                    </td>
                    <td class="px-4 py-2.5 text-muted-foreground">{item.isDir ? '' : formatSize(item.size)}</td>
                    <td class="px-4 py-2.5 text-muted-foreground">{formatTime(item.modifiedAt)}</td>
                    <td class="px-4 py-2.5" on:click|stopPropagation>
                      {#if !item.isDir}
                        <Button size="sm" variant="secondary" on:click={() => download(item)}>Download</Button>
                      {/if}
                    </td>
                  </tr>
                {/each}
                {#if entries.length === 0 && !loading}
                  <tr>
                    <td class="px-4 py-6 text-sm text-muted-foreground" colspan="4">Empty</td>
                  </tr>
                {/if}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  {/if}
  </div>
</div>
