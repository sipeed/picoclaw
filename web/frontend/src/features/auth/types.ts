export interface AuthStatus {
  enabled: boolean
  configured: boolean
  logged_in: boolean
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  success: boolean
  message?: string
}

export interface SetupRequest {
  username: string
  password: string
}

export interface SetupResponse {
  success: boolean
  message?: string
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}
