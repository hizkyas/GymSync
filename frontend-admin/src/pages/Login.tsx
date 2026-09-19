import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { login, type AuthResponse } from '../services/api';
import { Loader2, Dumbbell } from 'lucide-react';

interface LoginPageProps {
  onLogin: (token: string) => void;
}

export const Login: React.FC<LoginPageProps> = ({ onLogin }) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const mutation = useMutation<AuthResponse, Error, { email: string; password: string }>({
    mutationFn: login,
    onSuccess: (data) => {
      localStorage.setItem('gymfusion_token', data.token);
      onLogin(data.token);
    },
    onError: (err: any) => {
      setError(err.response?.data?.error || 'Login failed. Check your credentials.');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    mutation.mutate({ email, password });
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-card__logo">
          <Dumbbell size={36} color="#8b5cf6" />
        </div>
        <h1 className="login-card__title">Gymfusion</h1>
        <p className="login-card__subtitle">Admin & Front-Desk Portal</p>

        <form onSubmit={handleSubmit} className="login-form">
          {error && <div className="login-form__error">{error}</div>}
          <div className="form-group">
            <label htmlFor="login-email" className="form-label">Email</label>
            <input
              id="login-email"
              type="email"
              className="form-input"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@gym.com"
              required
              autoFocus
            />
          </div>
          <div className="form-group">
            <label htmlFor="login-password" className="form-label">Password</label>
            <input
              id="login-password"
              type="password"
              className="form-input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
            />
          </div>
          <button
            id="login-submit"
            type="submit"
            className="login-btn"
            disabled={mutation.isPending}
          >
            {mutation.isPending ? <Loader2 size={18} className="spin" /> : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
};
