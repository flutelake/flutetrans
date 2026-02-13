<script>
  import ConnectionList from './components/ConnectionList.svelte'
  import ConnectionForm from './components/ConnectionForm.svelte'
  import {connectionsStore, refreshConnections} from './state/connectionsStore.js'
  import {
    connect,
    deleteConnection,
    getConnection,
    initializeMasterPassword,
    saveConnection
  } from '../../lib/wails/connectionService.js'
  import {error as toastError, success} from './ui/feedback.js'

  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'
  import {Input} from '$lib/components/ui/input/index.js'

  export let onConnected = (_sessionID) => {}
  export let securityLoading = true
  export let securityStatus = null
  export let onSecurityChanged = async () => {}

  let selectedId = null
  let formMode = 'new'
  let formProfile = null
  let saving = false
  let rightPanel = 'actions'

  let setupPassword = ''
  let setupConfirm = ''
  let setupError = null
  let actionError = null

  async function setupMasterPassword() {
    setupError = null
    actionError = null

    const password = setupPassword
    const confirm = setupConfirm
    if (!password || password.trim() === '') {
      setupError = {message: 'Master password required'}
      return
    }
    if (password.length < 8) {
      setupError = {message: 'Use at least 8 characters'}
      return
    }
    if (password !== confirm) {
      setupError = {message: 'Passwords do not match'}
      return
    }

    try {
      await initializeMasterPassword(password)
      await onSecurityChanged()
      await refreshConnections()
      success('Master password set', 'Encrypted store initialized')
    } catch (err) {
      setupError = err
      toastError('Setup failed', err?.message ?? 'Unknown error')
    }
  }

  async function openNew() {
    actionError = null
    selectedId = null
    formMode = 'new'
    formProfile = null
    rightPanel = 'form'
  }

  async function select(id) {
    actionError = null
    selectedId = id
  }

  async function openEdit(id) {
    actionError = null
    selectedId = id
    formMode = 'edit'
    rightPanel = 'form'
    try {
      formProfile = await getConnection(id)
    } catch (err) {
      actionError = err
      toastError('Load failed', err?.message ?? 'Unknown error')
      closeForm()
    }
  }

  function closeForm() {
    rightPanel = 'actions'
    formProfile = null
    formMode = 'new'
  }

  async function remove(id) {
    actionError = null
    if (!confirm('Delete this connection?')) return

    try {
      await deleteConnection(id)
      if (selectedId === id) {
        selectedId = null
        closeForm()
      }
      await refreshConnections()
      success('Deleted', 'Connection removed')
    } catch (err) {
      actionError = err
      toastError('Delete failed', err?.message ?? 'Unknown error')
    }
  }

  async function quickConnect(id) {
    actionError = null
    try {
      const sessionID = await connect(id)
      onConnected(sessionID)
      success('Connecting', `Session ${sessionID.slice(0, 8)}`)
    } catch (err) {
      actionError = err
      toastError('Connect failed', err?.message ?? 'Unknown error')
    }
  }

  async function save(profile) {
    actionError = null
    saving = true
    try {
      const saved = await saveConnection(profile)
      selectedId = saved.id
      await refreshConnections()
      closeForm()
      success('Saved', 'Connection updated')
    } catch (err) {
      actionError = err
      toastError('Save failed', err?.message ?? 'Unknown error')
    } finally {
      saving = false
    }
  }

  async function cancelEdit() {
    actionError = null
    closeForm()
  }

  async function exportConnections() {
    actionError = null
    try {
      const payload = {
        exportedAt: new Date().toISOString(),
        items: storeState?.items ?? []
      }
      const blob = new Blob([JSON.stringify(payload, null, 2)], {type: 'application/json'})
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `connections-export-${Date.now()}.json`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      success('Exported', 'connections-export.json downloaded')
    } catch (err) {
      actionError = err
      toastError('Export failed', err?.message ?? 'Unknown error')
    }
  }

  $: storeState = $connectionsStore
</script>

<div class="flex-1 min-h-0 flex flex-col gap-4">
  {#if securityLoading}
    <div class="flex-1 min-h-0 flex items-center justify-center">
      <div class="text-sm text-muted-foreground">Loading…</div>
    </div>
  {:else if !securityStatus?.hasEncryptedStore}
    <div class="flex-1 min-h-0 flex items-center justify-center">
      <Card className="w-[520px]">
        <CardHeader className="text-left">
          <CardTitle>Set master password</CardTitle>
          <CardDescription>This password encrypts stored credentials on your device.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div class="space-y-2">
            <Input type="password" bind:value={setupPassword} placeholder="Create master password" />
            <Input type="password" bind:value={setupConfirm} placeholder="Confirm master password" />
            <Button className="w-full" on:click={setupMasterPassword}>Continue</Button>
          </div>
          <div class="text-left text-xs text-muted-foreground">
            If you forget this password, encrypted credentials cannot be recovered.
          </div>
          {#if setupError}
            <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
              {setupError.message ?? 'Setup failed'}
            </div>
          {/if}
        </CardContent>
      </Card>
    </div>
  {:else}
    <header class="flex items-center justify-between gap-4">
      <div class="text-left">
        <div class="text-xl font-semibold tracking-tight">Manage Connections</div>
        <div class="text-sm text-muted-foreground">Create, edit, and switch saved connections.</div>
      </div>
    </header>

    {#if storeState.error}
      <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
        {storeState.error.message ?? 'Failed to load connections'}
      </div>
    {/if}
    {#if actionError}
      <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
        {actionError.message ?? 'Action failed'}
      </div>
    {/if}

    <div class="grid grid-cols-[360px_minmax(0,1fr)] grid-rows-1 gap-4 flex-1 min-h-0">
      <div class="h-full min-h-0">
        <ConnectionList
          items={storeState.items}
          selectedId={selectedId}
          onSelect={select}
          onConnect={quickConnect}
          onEdit={openEdit}
          onDelete={remove}
        />
      </div>

      <div class="h-full min-h-0">
        {#if rightPanel === 'form'}
          <ConnectionForm
            mode={formMode}
            initialProfile={formProfile}
            saving={saving}
            onSave={save}
            onCancel={cancelEdit}
          />
        {:else}
          <Card className="h-full flex flex-col">
            <CardHeader className="space-y-1 text-left">
              <CardTitle className="text-base">Actions</CardTitle>
              <CardDescription>Manage saved connections.</CardDescription>
            </CardHeader>
            <CardContent className="flex-1 min-h-0">
              <div class="flex flex-col gap-2">
                <Button on:click={openNew}>New connection</Button>
                <Button variant="outline" on:click={exportConnections} disabled={!storeState?.items?.length}>Export connections</Button>
              </div>
            </CardContent>
          </Card>
        {/if}
      </div>
    </div>
  {/if}
</div>
