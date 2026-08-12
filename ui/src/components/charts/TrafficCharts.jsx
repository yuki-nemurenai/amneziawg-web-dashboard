import React, { useMemo, useState } from 'react';
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { formatBytes, formatChartTime } from '../../utils/traffic';

const C = {
  download: '#10B981',
  upload:   '#0EA5E9',
  grid:     'rgba(148,163,184,0.14)',
  tick:     '#94A3B8',
};

const PERIODS = [
  { id: 'all', label: 'All' },
  { id: '1h',  label: '1H' },
  { id: '6h',  label: '6H' },
  { id: '24h', label: '24H' },
];

function filterByPeriod(series, period) {
  if (!series || !series.length) return [];
  if (period === 'all') return series;
  const now = Math.floor(Date.now() / 1000);
  const windows = { '1h': 3600, '6h': 21600, '24h': 86400 };
  const cutoff = now - (windows[period] || 3600);
  return series.filter((p) => (p.timestamp || 0) >= cutoff);
}

function ChartTooltip({ active, payload, label }) {
  if (!active || !payload || !payload.length) return null;
  return (
    <div className="chart-tooltip">
      <p className="chart-tooltip-label">{label || ''}</p>
      <div className="chart-tooltip-rows">
        {payload.map((entry) => (
          <div key={entry.dataKey || entry.name} className="chart-tooltip-row">
            <span className="chart-tooltip-dot" style={{ background: entry.color }} />
            <span className="chart-tooltip-name">{entry.name}</span>
            <span className="chart-tooltip-value">{formatBytes(entry.value)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function EmptyState({ message = 'Collecting traffic samples…' }) {
  return (
    <div className="chart-empty">
      <div className="chart-empty-ring" />
      <p className="chart-empty-text">{message}</p>
      <p className="chart-empty-sub">Charts update automatically as traffic flows</p>
    </div>
  );
}

function PeriodSelector({ value, onChange }) {
  return (
    <div className="chart-period-selector">
      {PERIODS.map((p) => (
        <button
          key={p.id}
          type="button"
          onClick={() => onChange(p.id)}
          className={`chart-period-btn${value === p.id ? ' active' : ''}`}
        >
          {p.label}
        </button>
      ))}
    </div>
  );
}

function ChartCard({ title, subtitle, className = '', actions, children }) {
  return (
    <div className={`chart-card ${className}`}>
      <div className="chart-card-header">
        <div>
          <h3 className="chart-card-title">{title}</h3>
          {subtitle && <p className="chart-card-subtitle">{subtitle}</p>}
        </div>
        {actions && <div className="chart-card-actions">{actions}</div>}
      </div>
      <div className="chart-card-body">{children}</div>
    </div>
  );
}

function TrafficOverview({ series, hasData }) {
  const [period, setPeriod] = useState('all');
  const [filter, setFilter] = useState('all');

  const data = useMemo(() => filterByPeriod(series, period), [series, period]);
  const showDL = filter === 'all' || filter === 'download';
  const showUL = filter === 'all' || filter === 'upload';

  const now = Math.floor(Date.now() / 1000);
  const windows = { '1h': 3600, '6h': 21600, '24h': 86400 };
  const xDomain = period === 'all' 
    ? ['dataMin', 'dataMax'] 
    : [now - (windows[period] || 3600), now];

  return (
    <ChartCard
      title="Traffic Overview"
      subtitle="Bandwidth activity over time"
      actions={
        <div className="chart-card-actions-row">
          <div className="chart-filter-tabs">
            {['all', 'download', 'upload'].map((f) => (
              <button
                key={f}
                type="button"
                onClick={() => setFilter(f)}
                className={`chart-filter-tab${filter === f ? ' active' : ''}`}
              >
                {f === 'all' ? 'All' : f.charAt(0).toUpperCase() + f.slice(1)}
              </button>
            ))}
          </div>
          <PeriodSelector value={period} onChange={setPeriod} />
        </div>
      }
    >
      <div className="chart-area-wrap" style={{ position: 'relative', minHeight: 260 }}>
        {!hasData && <EmptyState />}
        {hasData && data.length === 0 && <EmptyState message="No traffic data for this period" />}
        {data.length > 0 && (
          <ResponsiveContainer width="100%" height={260}>
            <AreaChart data={data} margin={{ top: 10, right: 4, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="gDL" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={C.download} stopOpacity={0.3} />
                  <stop offset="100%" stopColor={C.download} stopOpacity={0.02} />
                </linearGradient>
                <linearGradient id="gUL" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={C.upload} stopOpacity={0.3} />
                  <stop offset="100%" stopColor={C.upload} stopOpacity={0.02} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={C.grid} />
              <XAxis
                type="number"
                dataKey="timestamp"
                domain={xDomain}
                scale="time"
                tickFormatter={(v) => formatChartTime(v)}
                tickLine={false}
                axisLine={false}
                tickMargin={10}
                minTickGap={40}
                tick={{ fill: C.tick, fontSize: 11, fontWeight: 600 }}
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                width={58}
                tickFormatter={(v) => formatBytes(v, 0)}
                tick={{ fill: C.tick, fontSize: 11, fontWeight: 600 }}
              />
              <Tooltip content={<ChartTooltip />} cursor={{ stroke: C.download, strokeDasharray: '4 4', strokeOpacity: 0.4 }} />
              {showDL && (
                <Area
                  type="monotone"
                  dataKey="download"
                  name="Download"
                  stroke={C.download}
                  strokeWidth={2.5}
                  fill="url(#gDL)"
                  dot={false}
                  activeDot={{ r: 5, strokeWidth: 2, stroke: '#fff' }}
                />
              )}
              {showUL && (
                <Area
                  type="monotone"
                  dataKey="upload"
                  name="Upload"
                  stroke={C.upload}
                  strokeWidth={2.5}
                  fill="url(#gUL)"
                  dot={false}
                  activeDot={{ r: 5, strokeWidth: 2, stroke: '#fff' }}
                />
              )}
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>

      <div className="chart-legend-row">
        <div className="chart-legend-pill">
          <span className="chart-legend-dot" style={{ background: C.download }} />
          <span>Download</span>
        </div>
        <div className="chart-legend-pill">
          <span className="chart-legend-dot" style={{ background: C.upload }} />
          <span>Upload</span>
        </div>
      </div>
    </ChartCard>
  );
}

function BandwidthBars({ series, hasData }) {
  const [period, setPeriod] = useState('all');
  const data = useMemo(() => filterByPeriod(series, period), [series, period]);

  const now = Math.floor(Date.now() / 1000);
  const windows = { '1h': 3600, '6h': 21600, '24h': 86400 };
  const xDomain = period === 'all' 
    ? ['dataMin', 'dataMax'] 
    : [now - (windows[period] || 3600), now];

  return (
    <ChartCard
      title="Bandwidth"
      subtitle="Per-interval totals"
      actions={<PeriodSelector value={period} onChange={setPeriod} />}
    >
      <div style={{ position: 'relative', minHeight: 200 }}>
        {!hasData && <EmptyState message="No bandwidth data yet" />}
        {hasData && data.length === 0 && <EmptyState message="No traffic data for this period" />}
        {data.length > 0 && (
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={data} barGap={2} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="barDL" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={C.download} stopOpacity={0.9} />
                  <stop offset="100%" stopColor={C.download} stopOpacity={0.5} />
                </linearGradient>
                <linearGradient id="barUL" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={C.upload} stopOpacity={0.9} />
                  <stop offset="100%" stopColor={C.upload} stopOpacity={0.5} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={C.grid} />
              <XAxis
                type="number"
                dataKey="timestamp"
                domain={xDomain}
                scale="time"
                tickFormatter={(v) => formatChartTime(v)}
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                minTickGap={40}
                tick={{ fill: C.tick, fontSize: 10, fontWeight: 600 }}
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                tickMargin={6}
                width={54}
                tickFormatter={(v) => formatBytes(v, 0)}
                tick={{ fill: C.tick, fontSize: 10, fontWeight: 600 }}
              />
              <Tooltip content={<ChartTooltip />} cursor={{ fill: 'rgba(148,163,184,0.08)' }} />
              <Bar dataKey="download" name="Download" fill="url(#barDL)" radius={[6, 6, 0, 0]} maxBarSize={16} />
              <Bar dataKey="upload" name="Upload" fill="url(#barUL)" radius={[6, 6, 0, 0]} maxBarSize={16} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>
    </ChartCard>
  );
}

export default function TrafficCharts({ series = [], hasData = false }) {
  const safeSeries = Array.isArray(series) ? series : [];

  return (
    <div className="traffic-charts-root space-y-4">
      <TrafficOverview series={safeSeries} hasData={hasData} />
      <BandwidthBars series={safeSeries} hasData={hasData} />
    </div>
  );
}
