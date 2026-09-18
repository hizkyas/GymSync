import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getMembers, type User } from '../services/api';
import { Search, UserCheck, UserX, Users } from 'lucide-react';

const statusColors: Record<string, { bg: string; text: string; label: string }> = {
  active:   { bg: 'rgba(16,185,129,0.15)', text: '#10b981', label: 'Active' },
  trialing: { bg: 'rgba(14,165,233,0.15)', text: '#0ea5e9', label: 'Trial' },
  past_due: { bg: 'rgba(245,158,11,0.15)', text: '#f59e0b', label: 'Past Due' },
  canceled: { bg: 'rgba(239,68,68,0.15)',  text: '#ef4444', label: 'Canceled' },
  paused:   { bg: 'rgba(107,114,128,0.15)', text: '#9ca3af', label: 'Paused' },
};

function getInitials(name: string): string {
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2);
}

function formatDate(dateStr: string): string {
  return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(dateStr));
}

export const Members: React.FC = () => {
  const [search, setSearch] = useState('');

  const { data: members = [], isLoading, error } = useQuery<User[]>({
    queryKey: ['members'],
    queryFn: getMembers,
    refetchInterval: 30000,
  });

  const filtered = members.filter(m =>
    m.full_name.toLowerCase().includes(search.toLowerCase()) ||
    m.email.toLowerCase().includes(search.toLowerCase()),
  );

  if (error) {
    return (
      <div className="page">
        <div className="page-header"><h1 className="page-title">Members</h1></div>
        <div className="error-box">Failed to load members. Is the API running?</div>
      </div>
    );
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Members</h1>
          <p className="page-subtitle">{members.length} registered members</p>
        </div>
      </div>

      {/* Search */}
      <div className="search-bar">
        <Search size={18} className="search-bar__icon" />
        <input
          id="member-search"
          type="text"
          className="search-bar__input"
          placeholder="Search by name or email..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {/* Table */}
      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>Member</th>
              <th>Email</th>
              <th>QR Token</th>
              <th>Status</th>
              <th>Joined</th>
              <th>Active</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              [...Array(8)].map((_, i) => (
                <tr key={i}>
                  {[...Array(6)].map((_, j) => (
                    <td key={j}><div className="skeleton-cell" /></td>
                  ))}
                </tr>
              ))
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={6} className="table-empty">
                  <Users size={32} />
                  <p>No members found</p>
                </td>
              </tr>
            ) : (
              filtered.map((member) => {
                const s = statusColors[member.subscription_status ?? ''] ?? statusColors.canceled;
                return (
                  <tr key={member.id} className="data-table__row">
                    <td>
                      <div className="member-cell">
                        <div className="member-cell__avatar">
                          {member.avatar_url
                            ? <img src={member.avatar_url} alt={member.full_name} />
                            : <span>{getInitials(member.full_name)}</span>
                          }
                        </div>
                        <span className="member-cell__name">{member.full_name}</span>
                      </div>
                    </td>
                    <td className="table-cell-muted">{member.email}</td>
                    <td>
                      <code className="qr-badge">{member.qr_token.slice(0, 8)}…</code>
                    </td>
                    <td>
                      <span className="status-badge" style={{ background: s.bg, color: s.text }}>
                        {s.label}
                      </span>
                    </td>
                    <td className="table-cell-muted">{formatDate(member.created_at)}</td>
                    <td>
                      {member.is_active
                        ? <UserCheck size={18} color="#10b981" />
                        : <UserX size={18} color="#ef4444" />
                      }
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};
