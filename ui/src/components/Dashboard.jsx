import React, { useMemo } from 'react';
import { AppIcons } from './icons/AppIcons';
import TrafficCharts from './charts/TrafficCharts';
import PeersTrafficChart from './charts/PeersTrafficChart';
import Sparkline from './charts/Sparkline';
import useLiveTrafficHistory from '../hooks/useLiveTrafficHistory';
import {
  buildTrafficSeries,
  buildPeersTrafficSeries,
  formatBytes,
  generatePlaceholderSeries,
  mergeTrafficHistory,
} from '../utils/traffic';

const toneStyles = {
  emerald: {
    icon: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-300',
    bar: 'bg-emerald-500',
    text: 'text-emerald-600 dark:text-emerald-300',
    glow: 'shadow-emerald-500/20',
    spark: '#10B981',
  },
  teal: {
    icon: 'bg-teal-50 text-teal-600 dark:bg-teal-500/10 dark:text-teal-300',
    bar: 'bg-teal-500',
    text: 'text-teal-600 dark:text-teal-300',
    glow: 'shadow-teal-500/20',
    spark: '#14B8A6',
  },
  sky: {
    icon: 'bg-sky-50 text-sky-600 dark:bg-sky-500/10 dark:text-sky-300',
    bar: 'bg-sky-500',
    text: 'text-sky-600 dark:text-sky-300',
    glow: 'shadow-sky-500/20',
    spark: '#0EA5E9',
  },
  amber: {
    icon: 'bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300',
    bar: 'bg-amber-500',
    text: 'text-amber-600 dark:text-amber-300',
    glow: 'shadow-amber-500/20',
    spark: '#F59E0B',
  },
};

function TrendBadge({ value, prevValue }) {
  if (value == null || prevValue == null || prevValue === 0) return null;
  const diff = value - prevValue;
  const pct = Math.abs(Math.round((diff / prevValue) * 100));
  if (pct < 1) return null;
  const up = diff > 0;
  const ArrowIcon = up ? AppIcons.arrowUp : AppIcons.arrowDown;
  if (!ArrowIcon) return null;
  return (
    <span
      className={`inline-flex items-center gap-0.5 text-[11px] font-bold px-1.5 py-0.5 rounded-md ${
        up
          ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400'
          : 'bg-slate-100 text-slate-500 dark:bg-white/[0.05] dark:text-slate-400'
      }`}
    >
      <ArrowIcon size={12} />
      {pct}%
    </span>
  );
}

function MetricCard({ title, value, detail, icon: IconComponent, tone, sparkline, trend }) {
  const styles = toneStyles[tone] || toneStyles.teal;

  return (
    <div className="metric-card finebank-card rounded-2xl p-5 flex flex-col gap-4">
      <div className="flex items-start justify-between">
        <div className={`w-12 h-12 rounded-2xl flex items-center justify-center shrink-0 shadow-lg ${styles.icon} ${styles.glow}`}>
          {IconComponent ? <IconComponent size={24} /> : null}
        </div>
        {trend && (
          <TrendBadge value={trend.current} prevValue={trend.prev} />
        )}
      </div>

      <div>
        <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-slate-500 dark:text-slate-400">
          {title}
        </p>
        <div className="mt-1.5 text-[26px] font-extrabold leading-none tracking-tight text-slate-950 dark:text-white">
          {value}
        </div>
        <p className="mt-1.5 text-[12px] font-medium text-slate-500 dark:text-slate-400 leading-relaxed">
          {detail}
        </p>
      </div>

      {sparkline && (
        <Sparkline
          data={sparkline.data}
          dataKey={sparkline.dataKey}
          color={sparkline.color}
          gradientId={sparkline.gradientId}
        />
      )}
    </div>
  );
}

