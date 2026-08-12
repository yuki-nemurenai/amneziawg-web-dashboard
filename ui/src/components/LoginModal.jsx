import React, { useState } from 'react';
import { ShieldCheck, Lock, User, KeyRound, Sparkles, ArrowRight, ShieldAlert, Eye, EyeOff } from 'lucide-react';

export default function LoginModal({ needsSetup, onLogin, onSetup }) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    if (needsSetup && password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setLoading(true);

    try {
      if (needsSetup) {
        await onSetup({ username: username.trim(), password });
      } else {
        await onLogin({ username: username.trim(), password });
      }
    } catch (err) {
      setError(err.message || 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/60 dark:bg-[#060913]/90 backdrop-blur-xl flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-900 max-w-md w-full rounded-lg p-8 border border-[var(--card-border)] shadow-2xl relative z-10 modal-scale-in">
        <div className="flex flex-col items-center text-center mb-6">
          <div className="relative mb-4">
            <div className="absolute -inset-2 rounded-2xl bg-teal-500/30 blur-md"></div>
            <img
              src="/logo.png"
              alt="AmneziaWG Logo"
              className="relative w-16 h-16 rounded-2xl object-cover border border-teal-500/40 shadow-xl"
            />
          </div>

          <h2 className="text-2xl font-extrabold text-slate-900 dark:text-white tracking-tight">
            {needsSetup ? 'Initial Admin Setup' : 'AmneziaWG Dashboard'}
          </h2>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1 max-w-xs leading-relaxed">
            {needsSetup
              ? 'Welcome! No administrator accounts were found in PostgreSQL. Please create the initial admin user.'
              : 'Enter your credentials.'}
          </p>
        </div>

        {error && (
          <div className="mb-5 p-3.5 rounded-lg bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 text-rose-700 dark:text-rose-300 text-xs flex items-center gap-2">
            <ShieldAlert className="w-4 h-4 text-rose-600 dark:text-rose-400 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
              Username
            </label>
            <div className="relative">
              <User className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                required
                placeholder="Enter your username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full pl-10 pr-4 py-3 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-sm placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none font-mono"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
              Password
            </label>
            <div className="relative">
              <Lock className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type={showPassword ? "text" : "password"}
                required
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full pl-10 pr-10 py-3 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-sm placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none font-mono"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
              >
                {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          {needsSetup && (
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                Confirm Password
              </label>
              <div className="relative">
                <KeyRound className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
                <input
                  type={showConfirmPassword ? "text" : "password"}
                  required
                  placeholder="Enter your password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="w-full pl-10 pr-10 py-3 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-sm placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none font-mono"
                />
                <button
                  type="button"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                >
                  {showConfirmPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full flex items-center justify-center gap-2 py-3 rounded-lg bg-teal-600 hover:bg-teal-700 text-white font-bold text-sm shadow-xl shadow-teal-600/20 active:scale-95 disabled:opacity-50 transition duration-200 mt-6"
          >
            {needsSetup ? <Sparkles className="w-4 h-4" /> : <ArrowRight className="w-4 h-4" />}
            <span>{loading ? 'Processing...' : needsSetup ? 'Create Admin Account' : 'Sign In to Dashboard'}</span>
          </button>
        </form>
      </div>
    </div>
  );
}
