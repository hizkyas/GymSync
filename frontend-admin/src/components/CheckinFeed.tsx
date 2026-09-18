import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { getRecentCheckIns, type RecentCheckIn } from '../services/api';
import { CheckCircle, XCircle, Clock } from 'lucide-react';

function timeAgo(dateStr: string): string {
  const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

function getInitials(name: string): string {
  return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2);
}

const statusColors: Record<string, string> = {
  active: '#10b981',
  trialing: '#0ea5e9',
  past_due: '#f59e0b',
  canceled: '#ef4444',
  paused: '#6b7280',
};

export const CheckinFeed: React.FC = () => {
  const { data: checkIns = [], isLoading, error } = useQuery({
    queryKey: ['recent-checkins'],
    queryFn: () => getRecentCheckIns(30),
    refetchInterval: 5000, // poll every 5 seconds for live updates
    staleTime: 0,
  });

  if (isLoading) {
    return (
      <div className="checkin-feed">
        <div className="checkin-feed__header">
          <div className="checkin-feed__title">
            <span className="live-dot" />
            Live Check-ins
          </div>
        </div>
        <div className="checkin-feed__loading">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="checkin-skeleton" style={{ animationDelay: `${i * 0.1}s` }} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="checkin-feed">
        <div className="checkin-feed__error">Failed to load check-in feed. Is the API running?</div>
      </div>
    );
  }

  return (
    <div className="checkin-feed">
      <div className="checkin-feed__header">
        <div className="checkin-feed__title">
          <span className="live-dot" />
          Live Check-ins
        </div>
        <span className="checkin-feed__count">{checkIns.length} recent</span>
      </div>
      <div className="checkin-feed__list">
        {checkIns.length === 0 ? (
          <div className="checkin-feed__empty">
            <Clock size={32} color="#4b5563" />
            <p>No check-ins yet today</p>
          </div>
        ) : (
          checkIns.map((ci: RecentCheckIn) => (
            <CheckinRow key={ci.check_in_id} ci={ci} />
          ))
        )}
      </div>
    </div>
  );
};

const CheckinRow: React.FC<{ ci: RecentCheckIn }> = ({ ci }) => {
  const statusColor = statusColors[ci.subscription_status] || '#6b7280';

  return (
    <div className={`checkin-row ${ci.access_granted ? 'checkin-row--granted' : 'checkin-row--denied'}`}>
      <div className="checkin-row__avatar">
        {ci.avatar_url
          ? <img src={ci.avatar_url} alt={ci.member_name} />
          : <span>{getInitials(ci.member_name)}</span>
        }
      </div>
      <div className="checkin-row__info">
        <span className="checkin-row__name">{ci.member_name}</span>
        <span className="checkin-row__email">{ci.member_email}</span>
        {ci.denial_reason && (
          <span className="checkin-row__reason">{ci.denial_reason}</span>
        )}
      </div>
      <div className="checkin-row__meta">
        <span className="checkin-row__status" style={{ color: statusColor }}>
          {ci.subscription_status?.replace('_', ' ')}
        </span>
        <span className="checkin-row__time">{timeAgo(ci.checked_in_at)}</span>
      </div>
      <div className="checkin-row__badge">
        {ci.access_granted
          ? <CheckCircle size={20} color="#10b981" />
          : <XCircle size={20} color="#ef4444" />
        }
      </div>
    </div>
  );
};
