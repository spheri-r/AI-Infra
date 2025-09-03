import axios from 'axios'

const API_BASE_URL = '/api/v1'

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor to add auth token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth-token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor to handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('auth-token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// Types
export interface User {
  id: number
  username: string
  email: string
  first_name: string
  last_name: string
  role: 'admin' | 'user'
  status: 'active' | 'inactive' | 'suspended'
  total_requests: number
  total_cost: number
  monthly_requests: number
  monthly_cost: number
  daily_request_limit: number
  monthly_request_limit: number
  daily_cost_limit: number
  monthly_cost_limit: number
  created_at: string
  updated_at: string
}

export interface Provider {
  id: number
  name: string
  type: 'openai' | 'anthropic' | 'google' | 'custom'
  status: 'active' | 'inactive' | 'maintenance'
  description: string
  base_url: string
  api_version: string
  default_rate_limit: number
  default_cost_per_token: number
  models?: LLMModel[]
  created_at: string
  updated_at: string
}

export interface LLMModel {
  id: number
  provider_id: number
  provider?: Provider
  name: string
  model_id: string
  description: string
  status: 'active' | 'inactive' | 'deprecated'
  max_tokens: number
  input_cost_per_1k: number
  output_cost_per_1k: number
  supports_streaming: boolean
  supports_functions: boolean
  supports_vision: boolean
  supports_embeddings: boolean
  total_requests: number
  total_cost: number
  created_at: string
  updated_at: string
}

export interface APIKey {
  id: number
  user_id: number
  provider_id: number
  provider?: Provider
  name: string
  status: 'active' | 'inactive' | 'revoked'
  daily_request_limit: number
  monthly_request_limit: number
  daily_cost_limit: number
  monthly_cost_limit: number
  total_requests: number
  total_cost: number
  daily_requests: number
  daily_cost: number
  monthly_requests: number
  monthly_cost: number
  last_used_at?: string
  created_at: string
  updated_at: string
}

export interface UsageAnalytics {
  total_requests: number
  successful_requests: number
  failed_requests: number
  total_cost: number
  total_tokens: number
  average_response_time: number
}

// Auth API
export const authAPI = {
  login: async (username: string, password: string) => {
    const response = await api.post('/auth/login', { username, password })
    return response.data
  },

  register: async (userData: {
    username: string
    email: string
    password: string
    first_name: string
    last_name: string
  }) => {
    const response = await api.post('/auth/register', userData)
    return response.data
  },
}

// Users API
export const usersAPI = {
  getCurrentUser: async (): Promise<User> => {
    const response = await api.get('/users/me')
    return response.data
  },

  getUsers: async (page = 1, limit = 10): Promise<{ users: User[]; total: number }> => {
    const response = await api.get(`/users?page=${page}&limit=${limit}`)
    return response.data
  },

  getUser: async (id: number): Promise<User> => {
    const response = await api.get(`/users/${id}`)
    return response.data
  },

  updateUser: async (id: number, data: Partial<User>): Promise<User> => {
    const response = await api.put(`/users/${id}`, data)
    return response.data
  },

  deleteUser: async (id: number): Promise<void> => {
    await api.delete(`/users/${id}`)
  },
}

// Providers API
export const providersAPI = {
  getProviders: async (): Promise<Provider[]> => {
    const response = await api.get('/providers')
    return response.data
  },

  getProvider: async (id: number): Promise<Provider> => {
    const response = await api.get(`/providers/${id}`)
    return response.data
  },

  createProvider: async (data: Omit<Provider, 'id' | 'created_at' | 'updated_at'>): Promise<Provider> => {
    const response = await api.post('/providers', data)
    return response.data
  },

  updateProvider: async (id: number, data: Partial<Provider>): Promise<Provider> => {
    const response = await api.put(`/providers/${id}`, data)
    return response.data
  },

  deleteProvider: async (id: number): Promise<void> => {
    await api.delete(`/providers/${id}`)
  },
}

// Models API
export const modelsAPI = {
  getModels: async (): Promise<LLMModel[]> => {
    const response = await api.get('/models')
    return response.data
  },

  getModel: async (id: number): Promise<LLMModel> => {
    const response = await api.get(`/models/${id}`)
    return response.data
  },

  updateModel: async (id: number, data: Partial<LLMModel>): Promise<LLMModel> => {
    const response = await api.put(`/models/${id}`, data)
    return response.data
  },

  deleteModel: async (id: number): Promise<void> => {
    await api.delete(`/models/${id}`)
  },
}

// API Keys API
export const apiKeysAPI = {
  getAPIKeys: async (): Promise<APIKey[]> => {
    const response = await api.get('/api-keys')
    return response.data
  },

  getAPIKey: async (id: number): Promise<APIKey> => {
    const response = await api.get(`/api-keys/${id}`)
    return response.data
  },

  createAPIKey: async (data: {
    provider_id: number
    name: string
    key_value: string
    daily_request_limit?: number
    monthly_request_limit?: number
    daily_cost_limit?: number
    monthly_cost_limit?: number
  }): Promise<APIKey> => {
    const response = await api.post('/api-keys', data)
    return response.data
  },

  updateAPIKey: async (id: number, data: Partial<APIKey>): Promise<APIKey> => {
    const response = await api.put(`/api-keys/${id}`, data)
    return response.data
  },

  deleteAPIKey: async (id: number): Promise<void> => {
    await api.delete(`/api-keys/${id}`)
  },
}

// Analytics API
export const analyticsAPI = {
  getOverview: async (): Promise<UsageAnalytics> => {
    const response = await api.get('/analytics/overview')
    return response.data
  },

  getUsageAnalytics: async (): Promise<UsageAnalytics> => {
    const response = await api.get('/analytics/usage')
    return response.data
  },

  getCostAnalytics: async (): Promise<UsageAnalytics> => {
    const response = await api.get('/analytics/costs')
    return response.data
  },
}

export default api 