<script>
  import ConnectionList from './components/ConnectionList.svelte'
  import ConnectionForm from './components/ConnectionForm.svelte'
  import {connectionsStore, refreshConnections} from './state/connectionsStore.js'
  import {sessionsStore} from './state/sessionsStore.js'
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
  import {t} from '$lib/i18n/index.js'

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
      setupError = {message: $t('security.setup.errors.required')}
      return
    }
    if (password.length < 8) {
      setupError = {message: $t('security.setup.errors.minLength')}
      return
    }
    if (password !== confirm) {
      setupError = {message: $t('security.setup.errors.mismatch')}
      return
    }

    try {
      await initializeMasterPassword(password)
      await onSecurityChanged()
      await refreshConnections()
      success($t('connections.toasts.masterPasswordSetTitle'), $t('connections.toasts.masterPasswordSetMessage'))
    } catch (err) {
      setupError = err
      toastError($t('security.setup.errors.failed'), err?.message ?? $t('connections.errors.unknownError'))
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
    if (!confirm($t('connections.confirmDelete'))) return

    try {
      await deleteConnection(id)
      if (selectedId === id) {
        selectedId = null
        closeForm()
      }
      await refreshConnections()
      success($t('connections.toasts.deletedTitle'), $t('connections.toasts.deletedMessage'))
    } catch (err) {
      actionError = err
      toastError($t('connections.errors.deleteFailedTitle'), err?.message ?? $t('connections.errors.unknownError'))
    }
  }

  async function quickConnect(id) {
    actionError = null
    try {
      const profile = storeState?.items?.find?.(x => x.id === id)
      const sessionID = await connect(id)
      sessionsStore.setSessionMeta(sessionID, {
        profileID: id,
        profileName: profile?.name ?? '',
        protocol: profile?.protocol ?? ''
      })
      onConnected(sessionID)
      success($t('connections.toasts.connectingTitle'), $t('connections.toasts.connectingMessage', {session: sessionID.slice(0, 8)}))
    } catch (err) {
      actionError = err
      toastError($t('connections.errors.connectFailedTitle'), err?.message ?? $t('connections.errors.unknownError'))
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
      success($t('connections.toasts.savedTitle'), $t('connections.toasts.savedMessage'))
    } catch (err) {
      actionError = err
      toastError($t('connections.errors.saveFailedTitle'), err?.message ?? $t('connections.errors.unknownError'))
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
      success($t('connections.actions.exportSuccessTitle'), $t('connections.actions.exportSuccessMessage'))
    } catch (err) {
      actionError = err
      toastError($t('connections.errors.exportFailedTitle'), err?.message ?? $t('connections.errors.unknownError'))
    }
  }

  $: storeState = $connectionsStore
</script>

<div class="flex-1 min-h-0 flex flex-col gap-4">
  {#if securityLoading}
    <div class="flex-1 min-h-0 flex items-center justify-center">
      <div class="text-sm text-muted-foreground">{$t('common.loading')}</div>
    </div>
  {:else if !securityStatus?.hasEncryptedStore}
    <div class="flex-1 min-h-0 flex items-center justify-center">
      <Card className="w-[520px]">
        <CardHeader className="text-left">
          <CardTitle>{$t('security.setup.title')}</CardTitle>
          <CardDescription>{$t('security.setup.description')}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div class="space-y-2">
            <Input type="password" bind:value={setupPassword} placeholder={$t('security.setup.createPlaceholder')} />
            <Input type="password" bind:value={setupConfirm} placeholder={$t('security.setup.confirmPlaceholder')} />
            <Button className="w-full" on:click={setupMasterPassword}>{$t('security.setup.continue')}</Button>
          </div>
          <div class="text-left text-xs text-muted-foreground">
            {$t('security.setup.forgetHint')}
          </div>
          {#if setupError}
            <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
              {setupError.message ?? $t('security.setup.errors.failed')}
            </div>
          {/if}
        </CardContent>
      </Card>
    </div>
  {:else}
    <header class="flex items-center justify-between gap-4">
      <div class="text-left">
        <div class="text-xl font-semibold tracking-tight">{$t('connections.manageTitle')}</div>
        <div class="text-sm text-muted-foreground">{$t('connections.manageDescription')}</div>
      </div>
    </header>

    {#if storeState.error}
      <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
        {storeState.error.message ?? $t('connections.errors.failedToLoadConnections')}
      </div>
    {/if}
    {#if actionError}
      <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
        {actionError.message ?? $t('connections.errors.actionFailed')}
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
              <CardTitle className="text-base">{$t('common.actions')}</CardTitle>
              <CardDescription>{$t('connections.list.description')}</CardDescription>
            </CardHeader>
            <CardContent className="flex-1 min-h-0">
              <div class="flex flex-col gap-2">
                <Button on:click={openNew}>{$t('connections.actions.new')}</Button>
                <Button variant="outline" on:click={exportConnections} disabled={!storeState?.items?.length}>{$t('connections.actions.export')}</Button>
              </div>
            </CardContent>
          </Card>
        {/if}
      </div>
    </div>
  {/if}
</div>
