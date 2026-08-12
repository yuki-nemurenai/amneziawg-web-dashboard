import React from 'react';
import { ShieldCheck, Plus, Settings, Zap, LogOut, User } from 'lucide-react';

export default function Header({ status, currentUser, onOpenAddClient, onOpenSettings, onLogout }) {
  return (
    <header className="sticky top-0 z-30 glass-card border-b border-slate-800/80 px-6 py-4 mb-8">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
        {/* Brand Logo & Name */}
        <div className="flex items-center gap-3.5 group cursor-pointer">
          <div className="relative">
            <div className="absolute -inset-1 rounded-2xl bg-gradient-to-r from-teal-500 via-emerald-500 to-cyan-500 opacity-70 blur group-hover:opacity-100 transition duration-300"></div>
            <img
              src="/logo.png"
              alt="AmneziaWG Logo"
              className="relative w-11 h-11 rounded-xl object-cover border border-teal-500/30 shadow-md group-hover:scale-105 transition duration-300"
            />
          </div>

          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-extrabold tracking-tight gradient-text">
                AmneziaWG
              </h1>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-teal-500/10 text-teal-300 font-mono border border-teal-500/30 flex items-center gap-1">
                <Zap className="w-3 h-3 text-teal-400 fill-teal-400" /> v2.0 Web
              </span>
            </div>
            <p className="text-[11px] text-slate-400 font-medium">Dashboard</p>
          </div>
        </div>

        {/* Live Interface Badge & Action Buttons */}
        <div className="flex items-center gap-3">
          {/* Status Badge */}
          <div className="hidden sm:flex items-center gap-2.5 px-3 py-1.5 rounded-full bg-slate-950/80 border border-slate-800/80 text-xs shadow-inner">
            <div className="pulse-ring">
              <div className={`w-2.5 h-2.5 rounded-full relative z-10 ${status?.is_running ? 'bg-emerald-400' : 'bg-amber-400'}`}></div>
            </div>
            <span className="text-slate-300 font-medium">
              {status?.interface || 'awg0'}: <span className="text-cyan-300 font-mono font-semibold">{status?.endpoint || 'Loading...'}</span>
            </span>
          </div>

          {/* Admin User Badge */}
          {currentUser && (
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs text-slate-300 font-medium">
              <User className="w-3.5 h-3.5 text-indigo-400" />
              <span className="font-semibold text-white">{currentUser.username}</span>
            </div>
          )}

          {/* Settings Button */}
          <button
            onClick={onOpenSettings}
            className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-slate-900/90 hover:bg-slate-800 text-slate-300 hover:text-white transition-all duration-200 text-xs font-semibold border border-slate-700/60 shadow-md active:scale-95"
          >
            <Settings className="w-4 h-4 text-cyan-400" />
            <span className="hidden md:inline">Settings</span>
          </button>

          {/* Add User Button */}
          <button
            onClick={onOpenAddClient}
            className="relative group overflow-hidden rounded-xl p-[1px] focus:outline-none"
          >
            <span className="absolute inset-0 bg-gradient-to-r from-indigo-500 via-purple-500 to-cyan-500 rounded-xl opacity-90 group-hover:opacity-100 transition duration-300 blur-sm"></span>
            <span className="relative flex items-center gap-2 px-4 py-2 rounded-[11px] bg-slate-950 hover:bg-slate-900 text-white transition-all duration-200 text-xs font-bold shadow-xl">
              <Plus className="w-4 h-4 text-indigo-400 group-hover:rotate-90 transition duration-300" />
              <span className="gradient-text-accent">Create User</span>
            </span>
          </button>

          {/* Logout Button */}
          <button
            onClick={onLogout}
            title="Log Out"
            className="p-2 rounded-xl bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 transition border border-rose-500/20 active:scale-95"
          >
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </div>
    </header>
  );
}
