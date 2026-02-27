<script>
  import ManageConnections from './features/connections/ManageConnections.svelte'
  import ActiveSessionsBar from './features/connections/components/ActiveSessionsBar.svelte'
  import FileBrowserPlaceholder from './features/browser/FileBrowserPlaceholder.svelte'
  import TransfersPage from './features/transfers/TransfersPage.svelte'
  import {connectionsStore, refreshConnections} from './features/connections/state/connectionsStore.js'
  import {sessionsStore} from './features/connections/state/sessionsStore.js'
  import {toasts} from './features/connections/ui/feedback.js'
  import {onDestroy, onMount} from 'svelte'

  import {
    getMasterPasswordStatus,
    isWailsAvailable,
    lockMasterPassword,
    setMasterPassword
  } from './lib/wails/connectionService.js'

  import {Button} from '$lib/components/ui/button/index.js'
  import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '$lib/components/ui/card/index.js'
  import {Input} from '$lib/components/ui/input/index.js'

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
  $: toastState = $toasts
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
          <CardTitle>已锁定</CardTitle>
          <CardDescription>请输入解锁密码继续使用。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <form class="space-y-3" on:submit|preventDefault={unlock}>
            <Input type="password" bind:value={unlockPassword} placeholder="Master password" />
            <Button className="w-full" type="submit">解锁</Button>
          </form>
          {#if unlockError}
            <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
              {unlockError.message ?? 'Unlock failed'}
            </div>
          {/if}
        </CardContent>
      </Card>
    </div>
  {/if}

  {#if toastState.length > 0}
    <div class="fixed right-4 top-4 z-50 flex w-80 flex-col gap-2">
      {#each toastState as t (t.id)}
        <div
          class={
            t.type === 'success'
              ? 'rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-left text-sm'
              : t.type === 'error'
                ? 'rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-left text-sm'
                : 'rounded-lg border border-border bg-card px-3 py-2 text-left text-sm'
          }
        >
          {#if t.title}
            <div class="font-medium">{t.title}</div>
          {/if}
          {#if t.message}
            <div class="text-muted-foreground">{t.message}</div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
