import {writable} from 'svelte/store'

export const toasts = writable([])

let counter = 0

export function pushToast({type = 'info', title = '', message = '', durationMs = 3000} = {}) {
  const id = String(++counter)
  const toast = {id, type, title, message}
  toasts.update(items => [toast, ...items].slice(0, 5))
  if (durationMs > 0) {
    setTimeout(() => {
      toasts.update(items => items.filter(t => t.id !== id))
    }, durationMs)
  }
  return id
}

export function success(title, message) {
  return pushToast({type: 'success', title, message})
}

export function error(title, message) {
  return pushToast({type: 'error', title, message, durationMs: 5000})
}

