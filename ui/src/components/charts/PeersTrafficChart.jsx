import React, { useMemo, useState } from 'react';
import {
  LineChart,
  Line,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { formatBytes, formatChartTime } from '../../utils/traffic';

const C = {
  grid: 'rgba(148,163,184,0.14)',
  tick: '#94A3B8',
};

const PEER_COLORS = [
  '#0EA5E9', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6',
  '#EC4899', '#14B8A6', '#F97316', '#6366F1', '#84CC16',
];

function filterByPeriod(series, period) {
  if (!series || !series.length) return [];
  if (period === 'all') return series;
  const now = Math.floor(Date.now() / 1000);
  const windows = { '1h': 3600, '6h': 21600, '24h': 86400 };
  const cutoff = now - (windows[period] || 3600);
  return series.filter((p) => (p.timestamp || 0) >= cutoff);
}

function CustomTooltip({ active, payload, label, filter }) {
  if (!active || !payload || !payload.length) return null;
  // sort by value descending
  const sorted = [...payload].sort((a, b) => b.value - a.value);
  return (
    <div className="chart-tooltip bg-white dark:bg-slate-900 border border-slate-200 dark:border-white/10 shadow-xl rounded-lg p-3 text-sm z-50">
      <p className="font-bold text-slate-900 dark:text-white mb-2">{label || ''}</p>
      <div className="flex flex-col gap-1.5">
        {sorted.map((entry) => (
          <div key={entry.name} className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ background: entry.color }} />
              <span className="text-slate-600 dark:text-slate-300 font-medium truncate max-w-[120px]">{entry.name}</span>
            </div>
            <span className="font-bold text-slate-900 dark:text-white shrink-0">{formatBytes(entry.value)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

export default function PeersTrafficChart({ data, peers, hasData }) {
  const [period, setPeriod] = useState('all');
  const [filter, setFilter] = useState('total');

  const chartData = useMemo(() => filterByPeriod(data, period), [data, period]);

  const now = Math.floor(Date.now() / 1000);
  const windows = { '1h': 3600, '6h': 21600, '24h': 86400 };
  const xDomain = period === 'all' 
    ? ['dataMin', 'dataMax'] 
    : [now - (windows[period] || 3600), now];

  return (
    <div className="chart-card finebank-card rounded-2xl p-5 mt-5">
      <div className="chart-card-header flex flex-col sm:flex-row sm:items-start justify-between mb-5 gap-4">
        <div>
          <h3 className="chart-card-title text-[16px] font-extrabold text-slate-950 dark:text-white">Top Peers Traffic</h3>
          <p className="chart-card-subtitle mt-0.5 text-[12px] text-slate-500 dark:text-slate-400">Activity for up to 10 most active peers</p>
        </div>
        <div className="chart-card-actions flex items-center gap-2 sm:gap-4 flex-wrap">
          <div className="chart-filter-tabs flex items-center bg-slate-100 dark:bg-slate-800/50 p-1 rounded-lg shrink-0">
            {['total', 'download', 'upload'].map((f) => (
              <button
                key={f}
                type="button"
                onClick={() => setFilter(f)}
                className={`chart-filter-tab px-3 py-1.5 text-[12px] font-bold rounded-md transition-colors ${
                  filter === f 
                    ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm' 
                    : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                }`}
              >
                {f.charAt(0).toUpperCase() + f.slice(1)}
              </button>
            ))}
          </div>
          
          <div className="chart-period-selector flex items-center bg-slate-100 dark:bg-slate-800/50 p-1 rounded-lg shrink-0">
            {[{id: 'all', label: 'All'}, {id: '1h', label: '1H'}, {id: '6h', label: '6H'}, {id: '24h', label: '24H'}].map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => setPeriod(p.id)}
                className={`chart-period-btn px-2.5 py-1.5 text-[11px] font-bold rounded-md transition-colors ${
                  period === p.id 
                    ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm' 
                    : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                }`}
              >
                {p.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="chart-area-wrap relative min-h-[300px]">
        {!hasData && (
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <div className="w-10 h-10 border-4 border-slate-100 dark:border-slate-800 border-t-teal-500 rounded-full animate-spin mb-3"></div>
            <p className="text-sm font-bold text-slate-600 dark:text-slate-300">Collecting traffic samples…</p>
          </div>
        )}
        {hasData && chartData.length === 0 && (
          <div className="absolute inset-0 flex flex-col items-center justify-center">
             <p className="text-sm font-bold text-slate-600 dark:text-slate-300">No traffic data for this period</p>
          </div>
        )}
        {chartData.length > 0 && (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={chartData} margin={{ top: 10, right: 4, left: 0, bottom: 0 }}>
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
              <Tooltip content={<CustomTooltip filter={filter} />} cursor={{ stroke: C.tick, strokeDasharray: '4 4', strokeOpacity: 0.4 }} />
              
              {peers.map((peer, index) => {
                const color = PEER_COLORS[index % PEER_COLORS.length];
                const dataKey = `${peer}_${filter}`;
                return (
                  <Line
                    key={peer}
                    type="monotone"
                    dataKey={dataKey}
                    name={peer}
                    stroke={color}
                    strokeWidth={2.5}
                    dot={false}
                    activeDot={{ r: 4, strokeWidth: 2, stroke: '#fff' }}
                  />
                );
              })}
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
      
      {peers.length > 0 && (
        <div className="flex flex-wrap items-center justify-center gap-4 mt-6 pt-4 border-t border-[var(--divider)]">
          {peers.map((peer, index) => (
            <div key={peer} className="flex items-center gap-1.5 text-[12px] font-bold text-slate-600 dark:text-slate-300">
              <span className="w-2.5 h-2.5 rounded-full" style={{ background: PEER_COLORS[index % PEER_COLORS.length] }} />
              {peer}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
