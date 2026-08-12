import React, { useEffect, useRef, useState } from 'react';
import { AppIcons } from './icons/AppIcons';

const pageMeta = {
  dashboard: {
    title: 'Overview',
    subtitle: 'Server health, traffic & tunnels',
    icon: AppIcons.spaceDashboard,
  },
  users: {
    title: 'Client Accounts',
    subtitle: 'Peers, credentials & profiles',
    icon: AppIcons.groups,
  },
  analytics: {
    title: 'Analytics',
    subtitle: 'Traffic intelligence & insights',
    icon: AppIcons.insights,
  },
};

function formatShortDate(date = new Date()) {
  return date.toLocaleDateString('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
  });
}

function EndpointBadge({ status }) {
  const endpoint = status?.endpoint;
  const isAuto = status?.auto_endpoint;
  const isRunning = status?.is_running;

  return (
    <div className="topbar-endpoint-badge">
      {/* Pulse dot */}
      <span className="topbar-status-dot-wrap">
        {isRunning && <span className="topbar-status-ping" />}
        <span className={`topbar-status-dot ${isRunning ? 'online' : 'offline'}`} />
      </span>

      <span className="topbar-endpoint-text">
        <span className="topbar-iface">{status?.interface || 'awg0'}{status?.mode && status.mode !== 'Unknown' ? ` (${status.mode.toLowerCase()} mode)` : ''}</span>
        <span className="topbar-sep"> / </span>
        {endpoint ? (
          <>
            <span className="topbar-endpoint-ip">{endpoint}</span>
          </>
        ) : (
          <span className="topbar-endpoint-pending">endpoint pending</span>
        )}
        {status?.location && (
          <>
            <span className="topbar-sep"> / </span>
            <span className="topbar-location">
              {status.location.country_code ? status.location.country_code.toUpperCase().replace(/./g, char => String.fromCodePoint(char.charCodeAt(0) + 127397)) : ''} {status.location.country}
            </span>
          </>
        )}
      </span>
    </div>
  );
}

function NotificationsPanel({ events, onClose }) {
  return (
    <div className="topbar-notif-panel">
      <div className="topbar-notif-header">
        <h3>Recent Activity</h3>
      </div>
      <div className="topbar-notif-body">
        {events.length === 0 ? (
          <div className="topbar-notif-empty">
            <AppIcons.notificationsOutline size={28} className="opacity-20 mb-2" />
            <p>No recent activity</p>
          </div>
        ) : (
          <div className="topbar-notif-list">
            {events.map((ev) => (
              <div key={ev.id} className="topbar-notif-item">
                <div className="mt-0.5 shrink-0">
                  {ev.type === 'success' ? (
                    <AppIcons.checkCircle size={16} className="text-emerald-500" />
                  ) : (
                    <AppIcons.info size={16} className="text-sky-500" />
                  )}
                </div>
                <div className="min-w-0">
                  <p className="topbar-notif-msg">{ev.message}</p>
                  <p className="topbar-notif-time">{ev.time}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default function Topbar({
  status,
  activeTab,
  setActiveTab,
  onOpenAddClient,
  onOpenSettings,
  onChangePassword,
  searchTerm,
  setSearchTerm,
  theme,
  toggleTheme,
  events = [],
  currentUser,
}) {
  const [showEvents, setShowEvents] = useState(false);
  const eventsRef = useRef(null);
  const page = pageMeta[activeTab] || pageMeta.dashboard;
  const username = currentUser?.username || 'Admin';

  useEffect(() => {
    function handleClickOutside(e) {
      if (eventsRef.current && !eventsRef.current.contains(e.target)) {
        setShowEvents(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <header className="topbar">
      {/* ── Row 1: page title + actions ─────────────────────────── */}
      <div className="topbar-row topbar-row-main">
        {/* Left: greeting + page title */}
        <div className="topbar-page-info">
          <p className="topbar-greeting">
            {formatShortDate()} · <span className="text-slate-900 dark:text-white font-bold">{username}</span>
          </p>
          <h2 className="topbar-title">{page.title}</h2>
        </div>

        {/* Right: search + add button + icon buttons */}
        <div className="topbar-actions">
          {/* Search */}
          <div className="topbar-search-wrap">
            <AppIcons.searchOutline
              size={17}
              className="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
            />
            <input
              type="text"
              placeholder="Search clients…"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="material-search-input topbar-search-input"
            />
          </div>

          {/* Add client button */}
          <button
            type="button"
            onClick={onOpenAddClient}
            className="material-primary-btn"
          >
            <AppIcons.add size={18} />
            <span>New Client</span>
          </button>

          {/* Icon actions */}
          <div className="topbar-icon-actions">
            {/* Theme toggle */}
            <button
              type="button"
              onClick={toggleTheme}
              title={theme === 'dark' ? 'Light mode' : 'Dark mode'}
              className="material-icon-btn"
            >
              {theme === 'dark' ? (
                <AppIcons.lightModeOutline size={19} className="text-amber-400" />
              ) : (
                <AppIcons.darkModeOutline size={19} />
              )}
            </button>

            {/* Notifications */}
            <div className="relative" ref={eventsRef}>
              <button
                type="button"
                onClick={() => setShowEvents(!showEvents)}
                title="Notifications"
                className="material-icon-btn relative"
              >
                <AppIcons.notificationsOutline size={19} />
                {events.length > 0 && (
                  <span className="topbar-notif-dot" />
                )}
              </button>
              {showEvents && (
                <NotificationsPanel events={events} onClose={() => setShowEvents(false)} />
              )}
            </div>

            {/* Settings */}
            <button
              type="button"
              onClick={onOpenSettings}
              title="Server Settings"
              className="material-icon-btn"
            >
              <AppIcons.settingsOutline size={19} />
            </button>

            {/* Avatar / Change password */}
            <button
              type="button"
              onClick={onChangePassword}
              title={`${username} · Change Password`}
              className="topbar-avatar cursor-pointer hover:scale-105 active:scale-95 transition"
            >
              {username.charAt(0).toUpperCase()}
            </button>
          </div>
        </div>
      </div>

      {/* ── Mobile Nav Tabs (< lg) ────────────────────────────── */}
      <div className="lg:hidden grid grid-cols-2 gap-2 mt-3 mb-1">
        {[
          { id: 'dashboard', label: 'Overview' },
          { id: 'users', label: 'Clients' },
        ].map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => setActiveTab(item.id)}
            className={`px-3 py-2 rounded-xl text-[12px] font-bold border transition ${
              activeTab === item.id
                ? 'bg-emerald-600 text-white border-emerald-600 shadow-md shadow-emerald-600/20'
                : 'bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-300 border-[var(--card-border)]'
            }`}
          >
            {item.label}
          </button>
        ))}
      </div>

      {/* ── Row 2: endpoint status + subtitle ──────────────────── */}
      <div className="topbar-row topbar-row-sub">
        <EndpointBadge status={status} />
        <p className="topbar-subtitle hidden sm:block">{page.subtitle}</p>
      </div>
    </header>
  );
}