function InfoRow({ label, value, icon: IconComponent, accent = false }) {
  return (
    <div className="flex items-center justify-between gap-3 py-3 border-b border-[var(--divider)] last:border-0">
      <div className="flex items-center gap-2.5 text-[13px] font-semibold text-slate-500 dark:text-slate-400 shrink-0">
        {IconComponent ? <IconComponent size={16} /> : null}
        <span>{label}</span>
      </div>
      <span
        className={`max-w-[180px] truncate text-[13px] font-bold text-right ${
          accent ? 'text-teal-600 dark:text-teal-300' : 'text-slate-900 dark:text-white'
        }`}
        title={value}
      >
        {value}
      </span>
    </div>
  );
}

function ServerPanel({ status, serverConfig, onlinePercent = 0 }) {
  const endpoint = status?.endpoint || serverConfig?.endpoint || '';
  const isAutoEndpoint = status?.auto_endpoint;

  const flag = status?.location?.country_code
    ? status.location.country_code.toUpperCase().replace(/./g, char => String.fromCodePoint(char.charCodeAt(0) + 127397))
    : '';
  const locationStr = status?.location ? `${flag} ${status.location.country}` : 'Unknown';

  const rows = [
    {
      label: 'Interface',
      value: status?.interface || 'awg0',
      icon: AppIcons.router,
    },
    {
      label: 'IP Address',
      value: endpoint
        ? endpoint.split(':')[0]
        : 'Not configured',
      icon: AppIcons.public,
      accent: true,
    },
    {
      label: 'AWG Mode',
      value: status?.mode || 'Unknown',
      icon: AppIcons.settingsOutline,
    },
    {
      label: 'Location',
      value: locationStr,
      icon: AppIcons.public,
    },
    {
      label: 'Subnet',
      value: serverConfig?.address || '—',
      icon: AppIcons.lan,
    },
    {
      label: 'Listen Port',
      value: serverConfig?.listen_port || '689',
      icon: AppIcons.port,
    },
  ];

  const safePercent = Math.min(100, Math.max(0, onlinePercent || 0));

  return (
    <div className="finebank-card rounded-2xl p-5 flex flex-col justify-between">
      <div>
        {/* Header */}
        <div className="flex items-start justify-between mb-6">
          <div>
            <h3 className="text-[16px] font-extrabold text-slate-950 dark:text-white">Server Info</h3>
            <p className="mt-0.5 text-[12px] text-slate-500 dark:text-slate-400">Interface & peer status</p>
          </div>
          <span
            className={`px-2.5 py-1 rounded-lg text-[11px] font-bold ${
              status?.is_running
                ? 'bg-teal-50 text-teal-600 dark:bg-teal-500/10 dark:text-teal-300'
                : 'bg-amber-50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-300'
            }`}
          >
            {status?.is_running ? '● Running' : '◌ Offline'}
          </span>
        </div>

        {/* Donut gauge */}
        <div className="flex items-center justify-center my-6">
          <div className="relative h-44 w-44">
            <svg viewBox="0 0 120 120" className="h-full w-full -rotate-90">
              <circle cx="60" cy="60" r="48" fill="none" stroke="rgba(148, 163, 184, 0.18)" strokeWidth="10" />
              {safePercent > 0 ? (
                <circle
                  cx="60" cy="60" r="48" fill="none"
                  stroke="#0D9488" strokeWidth="10" strokeLinecap="round"
                  strokeDasharray={`${safePercent * 3.016} 301.6`}
                  style={{ transition: 'stroke-dasharray 0.8s ease' }}
                />
              ) : (
                <circle
                  cx="60" cy="60" r="48" fill="none"
                  stroke="#0D9488" strokeWidth="10" strokeLinecap="round"
                  strokeDasharray="4 301.6"
                  opacity="0.4"
                />
              )}
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-3xl font-extrabold text-slate-950 dark:text-white">{safePercent}%</span>
              <span className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mt-0.5">online</span>
            </div>
          </div>
        </div>
      </div>

      {/* Info rows */}
      <div className="mt-4 pt-2 border-t border-[var(--divider)]">
        {rows.map((row) => (
          <InfoRow key={row.label} {...row} />
        ))}
      </div>
    </div>
  );
}

