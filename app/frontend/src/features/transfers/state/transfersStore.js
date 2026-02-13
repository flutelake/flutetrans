import {writable} from 'svelte/store'

import {getTransfers} from '../../../lib/wails/connectionService.js'

function createStore() {
  const {subscribe, set, update} = writable({
    items: [],
    loading: false,
    error: null
  })

  async function refresh() {
    update(s => ({...s, loading: true, error: null}))
    try {
      const items = await getTransfers()
      set({items: Array.isArray(items) ? items : [], loading: false, error: null})
    } catch (error) {
      update(s => ({...s, loading: false, error}))
      throw error
    }
  }

  function startListener() {
    const runtime = globalThis?.runtime
    if (!runtime || typeof runtime.EventsOnMultiple !== 'function') {
      return () => {}
    }

    return runtime.EventsOnMultiple('transfer:updated', payload => {
      const items = payload?.items
      if (!Array.isArray(items)) return
      update(s => ({...s, items}))
    }, -1)
  }

  return {subscribe, refresh, startListener}
}

export const transfersStore = createStore()

