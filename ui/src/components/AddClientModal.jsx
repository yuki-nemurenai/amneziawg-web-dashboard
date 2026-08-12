import React, { useState } from 'react';
import { X, UserPlus, Sparkles, Copy, Check, Download, ShieldCheck } from 'lucide-react';

export default function AddClientModal({ onClose, onSubmit }) {
  const [name, setName] = useState('');
  const [ip, setIp] = useState('');
  const [loading, setLoading] = useState(false);
  const [createdClient, setCreatedClient] = useState(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Client name is required');
      return;
    }
    setError('');
    setLoading(true);

    try {
      const res = await onSubmit({ name: name.trim(), ip: ip.trim() || undefined });
      setCreatedClient(res);
    } catch (err) {
      setError(err.message || 'Failed to create user account');
    } finally {
      setLoading(false);
    }
  };

  const copyConfigText = () => {
    if (createdClient?.config_text) {
      navigator.clipboard.writeText(createdClient.config_text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleDownloadConfig = () => {
    if (!createdClient?.config_text) return;
    const blob = new Blob([createdClient.config_text], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${createdClient.name}.conf`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-md flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-900 max-w-xl w-full rounded-lg p-6 border border-[var(--card-border)] shadow-2xl relative modal-scale-in">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-[var(--divider)] pb-4 mb-6">
          <div className="flex items-center gap-3">
            <div className="p-3 rounded-lg bg-teal-50 dark:bg-teal-500/10 text-teal-600 dark:text-teal-300 border border-teal-100 dark:border-teal-500/20">
              <UserPlus className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-lg font-extrabold text-slate-900 dark:text-white">Create AmneziaWG Profile</h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">Generate AWG keys, profile & QR code</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3.5 rounded-lg bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 text-rose-700 dark:text-rose-300 text-xs flex items-center gap-2">
            <span>{error}</span>
          </div>
        )}

        {!createdClient ? (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                Client Name <span className="text-teal-600 dark:text-teal-300">*</span>
              </label>
              <input
                type="text"
                required
                placeholder="e.g. ivanov, phone_alex, laptop"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full px-4 py-2.5 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-sm placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none font-mono"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">
                Tunnel IP Address (optional)
              </label>
              <input
                type="text"
                placeholder="Leave blank for auto-allocation (e.g. 172.24.170.6)"
                value={ip}
                onChange={(e) => setIp(e.target.value)}
                className="w-full px-4 py-2.5 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-sm placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none font-mono"
              />
              <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1">
                The system automatically assigns the next available IP address in the VPN subnet.
              </p>
            </div>

            <div className="pt-4 flex items-center justify-end gap-3 border-t border-[var(--divider)]">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 text-xs font-semibold transition"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex items-center gap-2 px-5 py-2.5 rounded-lg bg-teal-600 hover:bg-teal-700 text-white text-xs font-bold shadow-lg shadow-teal-600/20 active:scale-95 disabled:opacity-50 transition"
              >
                <Sparkles className="w-4 h-4" />
                <span>{loading ? 'Creating...' : 'Generate Profile'}</span>
              </button>
            </div>
          </form>
        ) : (
          /* Result View with QR & Config */
          <div className="space-y-6">
            <div className="p-3.5 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 text-emerald-800 dark:text-emerald-300 text-xs flex items-center justify-between">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
                <span>
                  <strong>User created!</strong> Assigned IP:{' '}
                  <span className="font-mono font-bold text-slate-900 dark:text-white">{createdClient.ip}</span>
                </span>
              </div>
            </div>

            <div className="flex flex-col sm:flex-row items-center gap-6">
              {/* QR Code */}
              {createdClient.qr_code_svg && (
                <div className="flex flex-col items-center gap-2">
                  <div className="p-3.5 rounded-lg bg-white shadow-md border border-slate-200">
                    <img
                      src={createdClient.qr_code_svg}
                      alt="AmneziaWG QR Code"
                      className="w-48 h-48 object-contain"
                    />
                  </div>
                  <span className="text-[11px] text-slate-500 dark:text-slate-400 font-medium">Scan using AmneziaVPN or AmneziaWG apps</span>
                </div>
              )}

              {/* Config Actions */}
              <div className="flex-1 space-y-3 w-full">
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300">
                  Configuration (.conf):
                </label>
                <textarea
                  readOnly
                  rows={8}
                  value={createdClient.config_text}
                  className="w-full p-3.5 rounded-lg border border-slate-800 text-[11px] focus:outline-none shadow-inner select-all config-code-box"
                />

                <div className="flex items-center gap-2">
                  <button
                    onClick={copyConfigText}
                    className="flex-1 flex items-center justify-center gap-2 px-3.5 py-2.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-semibold transition active:scale-95"
                  >
                    {copied ? <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400" /> : <Copy className="w-4 h-4" />}
                    <span>{copied ? 'Copied!' : 'Copy Text'}</span>
                  </button>

                  <button
                    type="button"
                    onClick={handleDownloadConfig}
                    className="flex-1 flex items-center justify-center gap-2 px-3.5 py-2.5 rounded-lg bg-teal-600 hover:bg-teal-700 text-white text-xs font-bold shadow-md shadow-teal-600/20 transition active:scale-95"
                  >
                    <Download className="w-4 h-4" />
                    <span>Download .conf</span>
                  </button>
                </div>
              </div>
            </div>

            <div className="pt-4 border-t border-[var(--divider)] flex justify-end">
              <button
                onClick={onClose}
                className="px-6 py-2.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 text-xs font-bold transition"
              >
                Done
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
