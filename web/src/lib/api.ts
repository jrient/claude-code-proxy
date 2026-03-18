const API_BASE = '/api'

function getToken(): string | null {
  return localStorage.getItem('admin_token')
}

export function setToken(token: string) {
  localStorage.setItem('admin_token', token)
}

export function clearToken() {
  localStorage.removeItem('admin_token')
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  if (res.status === 401) {
    clearToken()
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }

  return res.json()
}

// Auth
export const login = (password: string) =>
  request<{ token: string }>('/login', {
    method: 'POST',
    body: JSON.stringify({ password }),
  })

// Dashboard
export const getDashboard = () =>
  request<{
    total_requests: number
    total_tokens: number
    active_providers: number
    error_rate: number
    avg_latency: number
    total_cost: number
    requests_today: number
    tokens_today: number
  }>('/dashboard')

// Providers
export const getProviders = () =>
  request<Array<{
    id: number
    name: string
    type: string
    base_url: string
    api_key: string
    priority: number
    weight: number
    enabled: boolean
    health_status: string
    created_at: string
  }>>('/providers')

export const createProvider = (data: {
  name: string
  type: string
  base_url: string
  api_key: string
  priority: number
  weight: number
  models?: Array<{ source: string; target: string }>
}) =>
  request<{ id: number }>('/providers', {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const updateProvider = (id: number, data: any) =>
  request(`/providers/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })

export const deleteProvider = (id: number) =>
  request(`/providers/${id}`, { method: 'DELETE' })

// API Keys
export const getAPIKeys = () =>
  request<Array<{
    id: number
    name: string
    key_prefix: string
    enabled: boolean
    rate_limit: number
    daily_token_limit: number
    allowed_models: string
    created_at: string
  }>>('/apikeys')

export const createAPIKey = (data: {
  name: string
  rate_limit: number
  daily_token_limit: number
  allowed_models: string
}) =>
  request<{ key: string; info: any }>('/apikeys', {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const updateAPIKey = (id: number, data: any) =>
  request(`/apikeys/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })

export const deleteAPIKey = (id: number) =>
  request(`/apikeys/${id}`, { method: 'DELETE' })

// Stats
export const getTimeSeries = (params: {
  period?: string
  days?: number
  api_key_id?: number
  provider_id?: number
}) => {
  const query = new URLSearchParams()
  if (params.period) query.set('period', params.period)
  if (params.days) query.set('days', String(params.days))
  if (params.api_key_id) query.set('api_key_id', String(params.api_key_id))
  if (params.provider_id) query.set('provider_id', String(params.provider_id))
  return request<Array<{
    time: string
    requests: number
    tokens: number
    errors: number
  }>>(`/stats/timeseries?${query}`)
}

export const getModelStats = (days = 30) =>
  request<Array<{
    model: string
    requests: number
    prompt_tokens: number
    completion_tokens: number
    avg_latency: number
    estimated_cost: number
  }>>(`/stats/models?days=${days}`)

export const getRecentLogs = (limit = 50, offset = 0) =>
  request<{
    logs: Array<{
      id: number
      api_key_id: number
      provider_id: number
      model: string
      prompt_tokens: number
      completion_tokens: number
      total_tokens: number
      latency_ms: number
      status_code: number
      error_msg: string
      stream: boolean
      created_at: string
    }>
    total: number
  }>(`/stats/logs?limit=${limit}&offset=${offset}`)
