export type User = {
  id: number
  email: string
  name: string
}

export type PublicBot = {
  bot_name: string
  welcome_message: string
  bot_active: boolean
  configured: boolean
}

export type ChatSource = {
  doc_id: number
  filename: string
  snippet: string
  score: number
}

export type Document = {
  id: number
  filename: string
  file_type: string
  size_bytes: number
  status: 'pending' | 'processing' | 'ready' | 'failed'
  error_msg?: string
  chunk_count: number
  created_at?: string
  updated_at?: string
}

export type Settings = {
  bot_name: string
  welcome_message: string
  bot_active: boolean
  temperature: number
  max_tokens: number
  top_k: number
  min_score: number
  updated_at?: string
}

export type Conversation = {
  id: number
  title: string
  message_count: number
  created_at?: string
  updated_at?: string
}

export type ChatMessage = {
  id: number
  conversation_id: number
  role: 'user' | 'bot'
  content: string
  sources?: ChatSource[]
  latency_ms?: number
  token_usage?: number
  created_at?: string
}

export type Overview = {
  total_conversations: number
  total_messages: number
  total_bot_messages: number
  avg_latency_ms: number
  daily: { date: string; chats: number }[]
}

export type TopQuestion = {
  question: string
  count: number
}
