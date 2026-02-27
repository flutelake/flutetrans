<script>
  import ManageConnections from './features/connections/ManageConnections.svelte'
  import ActiveSessionsBar from './features/connections/components/ActiveSessionsBar.svelte'
  import FileBrowserPlaceholder from './features/browser/FileBrowserPlaceholder.svelte'
  import TransfersPage from './features/transfers/TransfersPage.svelte'
  import {connectionsStore, refreshConnections} from './features/connections/state/connectionsStore.js'
  import {sessionsStore} from './features/connections/state/sessionsStore.js'
  import {onDestroy, onMount} from 'svelte'
  import {Toaster} from 'svelte-french-toast'

  import {
    getMasterPasswordStatus,
    isWailsAvailable,
    lockMasterPassword,
    setMasterPassword
  } from './lib/wails/connectionService.js'

  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'
  import {Input} from '$lib/components/ui/input/index.js'
  import {t} from '$lib/i18n/index.js'

  const unsubscribe = sessionsStore.startListener()
  onDestroy(() => {
    if (unsubscribe) unsubscribe()
  })

  let securityLoading = true
  let securityStatus = null
  let unlockPassword = ''
  let unlockError = null

  async function loadSecurityStatus() {
    securityLoading = true
    unlockError = null
    try {
      securityStatus = await getMasterPasswordStatus()
    } catch (err) {
      securityStatus = {unlocked: false, hasEncryptedStore: false}
      unlockError = err
    } finally {
      securityLoading = false
    }
  }

  async function unlock() {
    unlockError = null
    try {
      await setMasterPassword(unlockPassword)
      await loadSecurityStatus()
      if (securityStatus?.unlocked) {
        await refreshConnections()
      }
    } catch (err) {
      unlockError = err
    }
  }

  async function lockNow() {
    unlockError = null
    try {
      await lockMasterPassword()
      unlockPassword = ''
      await loadSecurityStatus()
      connectionsStore.set({items: [], loading: false, error: null})
    } catch (err) {
      unlockError = err
    }
  }

  onMount(() => {
    let cancelled = false

    const run = async () => {
      if (cancelled) return

      if (!isWailsAvailable()) {
        setTimeout(run, 200)
        return
      }

      await loadSecurityStatus()
      if (securityStatus?.unlocked) {
        await refreshConnections()
      }
    }

    run().catch(() => {})
    return () => {
      cancelled = true
    }
  })

  $: state = $sessionsStore
  $: currentSession = state.sessions.find(s => s.sessionID === state.current)
  $: locked = !!(securityStatus?.hasEncryptedStore && !securityStatus?.unlocked)
  $: canLock = !!(securityStatus?.hasEncryptedStore && securityStatus?.unlocked)
</script>

<div class="h-screen min-h-0 p-6 flex flex-col gap-4 overflow-hidden">
  <ActiveSessionsBar
    sessions={state.sessions}
    current={state.current}
    onSelect={(id) => sessionsStore.setCurrent(id)}
    locked={locked}
    canLock={canLock}
    onLock={lockNow}
  />

  <div class="flex-1 min-h-0 flex">
    {#if state.current === 'connections'}
      <ManageConnections
        securityLoading={securityLoading}
        securityStatus={securityStatus}
        onSecurityChanged={loadSecurityStatus}
        onConnected={(sessionID) => sessionsStore.setCurrent(sessionID)}
      />
    {:else if state.current === 'transfers'}
      <TransfersPage
        sessions={state.sessions}
        onSelectSession={(sessionID) => sessionsStore.setCurrent(sessionID)}
      />
    {:else}
      <FileBrowserPlaceholder
        session={currentSession}
        onDisconnect={(sessionID) => sessionsStore.disconnect(sessionID)}
        onOpenTransfers={() => sessionsStore.setCurrent('transfers')}
      />
    {/if}
  </div>

  {#if securityLoading}
    <div class="fixed inset-0 z-[100] bg-background/60 backdrop-blur-sm"></div>
  {:else if locked}
    <div class="fixed inset-0 z-[100] bg-background/60 backdrop-blur-sm flex items-center justify-center p-6">
      <Card className="w-[520px]">
        <CardHeader className="text-left">
          <CardTitle>{$t('security.lockedTitle')}</CardTitle>
          <CardDescription>{$t('security.lockedDescription')}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <form class="space-y-3" on:submit|preventDefault={unlock}>
            <Input type="password" bind:value={unlockPassword} placeholder={$t('security.masterPasswordPlaceholder')} />
            <Button className="w-full" type="submit">{$t('security.unlock')}</Button>
          </form>
          {#if unlockError}
            <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
              {unlockError.message ?? $t('security.unlockFailed')}
            </div>
          {/if}
        </CardContent>
      </Card>
    </div>
  {/if}
  <Toaster position="top-right" />
</div>
