import {writable} from 'svelte/store'
import {listConnections} from '../../../lib/wails/connectionService.js'

export const connectionsStore = writable({
  items: [],
  loading: false,
  error: null
})

export async function refreshConnections() {
  connectionsStore.update(s => ({...s, loading: true, error: null}))
  try {
    const items = await listConnections()
    connectionsStore.set({items, loading: false, error: null})
  } catch (error) {
    connectionsStore.update(s => ({...s, loading: false, error}))
    throw error
  }
}
