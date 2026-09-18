import { useQuery } from '@tanstack/react-query';
import { getStats, type Stats } from '../services/api';
import { StatsCard } from '../components/StatsCard';
import { CheckinFeed } from '../components/CheckinFeed';
import { QRScanner } from '../components/QRScanner';
import { Users, TrendingUp, Activity, DollarSign } from 'lucide-react';

function formatCurrency(cents: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD', minimumFractionDigits: 0,
  }).format(cents / 100);
}

export const Dashboard: React.FC = () => {
  const { data: stats, isLoading } = useQuery<Stats>({
    queryKey: ['stats'],
    queryFn: getStats,
    refetchInterval: 30000,
  });

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1 className="page-title">Dashboard</h1>
          <p className="page-subtitle">Real-time gym activity overview</p>
        </div>
        <div className="page-header__badge">
          <span className="live-dot" />
          Live
        </div>
      </div>

      {/* Stats Grid */}
      <div className="stats-grid">
        <StatsCard
          title="Active Members"
          value={isLoading ? '—' : (stats?.active_members ?? 0)}
          subtitle={`of ${stats?.total_members ?? 0} total`}
          icon={Users}
          color="emerald"
        />
        <StatsCard
          title="Monthly Revenue"
          value={isLoading ? '—' : formatCurrency(stats?.monthly_revenue_cents ?? 0)}
          subtitle="from active subscriptions"
          icon={DollarSign}
          color="violet"
        />
        <StatsCard
          title="Today's Check-ins"
          value={isLoading ? '—' : (stats?.today_check_ins ?? 0)}
          subtitle="since midnight"
          icon={Activity}
          color="amber"
        />
        <StatsCard
          title="Occupancy"
          value={isLoading ? '—' : `${stats?.active_members ?? 0}`}
          subtitle="currently active"
          icon={TrendingUp}
          color="sky"
        />
      </div>

      {/* Main Content */}
      <div className="dashboard-grid">
        <div className="dashboard-grid__feed">
          <CheckinFeed />
        </div>
        <div className="dashboard-grid__sidebar">
          <QRScanner />
        </div>
      </div>
    </div>
  );
};
