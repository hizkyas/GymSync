import { useState, useEffect } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route, NavLink, Navigate } from 'react-router-dom';
import { Dashboard } from './pages/Dashboard';
import { Members } from './pages/Members';
import { Login } from './pages/Login';
import { Dumbbell, LayoutDashboard, Users, LogOut, Activity } from 'lucide-react';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
});

function AppLayout({ onLogout }: { onLogout: () => void }) {
  return (
    <div className="app-layout">
      {/* Sidebar */}
      <aside className="sidebar">
        <div className="sidebar__logo">
          <Dumbbell size={28} color="#8b5cf6" />
          <span>Gymfusion</span>
        </div>
        <nav className="sidebar__nav">
          <NavLink to="/dashboard" className={({ isActive }) => `sidebar__link ${isActive ? 'sidebar__link--active' : ''}`}>
            <LayoutDashboard size={18} />
            Dashboard
          </NavLink>
          <NavLink to="/members" className={({ isActive }) => `sidebar__link ${isActive ? 'sidebar__link--active' : ''}`}>
            <Users size={18} />
            Members
          </NavLink>
          <NavLink to="/activity" className={({ isActive }) => `sidebar__link ${isActive ? 'sidebar__link--active' : ''}`}>
            <Activity size={18} />
            Activity
          </NavLink>
        </nav>
        <div className="sidebar__footer">
          <button id="logout-btn" className="sidebar__logout" onClick={onLogout}>
            <LogOut size={16} />
            Sign Out
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="main-content">
        <Routes>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/members" element={<Members />} />
          <Route path="/activity" element={<Dashboard />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </main>
    </div>
  );
}

export default function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem('gymfusion_token');
    setIsAuthenticated(!!token);
  }, []);

  const handleLogin = (_token: string) => setIsAuthenticated(true);
  const handleLogout = () => {
    localStorage.removeItem('gymfusion_token');
    setIsAuthenticated(false);
  };

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        {isAuthenticated ? (
          <AppLayout onLogout={handleLogout} />
        ) : (
          <Routes>
            <Route path="/login" element={<Login onLogin={handleLogin} />} />
            <Route path="*" element={<Navigate to="/login" replace />} />
          </Routes>
        )}
      </BrowserRouter>
    </QueryClientProvider>
  );
}
