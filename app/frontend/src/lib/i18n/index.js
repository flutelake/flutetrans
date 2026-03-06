import {derived, writable} from 'svelte/store'
import {defaultLocale, messages, supportedLocales} from './messages.js'

const STORAGE_KEY_OVERRIDE = 'fluteTrans.localeOverride'
const STORAGE_KEY_LEGACY = 'fluteTrans.locale'

function safeGetStorageLocale() {
  try {
    const storage = globalThis?.localStorage
    const fromOverride = storage?.getItem?.(STORAGE_KEY_OVERRIDE)
    if (fromOverride) return String(fromOverride)

    const legacy = storage?.getItem?.(STORAGE_KEY_LEGACY)
    if (legacy) {
      try {
        storage?.setItem?.(STORAGE_KEY_OVERRIDE, String(legacy))
        storage?.removeItem?.(STORAGE_KEY_LEGACY)
      } catch {}
      return String(legacy)
    }

    return ''
  } catch {
    return ''
  }
}

function detectBrowserLocale() {
  try {
    const v = globalThis?.navigator?.language ?? ''
    return String(v).toLowerCase()
  } catch {
    return ''
  }
}

function normalizeLocale(input) {
  const v = String(input ?? '').toLowerCase()
  if (v.startsWith('zh')) return 'zh'
  if (v.startsWith('en')) return 'en'
  return ''
}

function detectInitialLocale() {
  const fromStorage = normalizeLocale(safeGetStorageLocale())
  if (fromStorage) return fromStorage
  const fromBrowser = normalizeLocale(detectBrowserLocale())
  if (fromBrowser) return fromBrowser
  return defaultLocale
}

export const locale = writable(detectInitialLocale())
export const localeOptions = supportedLocales

export function setLocale(next) {
  const v = normalizeLocale(next) || defaultLocale
  locale.set(v)
  try {
    globalThis?.localStorage?.setItem?.(STORAGE_KEY_OVERRIDE, v)
  } catch {}
}

function formatMessage(template, vars) {
  const text = String(template ?? '')
  const values = vars && typeof vars === 'object' ? vars : {}
  return text.replace(/\{(\w+)\}/g, (_m, key) => {
    const v = values[key]
    return v == null ? '' : String(v)
  })
}

function getByPath(obj, path) {
  const parts = String(path ?? '').split('.').filter(Boolean)
  let cur = obj
  for (const p of parts) {
    if (!cur || typeof cur !== 'object') return undefined
    cur = cur[p]
  }
  return cur
}

export const t = derived(locale, $locale => {
  return (key, vars) => {
    const localeDict = messages[$locale] ?? {}
    const enDict = messages.en ?? {}
    const raw = getByPath(localeDict, key) ?? getByPath(enDict, key)
    if (raw == null) return String(key ?? '')
    return formatMessage(raw, vars)
  }
})

locale.subscribe(next => {
  const v = normalizeLocale(next) || defaultLocale
  try {
    const doc = globalThis?.document
    if (doc?.documentElement) {
      doc.documentElement.lang = v === 'zh' ? 'zh-CN' : 'en'
    }
  } catch {}
})
