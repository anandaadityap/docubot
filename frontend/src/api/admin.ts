import { api } from './client'
import type { Conversation, ChatMessage, Document, Overview, Settings, TopQuestion, User } from './types'

export const authApi = {
  register: (name: string, email: string, password: string) =>
    api<User>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify({ name, email, password }),
    }),
  login: (email: string, password: string) =>
    api<{ token: string; user: User }>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  me: () => api<User>('/api/v1/auth/me'),
}

export const documentsApi = {
  list: () => api<Document[]>('/api/v1/documents'),
  get: (id: number) => api<Document & { chunks: { position: number; content: string; token_count: number }[] }>(`/api/v1/documents/${id}`),
  upload: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return api<{ id: number; filename: string; status: string }>('/api/v1/documents', { method: 'POST', body: fd })
  },
  remove: (id: number) => api<void>(`/api/v1/documents/${id}`, { method: 'DELETE' }),
  reprocess: (id: number) => api<{ status: string }>(`/api/v1/documents/${id}/reprocess`, { method: 'POST' }),
}

export const settingsApi = {
  get: () => api<Settings>('/api/v1/settings'),
  put: (body: Settings) => api<Settings>('/api/v1/settings', { method: 'PUT', body: JSON.stringify(body) }),
}

export const conversationsApi = {
  list: (page = 1, limit = 20) =>
    api<{ items: Conversation[]; total: number; page: number }>(`/api/v1/conversations?page=${page}&limit=${limit}`),
  get: (id: number) => api<Conversation & { messages: ChatMessage[] }>(`/api/v1/conversations/${id}`),
}

export const analyticsApi = {
  overview: () => api<Overview>('/api/v1/analytics/overview'),
  topQuestions: (limit = 10) => api<TopQuestion[]>(`/api/v1/analytics/top-questions?limit=${limit}`),
}
