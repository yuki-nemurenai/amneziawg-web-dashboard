import React, { useState } from 'react';
import { X, Settings, Save, Check, Network } from 'lucide-react';

export default function NetworkSettingsModal({ serverConfig, status, onClose, onSave }) {
  const [formData, setFormData] = useState({
    listen_port: serverConfig?.listen_port || '689',
    address: serverConfig?.address || '172.20.0.1/16',
    dns: serverConfig?.dns || '1.1.1.1, 1.0.0.1',
    endpoint: serverConfig?.endpoint || '',
    lan_allowed: serverConfig?.lan_allowed || '0.0.0.0/0, ::/0',
    persistent_keepalive: serverConfig?.persistent_keepalive || '25',
    post_up: serverConfig?.post_up || 'iptables -A FORWARD -i awg0 -j ACCEPT; iptables -A FORWARD -o awg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE',
    post_down: serverConfig?.post_down || 'iptables -D FORWARD -i awg0 -j ACCEPT; iptables -D FORWARD -o awg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE',
  });

  const [loading, setLoading] = useState(false);
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (field, val) => {
    setFormData((prev) => ({ ...prev, [field]: val }));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const payload = {
        ...serverConfig,
        ...formData,
      };
      await onSave(payload);
      setSavedSuccess(true);
      setTimeout(() => {
        setSavedSuccess(false);
        onClose();
      }, 1500);
    } catch (err) {
      setError(err.message || 'Failed to save server configuration');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-md flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-900 max-w-2xl w-full max-h-[90vh] rounded-lg p-6 border border-[var(--card-border)] shadow-2xl flex flex-col modal-scale-in">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-[var(--divider)] pb-4 mb-5">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-lg bg-teal-50 dark:bg-teal-500/10 text-teal-600 dark:text-teal-300 border border-teal-100 dark:border-teal-500/20">
              <Network className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-lg font-extrabold text-slate-900 dark:text-white">Network & Interface</h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">Configure server subnet, port, and routing rules</p>
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
          <div className="mb-4 p-3.5 rounded-lg bg-rose-50 dark:bg-rose-500/10 border border-rose-200 dark:border-rose-500/20 text-rose-700 dark:text-rose-300 text-xs">
            {error}
          </div>
        )}

        {savedSuccess && (
          <div className="mb-4 p-3.5 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 text-emerald-700 dark:text-emerald-300 text-xs flex items-center gap-2 font-medium">
            <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
            <span>Settings successfully saved and reloaded live on AWG interface!</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto pr-2 space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1 flex justify-between">
                <span>Port (ListenPort)</span>
                <span className="text-teal-600 dark:text-teal-400 text-[10px]" title="Измените переменную AWG_PORT в файле .env">Read-only (change AWG_PORT in .env)</span>
              </label>
              <input
                type="text"
                value={formData.listen_port}
                disabled
                className="w-full px-3.5 py-2.5 rounded-lg finebank-input text-slate-500 dark:text-slate-500 bg-slate-50 dark:bg-slate-800/50 text-xs font-mono cursor-not-allowed"
                title="Для изменения публичного порта отредактируйте AWG_PORT в compose.yaml"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                Server Subnet (Address)
              </label>
              <input
                type="text"
                value={formData.address}
                onChange={(e) => handleChange('address', e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                Public Endpoint (IP)
              </label>
              <input
                type="text"
                value={formData.endpoint}
                onChange={(e) => handleChange('endpoint', e.target.value)}
                placeholder={status?.endpoint ? status.endpoint.split(':')[0] : 'Auto-detect IP'}
                className="w-full px-3.5 py-2.5 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono placeholder-slate-400 dark:placeholder-slate-500"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                DNS Servers
              </label>
              <input
                type="text"
                value={formData.dns}
                onChange={(e) => handleChange('dns', e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
              />
            </div>

            <div className="sm:col-span-2">
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                Client LAN Allowed Subnets
              </label>
              <input
                type="text"
                value={formData.lan_allowed}
                onChange={(e) => handleChange('lan_allowed', e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
              />
            </div>

            <div className="sm:col-span-2">
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                PostUp (iptables rules)
              </label>
              <input
                type="text"
                value={formData.post_up}
                onChange={(e) => handleChange('post_up', e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
              />
            </div>

            <div className="sm:col-span-2">
              <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
                PostDown (iptables teardown rules)
              </label>
              <input
                type="text"
                value={formData.post_down}
                onChange={(e) => handleChange('post_down', e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
              />
            </div>
          </div>

          <div className="pt-4 mt-2 flex items-center justify-end gap-3 border-t border-[var(--divider)]">
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
              className="flex items-center gap-2 px-5 py-2.5 rounded-lg bg-teal-600 hover:bg-teal-700 text-white text-xs font-bold shadow-md shadow-teal-600/20 transition active:scale-95"
            >
              <Save className="w-4 h-4" />
              <span>{loading ? 'Saving...' : 'Save Settings'}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
