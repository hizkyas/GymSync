import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const apiClient = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  headers: { 'Content-Type': 'application/json' },
});

// Attach JWT token on every request
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('gym_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Redirect to login on 401
apiClient.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('gym_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  },
);

// ─── Auth ─────────────────────────────────────────────────────────────────────
export interface LoginPayload { email: string; password: string }
export interface RegisterPayload { email: string; password: string; full_name: string; phone?: string }
export interface AuthResponse { token: string; user: User }

export async function login(payload: LoginPayload): Promise<AuthResponse> {
  const { data } = await apiClient.post<AuthResponse>('/auth/login', payload);
  return data;
}

export async function register(payload: RegisterPayload): Promise<AuthResponse> {
  const { data } = await apiClient.post<AuthResponse>('/auth/register', payload);
  return data;
}

export async function getMe(): Promise<User> {
  const { data } = await apiClient.get<User>('/auth/me');
  return data;
}

// ─── Check-in ─────────────────────────────────────────────────────────────────
export interface CheckInPayload { qr_token: string; notes?: string }
export interface CheckInResponse {
  access_granted: boolean;
  denial_reason?: string;
  member_name: string;
  member_email: string;
  avatar_url?: string;
  checked_in_at: string;
  check_in_id: string;
}

export async function checkIn(payload: CheckInPayload): Promise<CheckInResponse> {
  const { data } = await apiClient.post<CheckInResponse>('/checkin/', payload);
  return data;
}

// ─── Dashboard ────────────────────────────────────────────────────────────────
export interface Stats {
  active_members: number;
  total_members: number;
  today_check_ins: number;
  monthly_revenue_cents: number;
}

export interface RecentCheckIn {
  check_in_id: string;
  member_name: string;
  member_email: string;
  avatar_url?: string;
  access_granted: boolean;
  denial_reason?: string;
  subscription_status: string;
  checked_in_at: string;
}

export interface User {
  id: string;
  email: string;
  full_name: string;
  role: 'admin' | 'trainer' | 'member';
  phone?: string;
  avatar_url?: string;
  qr_token: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  subscription_status?: string;
}

export async function getStats(): Promise<Stats> {
  const { data } = await apiClient.get<Stats>('/stats');
  return data;
}

export async function getRecentCheckIns(limit = 50): Promise<RecentCheckIn[]> {
  const { data } = await apiClient.get<RecentCheckIn[]>(`/checkins/recent?limit=${limit}`);
  return data;
}

export async function getMembers(): Promise<User[]> {
  const { data } = await apiClient.get<User[]>('/members');
  return data;
}
