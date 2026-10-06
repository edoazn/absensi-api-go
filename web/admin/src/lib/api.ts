const ACCESS_KEY = "absensi.access"
const REFRESH_KEY = "absensi.refresh"

export class ApiError extends Error {
  status: number
  errors?: Record<string, string>

  constructor(status: number, message: string, errors?: Record<string, string>) {
    super(message)
    this.status = status
    this.errors = errors
  }
}

export interface Envelope<T = unknown> {
  success: boolean
  data?: T
  message?: string
  errors?: Record<string, string>
}

export interface UserProfile {
  id: number
  name: string
  identity_number: string
  email: string | null
  role: "admin" | "mahasiswa"
}

interface AuthData {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  user: UserProfile
}

export const tokens = {
  get access() {
    return localStorage.getItem(ACCESS_KEY)
  },
  get refresh() {
    return localStorage.getItem(REFRESH_KEY)
  },
  save(access: string, refresh: string) {
    localStorage.setItem(ACCESS_KEY, access)
    localStorage.setItem(REFRESH_KEY, refresh)
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}

export async function refreshTokens(): Promise<boolean> {
  const refreshToken = tokens.refresh
  if (!refreshToken) return false
  try {
    const res = await fetch("/api/v1/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) return false
    const env = (await res.json()) as Envelope<AuthData>
    if (!env.success || !env.data) return false
    tokens.save(env.data.access_token, env.data.refresh_token)
    return true
  } catch {
    return false
  }
}

export async function api<T = unknown>(
  path: string,
  options: RequestInit & { retry?: boolean } = {},
): Promise<Envelope<T>> {
  const { retry, ...init } = options
  const headers = new Headers(init.headers)
  if (!(init.body instanceof FormData) && init.body != null) {
    headers.set("Content-Type", "application/json")
  }
  if (tokens.access) {
    headers.set("Authorization", `Bearer ${tokens.access}`)
  }

  let res = await fetch(`/api/v1${path}`, { ...init, headers })

  if (res.status === 401 && !retry && tokens.refresh && path !== "/auth/login") {
    if (await refreshTokens()) {
      return api<T>(path, { ...options, retry: true })
    }
    tokens.clear()
  }

  const env = (await res.json().catch(() => ({}))) as Envelope<T>
  if (!res.ok || env.success === false) {
    throw new ApiError(res.status, env.message ?? "Terjadi kesalahan", env.errors)
  }
  return env
}

export async function apiBlob(path: string): Promise<Blob> {
  const headers = new Headers()
  if (tokens.access) {
    headers.set("Authorization", `Bearer ${tokens.access}`)
  }
  let res = await fetch(`/api/v1${path}`, { headers })

  if (res.status === 401 && (await refreshTokens())) {
    headers.set("Authorization", `Bearer ${tokens.access}`)
    res = await fetch(`/api/v1${path}`, { headers })
  }

  if (!res.ok) {
    throw new ApiError(res.status, "Gagal mengunduh berkas")
  }
  return res.blob()
}

export async function login(identityNumber: string, password: string): Promise<UserProfile> {
  const env = await api<AuthData>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ identity_number: identityNumber, password }),
  })
  if (!env.data) throw new ApiError(500, "Respon login tidak valid")
  tokens.save(env.data.access_token, env.data.refresh_token)
  return env.data.user
}

export async function fetchMe(): Promise<UserProfile | null> {
  if (!tokens.access) return null
  try {
    const env = await api<UserProfile>("/me")
    return env.data ?? null
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return null
    throw err
  }
}

export async function logout(): Promise<void> {
  try {
    await api("/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refresh_token: tokens.refresh }),
    })
  } finally {
    tokens.clear()
  }
}
