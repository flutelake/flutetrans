<script>
  import ManageConnections from './features/connections/ManageConnections.svelte'
  import ActiveSessionsBar from './features/connections/components/ActiveSessionsBar.svelte'
  import FileBrowserPlaceholder from './features/browser/FileBrowserPlaceholder.svelte'
  import TransfersPage from './features/transfers/TransfersPage.svelte'
  import {connectionsStore, refreshConnections} from './features/connections/state/connectionsStore.js'
  import {sessionsStore} from './features/connections/state/sessionsStore.js'
  import {onDestroy, onMount} from 'svelte'
  import {Toaster} from 'svelte-french-toast'
  import toast from 'svelte-french-toast'

  import {
    changeMasterPassword,
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
  let unlockErrorMessage = ''

  let changeOpen = false
  let changeWorking = false
  let changeCurrent = ''
  let changeNext = ''
  let changeConfirm = ''
  let changeError = null

  function toastText(title, message) {
    return [title, message].filter(v => v != null && String(v).trim() !== '').join('\n')
  }

  function resolveUnlockErrorMessage(err) {
    if (!err) return ''
    const code = err?.code
    const message = String(err?.message ?? '')
    if (code === 2002) return $t('security.errors.invalidMasterPassword')
    if (code === 1001 && message === 'master password required') return $t('security.errors.required')
    if (code === 1003 && message === 'invalid encrypted store') return $t('security.errors.invalidEncryptedStore')
    return err?.message ?? $t('security.unlockFailed')
  }

  function resolveChangeErrorMessage(err) {
    if (!err) return ''
    const code = err?.code
    const message = String(err?.message ?? '')

    if (code === 2002) return $t('security.errors.invalidMasterPassword')
    if (code === 1001 && message === 'master password required') return $t('security.errors.required')
    if (code === 1001 && message === 'new master password required') return $t('security.change.errors.newRequired')
    if (code === 1001 && message === 'new master password too short') return $t('security.change.errors.minLength', {min: 8})
    if (code === 1003 && message === 'invalid encrypted store') return $t('security.errors.invalidEncryptedStore')
    return err?.message ?? $t('security.change.failedTitle')
  }

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

  function openChangePassword() {
    if (locked || !canLock) return
    changeError = null
    changeCurrent = ''
    changeNext = ''
    changeConfirm = ''
    changeOpen = true
  }

  function closeChangePassword() {
    if (changeWorking) return
    changeOpen = false
    changeError = null
  }

  async function submitChangePassword() {
    if (changeWorking) return
    changeError = null

    if (!changeCurrent || changeCurrent.trim() === '') {
      changeError = {code: 1001, message: 'master password required'}
      return
    }
    if (!changeNext || changeNext.trim() === '') {
      changeError = {code: 1001, message: 'new master password required'}
      return
    }
    if (String(changeNext).length < 8) {
      changeError = {code: 1001, message: 'new master password too short'}
      return
    }
    if (changeNext !== changeConfirm) {
      changeError = {code: 1001, message: 'password mismatch'}
      return
    }

    changeWorking = true
    try {
      await changeMasterPassword(changeCurrent, changeNext)
      changeOpen = false
      changeCurrent = ''
      changeNext = ''
      changeConfirm = ''
      await loadSecurityStatus()
      await refreshConnections()
      toast.success(toastText($t('security.change.successTitle'), ''), {duration: 3000})
    } catch (err) {
      changeError = err
      toast.error(toastText($t('security.change.failedTitle'), resolveChangeErrorMessage(err)), {duration: 5000})
    } finally {
      changeWorking = false
    }
  }

  $: state = $sessionsStore
  $: currentSession = state.sessions.find(s => s.sessionID === state.current)
  $: locked = !!(securityStatus?.hasEncryptedStore && !securityStatus?.unlocked)
  $: canLock = !!(securityStatus?.hasEncryptedStore && securityStatus?.unlocked)
  $: unlockErrorMessage = resolveUnlockErrorMessage(unlockError)
  $: changeErrorMessage = changeError?.message === 'password mismatch' ? $t('security.change.errors.mismatch') : resolveChangeErrorMessage(changeError)
</script>

<div class="h-screen min-h-0 p-6 flex flex-col gap-4 overflow-hidden">
  <ActiveSessionsBar
    sessions={state.sessions}
    current={state.current}
    onSelect={(id) => sessionsStore.setCurrent(id)}
    locked={locked}
    canLock={canLock}
    onLock={lockNow}
    onChangePassword={openChangePassword}
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
  {:else if changeOpen}
    <div class="fixed inset-0 z-[110] bg-background/60 backdrop-blur-sm flex items-center justify-center p-6">
      <button
        type="button"
        class="absolute inset-0"
        aria-label={$t('common.close')}
        on:click={closeChangePassword}
      ></button>
      <Card className="relative w-[520px]">
        <CardHeader className="text-left">
          <CardTitle>{$t('security.change.title')}</CardTitle>
          <CardDescription>{$t('security.change.description')}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <form class="space-y-3" on:submit|preventDefault={submitChangePassword}>
            <Input type="password" bind:value={changeCurrent} placeholder={$t('security.change.currentPlaceholder')} disabled={changeWorking} />
            <Input type="password" bind:value={changeNext} placeholder={$t('security.change.newPlaceholder')} disabled={changeWorking} />
            <Input type="password" bind:value={changeConfirm} placeholder={$t('security.change.confirmPlaceholder')} disabled={changeWorking} />
            <div class="flex items-center justify-end gap-2">
              <Button type="button" variant="secondary" on:click={closeChangePassword} disabled={changeWorking}>
                {$t('common.cancel')}
              </Button>
              <Button type="submit" disabled={changeWorking}>
                {$t('security.change.submit')}
              </Button>
            </div>
          </form>
          {#if changeError}
            <div class="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-left">
              {changeErrorMessage}
            </div>
          {/if}
        </CardContent>
      </Card>
    </div>
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
              {unlockErrorMessage}
            </div>
          {/if}
        </CardContent>
      </Card>
    </div>
  {/if}
  <Toaster position="top-right" />
</div>
