// All API calls use relative paths so the SPA works under any kernel pathPrefix.
// The admin SPA is served from `<prefix>/admin/`, so `../api/...` resolves to
// `<prefix>/api/...`.
const API_BASE = '../api'

export class UnauthenticatedError extends Error {
  constructor() {
    super('UNAUTHENTICATED')
    this.name = 'UnauthenticatedError'
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method, credentials: 'include' }
  if (body !== undefined) {
    init.headers = { 'Content-Type': 'application/json' }
    init.body = JSON.stringify(body)
  }
  const res = await fetch(API_BASE + path, init)
  if (res.status === 401) throw new UnauthenticatedError()
  if (!res.ok) {
    let msg = `${method} ${path} ${res.status}`
    try {
      const e: any = await res.json()
      if (e?.error) msg += ': ' + e.error
    } catch {
      // ignore
    }
    throw new Error(msg)
  }
  return (await res.json()) as T
}

export interface MetaField {
  name: string
  type: string
  required?: boolean
  default?: unknown
  max_len?: number
  target?: string
  through?: string
}

export interface MetaCollection {
  name: string
  display: string
  fields: MetaField[]
  acl?: Record<string, string[]>
}

export interface User {
  id: string
  name: string
}

export interface ListResult<T = Record<string, unknown>> {
  data: T[]
  total: number
  page: number
  size: number
}

export const api = {
  whoami: () => request<{ data: User }>('GET', '/meta/whoami'),
  collections: () => request<{ data: MetaCollection[] }>('GET', '/meta/collections'),
  list: (name: string, page: number, size: number) =>
    request<ListResult>('GET', `/${name}?page=${page}&size=${size}`),
  get: (name: string, id: number | string) =>
    request<{ data: Record<string, unknown> }>('GET', `/${name}/${id}`),
  create: (name: string, body: Record<string, unknown>) =>
    request<{ data: Record<string, unknown> }>('POST', `/${name}`, body),
  update: (name: string, id: number | string, body: Record<string, unknown>) =>
    request<{ data: Record<string, unknown> }>('PATCH', `/${name}/${id}`, body),
  remove: (name: string, id: number | string) =>
    request<{ data: unknown }>('DELETE', `/${name}/${id}`),
  logout: () => request<unknown>('POST', '/auth/logout').catch(() => undefined),
  localLogin: (username: string, password: string) =>
    request<{ data: User }>('POST', '/auth/local/login', { username, password }),
}

export function loginURL(returnTo: string): string {
  return `../api/auth/login?return=${encodeURIComponent(returnTo)}`
}
