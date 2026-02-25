import {writable} from 'svelte/store'
import {connect as connectService, disconnect as disconnectService} from '../../../lib/wails/connectionService.js'

function createStore() {
  const {subscribe, update, set} = writable({
    sessions: [],
    current: 'connections'
  })

  function upsert(sessionID, patch) {
    update(s => {
      const index = s.sessions.findIndex(x => x.sessionID === sessionID)
      if (index === -1) {
        return {
          ...s,
          sessions: [{sessionID, status: 'connecting', message: '', ...patch}, ...s.sessions]
        }
      }
      const next = [...s.sessions]
      next[index] = {...next[index], ...patch}
      return {...s, sessions: next}
    })
  }

  function remove(sessionID) {
    update(s => ({
      ...s,
      sessions: s.sessions.filter(x => x.sessionID !== sessionID),
      current: s.current === sessionID ? 'connections' : s.current
    }))
  }

  function setCurrent(id) {
    update(s => ({...s, current: id}))
  }

  function setSessionMeta(sessionID, meta) {
    if (!sessionID) return
    upsert(sessionID, meta)
  }

  async function connect(profileID) {
    const sessionID = await connectService(profileID)
    setCurrent(sessionID)
    return sessionID
  }

  async function disconnect(sessionID) {
    await disconnectService(sessionID)
  }

  function startListener() {
    const runtime = globalThis?.runtime
    if (!runtime || typeof runtime.EventsOnMultiple !== 'function') {
      return () => {}
    }

    return runtime.EventsOnMultiple('connection:status_changed', payload => {
      if (!payload || !payload.sessionID) return
      if (payload.status === 'disconnected') {
        remove(payload.sessionID)
        return
      }
      upsert(payload.sessionID, {
        status: payload.status,
        message: payload.message
      })
    }, -1)
  }

  return {subscribe, set, setCurrent, setSessionMeta, connect, disconnect, startListener}
}

export const sessionsStore = createStore()