export default function Dashboard({ status, serverConfig }) {
  const liveHistory = useLiveTrafficHistory(status);

  const chartSeries = useMemo(() => {
    const merged = mergeTrafficHistory(status?.traffic_history, liveHistory);
    const series = buildTrafficSeries(merged);
    return series.length > 0 ? series : generatePlaceholderSeries();
  }, [status?.traffic_history, liveHistory]);

  const { data: peersSeries, peers: activePeers } = useMemo(() => {
    return buildPeersTrafficSeries(status?.peer_traffic_history, serverConfig?.peers || []);
  }, [status?.peer_traffic_history, serverConfig?.peers]);

  const hasChartData = Array.isArray(chartSeries) && chartSeries.some((p) => p && ((p.download || 0) > 0 || (p.upload || 0) > 0));
  const hasPeersChartData = Array.isArray(peersSeries) && peersSeries.length > 0 && activePeers.length > 0;

  const onlinePeers = status?.online_peers || 0;
  const totalPeers = status?.total_peers || 0;
  const offlinePeers = Math.max(totalPeers - onlinePeers, 0);
  const onlinePercent = totalPeers > 0 ? Math.round((onlinePeers / totalPeers) * 100) : 0;

  const totalRx = status?.total_rx_bytes || 0;
  const totalTx = status?.total_tx_bytes || 0;
  const totalVolume = totalRx + totalTx;

  const latest = chartSeries.length > 0 ? chartSeries[chartSeries.length - 1] : null;
  const prev = chartSeries.length >= 2 ? chartSeries[chartSeries.length - 2] : null;

  const cards = [
    {
      title: 'Active Tunnels',
      value: `${onlinePeers} / ${totalPeers}`,
      detail: offlinePeers > 0 ? `${offlinePeers} peer${offlinePeers > 1 ? 's' : ''} offline` : 'All peers connected',
      icon: AppIcons.vpnKey,
      tone: 'emerald',
    },
    {
      title: 'Total Download',
      value: status?.total_rx_formatted || '0 B',
      detail: latest ? `Current: ${latest.downloadRateFormatted}` : 'Incoming tunnel traffic',
      icon: AppIcons.download,
      tone: 'emerald',
      sparkline: {
        data: chartSeries,
        dataKey: 'download',
        color: toneStyles.emerald.spark,
        gradientId: 'metricDownloadSpark',
      },
      trend: prev ? { current: latest?.download, prev: prev?.download } : null,
    },
    {
      title: 'Total Upload',
      value: status?.total_tx_formatted || '0 B',
      detail: latest ? `Current: ${latest.uploadRateFormatted}` : 'Outgoing tunnel traffic',
      icon: AppIcons.upload,
      tone: 'teal',
      sparkline: {
        data: chartSeries,
        dataKey: 'upload',
        color: toneStyles.teal.spark,
        gradientId: 'metricUploadSpark',
      },
      trend: prev ? { current: latest?.upload, prev: prev?.upload } : null,
    },
    {
      title: 'Throughput',
      value: formatBytes(totalVolume),
      detail: `Combined throughput across interface`,
      icon: AppIcons.signal,
      tone: 'teal',
    },
  ];

  return (
    <section className="space-y-5">
      {/* Top 4 metric cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => (
          <MetricCard key={card.title} {...card} />
        ))}
      </div>

      {/* Main charts + Server health panel */}
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1fr_340px]">
        <div className="space-y-4">
          <TrafficCharts series={chartSeries} hasData={hasChartData} />
          <PeersTrafficChart data={peersSeries} peers={activePeers} hasData={hasPeersChartData} />
        </div>
        <ServerPanel status={status} serverConfig={serverConfig} onlinePercent={onlinePercent} />
      </div>
    </section>
  );
}
