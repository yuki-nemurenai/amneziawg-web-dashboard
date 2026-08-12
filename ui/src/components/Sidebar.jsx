import React from 'react';
import { AppIcons, NavIcon } from './icons/AppIcons';

export default function Sidebar({
  activeTab,
  setActiveTab,
  clientsCount,
  currentUser,
  onLogout,
  onChangePassword,
  onOpenNetworkSettings,
  onOpenAmneziaSettings,
}) {
  const mainNav = [
    {
      id: 'dashboard',
      label: 'Overview',
      filled: AppIcons.spaceDashboard,
      outlined: AppIcons.spaceDashboardOutline,
    },
    {
      id: 'users',
      label: 'Clients',
      filled: AppIcons.groups,
      outlined: AppIcons.groupsOutline,
      count: clientsCount || 0,
    },
  ];

  const systemNav = [
    {
      id: 'settings',
      label: 'Server Settings',
      filled: AppIcons.settings,
      outlined: AppIcons.settingsOutline,
      action: onOpenNetworkSettings,
    },
    {
      id: 'obfuscation',
      label: 'AmneziaWG Settings',
      filled: AppIcons.tune,
      outlined: AppIcons.tuneOutline,
      action: onOpenAmneziaSettings,
    },
  ];

  return (
    <aside className="hidden lg:flex w-[290px] bg-[var(--sidebar-bg)] border-r border-[var(--sidebar-border)] flex-col justify-between h-screen sticky top-0 z-40 select-none transition-colors duration-300 text-white">
      <div>
        <div className="px-6 py-7 flex items-center gap-3.5">
          <div className="relative group">
            <div className="absolute -inset-1 rounded-2xl bg-gradient-to-r from-teal-500 to-cyan-500 opacity-40 blur group-hover:opacity-75 transition duration-300"></div>
            <img
              src="/logo.png"
              alt="AmneziaWG Logo"
              className="relative w-11 h-11 rounded-2xl object-cover border border-teal-500/30 shadow-lg shadow-teal-500/20"
            />
          </div>
          <div>
            <h1 className="text-[17px] font-extrabold tracking-tight leading-none">
              Amnezia<span className="text-teal-400">WG</span>
            </h1>
            <p className="text-[11px] text-slate-400 font-semibold mt-1.5 uppercase tracking-[0.18em]">
              Dashboard
            </p>
          </div>
        </div>

        <div className="mx-5 rounded-2xl border border-white/10 bg-white/[0.04] p-4">
          <div className="flex items-center justify-between text-[11px] font-bold uppercase tracking-wider text-slate-400">
            <span>Server Interface</span>
            <AppIcons.speedOutline size={16} className="text-teal-400" />
          </div>
          <div className="mt-3.5 flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-slate-950/70 text-teal-400">
              <AppIcons.router size={22} />
            </div>
            <div>
              <div className="text-sm font-bold">awg0</div>
              <div className="text-[11px] font-medium text-slate-400">
                {clientsCount || 0} configured peers
              </div>
            </div>
          </div>
        </div>

        <div className="px-5 pt-7 pb-3 space-y-1">
          <div className="px-3 text-[10px] font-bold text-slate-500 uppercase tracking-[0.22em] mb-3">
            Main Menu
          </div>
          {mainNav.map((item) => {
            const isActive = activeTab === item.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => setActiveTab(item.id)}
                className={`w-full flex items-center justify-between px-3.5 py-3 rounded-xl text-[13px] font-bold transition-all duration-200 ${
                  isActive ? 'sidebar-active-item' : 'sidebar-inactive-item'
                }`}
              >
                <div className="flex items-center gap-3">
                  <NavIcon
                    active={isActive}
                    filled={item.filled}
                    outlined={item.outlined}
                    size={22}
                    className={isActive ? 'text-teal-300' : 'text-slate-400'}
                  />
                  <span>{item.label}</span>
                </div>
                {item.count !== undefined && (
                  <span
                    className={`min-w-7 text-center text-[10px] px-2 py-0.5 rounded-lg font-bold ${
                      isActive ? 'bg-teal-500/20 text-teal-200' : 'bg-white/[0.08] text-slate-300'
                    }`}
                  >
                    {item.count}
                  </span>
                )}
              </button>
            );
          })}
        </div>

        <div className="px-5 pt-2 space-y-1">
          <div className="px-3 text-[10px] font-bold text-slate-500 uppercase tracking-[0.22em] mb-3">
            Configuration
          </div>
          {systemNav.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={item.action}
              className="w-full flex items-center gap-3 px-3.5 py-3 rounded-xl text-[13px] font-bold sidebar-inactive-item transition-all duration-200"
            >
              <NavIcon
                active={false}
                filled={item.filled}
                outlined={item.outlined}
                size={22}
                className="text-slate-400"
              />
              <span>{item.label}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="p-5">
        <div className="p-3.5 rounded-2xl bg-white/[0.06] border border-white/10 flex items-center justify-between">
          <div className="flex items-center gap-3 overflow-hidden">
            <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-[#0D9488] to-[#14B8A6] flex items-center justify-center font-bold text-white shadow-md text-sm shrink-0">
              {currentUser?.username?.charAt(0).toUpperCase() || 'A'}
            </div>
            <div className="overflow-hidden">
              <div className="text-[13px] font-bold text-white truncate max-w-[130px]">
                {currentUser?.username || 'Admin'}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-0.5">
            <button
              type="button"
              onClick={onChangePassword}
              title="Change Password"
              className="material-icon-btn material-icon-btn--sidebar"
            >
              <AppIcons.vpnKey size={18} />
            </button>
            <button
              type="button"
              onClick={onLogout}
              title="Log Out"
              className="material-icon-btn material-icon-btn--sidebar"
            >
              <AppIcons.logout size={19} />
            </button>
          </div>
        </div>
      </div>
    </aside>
  );
}
