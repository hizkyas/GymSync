import { type LucideIcon } from 'lucide-react';

interface StatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: LucideIcon;
  color: 'emerald' | 'violet' | 'amber' | 'sky';
  trend?: { value: number; label: string };
}

const colorMap = {
  emerald: {
    bg: 'rgba(16, 185, 129, 0.12)',
    border: 'rgba(16, 185, 129, 0.25)',
    icon: '#10b981',
    glow: 'rgba(16, 185, 129, 0.3)',
    text: '#10b981',
  },
  violet: {
    bg: 'rgba(139, 92, 246, 0.12)',
    border: 'rgba(139, 92, 246, 0.25)',
    icon: '#8b5cf6',
    glow: 'rgba(139, 92, 246, 0.3)',
    text: '#8b5cf6',
  },
  amber: {
    bg: 'rgba(245, 158, 11, 0.12)',
    border: 'rgba(245, 158, 11, 0.25)',
    icon: '#f59e0b',
    glow: 'rgba(245, 158, 11, 0.3)',
    text: '#f59e0b',
  },
  sky: {
    bg: 'rgba(14, 165, 233, 0.12)',
    border: 'rgba(14, 165, 233, 0.25)',
    icon: '#0ea5e9',
    glow: 'rgba(14, 165, 233, 0.3)',
    text: '#0ea5e9',
  },
};

export const StatsCard: React.FC<StatsCardProps> = ({
  title, value, subtitle, icon: Icon, color, trend,
}) => {
  const c = colorMap[color];

  return (
    <div className="stats-card" style={{ borderColor: c.border, background: `linear-gradient(135deg, ${c.bg}, rgba(15,15,25,0.8))` }}>
      <div className="stats-card__header">
        <div className="stats-card__icon-wrap" style={{ background: c.bg, boxShadow: `0 0 20px ${c.glow}` }}>
          <Icon size={22} color={c.icon} />
        </div>
        {trend && (
          <span className={`stats-card__trend ${trend.value >= 0 ? 'stats-card__trend--up' : 'stats-card__trend--down'}`}>
            {trend.value >= 0 ? '↑' : '↓'} {Math.abs(trend.value)}% {trend.label}
          </span>
        )}
      </div>
      <div className="stats-card__value" style={{ color: c.text }}>{value}</div>
      <div className="stats-card__title">{title}</div>
      {subtitle && <div className="stats-card__subtitle">{subtitle}</div>}
    </div>
  );
};
