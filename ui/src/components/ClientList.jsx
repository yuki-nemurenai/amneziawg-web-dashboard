import React, { useMemo, useState } from 'react';
import { AppIcons } from './icons/AppIcons';
import { Wifi, ArrowDownRight, ArrowUpRight, Clock, QrCode, Trash2, MoreHorizontal, ShieldAlert, KeyRound, Download } from 'lucide-react';

function StatusBadge({ online }) {
  return online ? (
    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-bold bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">
      <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
      Online
    </span>
  ) : (
    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[11px] font-bold bg-slate-100 text-slate-500 dark:bg-white/10 dark:text-slate-300">
      <span className="w-1.5 h-1.5 rounded-full bg-slate-400" />
      Offline
    </span>
  );
}

function ActionButton({ title, children, onClick, tone = 'default' }) {
  const toneClass =
    tone === 'primary'
      ? 'bg-teal-600 text-white hover:bg-teal-700 border-teal-600 shadow-sm shadow-teal-600/20'
      : tone === 'danger'
        ? 'bg-rose-50 text-rose-600 hover:bg-rose-100 border-rose-100 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-500/20'
        : 'bg-white text-slate-600 hover:bg-slate-50 border-slate-200 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 dark:border-white/10';

  return (
    <button
      onClick={onClick}
      title={title}
      className={`inline-flex h-9 min-w-9 items-center justify-center gap-2 rounded-lg border px-2.5 text-[12px] font-bold transition active:scale-95 ${toneClass}`}
    >
      {children}
    </button>
  );
}

export default function ClientList({ clients, searchTerm, onShowQR, onDeleteClient, onDownloadConfig }) {
  const [deleteConfirm, setDeleteConfirm] = useState(null);

  const filteredClients = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    if (!query) return clients;

    return clients.filter(
      (client) =>
        client.name.toLowerCase().includes(query) ||
        (client.ip && client.ip.includes(query)) ||
        (client.public_key && client.public_key.toLowerCase().includes(query))
    );
  }, [clients, searchTerm]);

  const onlineCount = clients.filter((client) => client.is_online).length;

  return (
    <section className="finebank-card rounded-lg overflow-hidden transition-colors duration-300">
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 px-5 py-5 border-b border-[var(--divider)]">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-extrabold text-slate-950 dark:text-white">Client Accounts</h2>
            <span className="text-[11px] px-2.5 py-1 rounded-md bg-slate-100 text-slate-600 dark:bg-white/10 dark:text-slate-300 font-bold">
              {filteredClients.length} shown
            </span>
          </div>
          <p className="text-[13px] text-slate-500 dark:text-slate-400 mt-1">
            Peer access, handshakes, bandwidth totals, and configuration actions
          </p>
        </div>

        <div className="grid grid-cols-2 sm:flex items-center gap-2 text-[12px] font-bold">
          <div className="rounded-lg border border-[var(--card-border)] bg-slate-50 dark:bg-white/[0.03] px-3 py-2">
            <span className="text-slate-500 dark:text-slate-400">Online</span>{' '}
            <span className="text-emerald-600 dark:text-emerald-300">{onlineCount}</span>
          </div>
          <div className="rounded-lg border border-[var(--card-border)] bg-slate-50 dark:bg-white/[0.03] px-3 py-2">
            <span className="text-slate-500 dark:text-slate-400">Total</span>{' '}
            <span className="text-slate-900 dark:text-white">{clients.length}</span>
          </div>
        </div>
      </div>

      {filteredClients.length === 0 ? (
        <div className="py-20 text-center">
          <div className="mx-auto w-12 h-12 rounded-lg bg-slate-100 dark:bg-white/10 flex items-center justify-center text-slate-400">
            <AppIcons.key size={24} />
          </div>
          <p className="mt-4 text-sm font-bold text-slate-700 dark:text-slate-200">
            {searchTerm ? 'No clients match this search.' : 'No clients configured yet.'}
          </p>
          <p className="mt-1 text-[13px] text-slate-500 dark:text-slate-400">
            Create a profile to generate keys, config, and QR access.
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[960px] text-left border-collapse">
            <thead>
              <tr className="bg-slate-50 dark:bg-white/[0.03] border-b border-[var(--divider)] text-[11px] font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                <th className="py-3.5 px-5 font-bold">Client</th>
                <th className="py-3.5 px-4 font-bold">Status</th>
                <th className="py-3.5 px-4 font-bold">VPN IP</th>
                <th className="py-3.5 px-4 font-bold">Bandwidth</th>
                <th className="py-3.5 px-4 font-bold">Handshake</th>
                <th className="py-3.5 px-5 text-right font-bold">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[var(--divider)] text-[13px]">
              {filteredClients.map((client) => (
                <tr key={client.name} className="group hover:bg-slate-50/70 dark:hover:bg-white/[0.03] transition duration-150">
                  <td className="py-4 px-5">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-slate-800 to-slate-600 dark:from-teal-500 dark:to-sky-500 text-white flex items-center justify-center font-extrabold text-sm shadow-sm">
                        {client.name.charAt(0).toUpperCase()}
                      </div>
                      <div className="min-w-0">
                        <div className="font-extrabold text-slate-950 dark:text-white truncate max-w-[220px]">
                          {client.name}
                        </div>
                        <div className="flex items-center gap-1.5 text-[11px] text-slate-500 dark:text-slate-400 font-medium" title={client.public_key}>
                          <AppIcons.key size={14} />
                          <span className="truncate max-w-[210px]">{client.public_key ? `${client.public_key.substring(0, 24)}...` : 'No key'}</span>
                        </div>
                      </div>
                    </div>
                  </td>

                  <td className="py-4 px-4">
                    <StatusBadge online={client.is_online} />
                  </td>

                  <td className="py-4 px-4 font-mono font-bold text-xs text-slate-700 dark:text-slate-200">
                    <span className="inline-flex items-center gap-2 px-2.5 py-1.5 rounded-md bg-slate-100 dark:bg-white/10">
                      <Wifi className="w-3.5 h-3.5 text-teal-500" />
                      {client.ip || '172.24.170.X'}
                    </span>
                  </td>

                  <td className="py-4 px-4">
                    <div className="flex items-center gap-2.5 font-mono text-xs">
                      <span className="inline-flex items-center gap-1.5 text-sky-600 dark:text-sky-300 font-bold">
                        <ArrowDownRight className="w-3.5 h-3.5" />
                        {client.rx_formatted || '0 B'}
                      </span>
                      <span className="inline-flex items-center gap-1.5 text-amber-600 dark:text-amber-300 font-bold">
                        <ArrowUpRight className="w-3.5 h-3.5" />
                        {client.tx_formatted || '0 B'}
                      </span>
                    </div>
                  </td>

                  <td className="py-4 px-4">
                    <div className="flex items-center gap-1.5 text-[12px] font-semibold text-slate-500 dark:text-slate-400">
                      <Clock className="w-3.5 h-3.5" />
                      <span>{client.latest_handshake_text || 'Offline'}</span>
                    </div>
                  </td>

                  <td className="py-4 px-5 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <ActionButton onClick={() => onShowQR(client)} title="Show QR Code">
                        <QrCode className="w-4 h-4" />
                        <span>QR</span>
                      </ActionButton>
                      <ActionButton onClick={() => onDownloadConfig(client.name)} title="Download .conf file" tone="primary">
                        <Download className="w-4 h-4" />
                        <span>Config</span>
                      </ActionButton>
                      <ActionButton onClick={() => setDeleteConfirm(client.name)} title="Delete Client" tone="danger">
                        <Trash2 className="w-4 h-4" />
                      </ActionButton>
                      <button
                        title="More"
                        className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-slate-200 dark:border-white/10 bg-white dark:bg-slate-900 text-slate-500 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 transition"
                      >
                        <MoreHorizontal className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {deleteConfirm && (
        <div className="fixed inset-0 z-50 bg-slate-950/50 dark:bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-white dark:bg-slate-900 max-w-md w-full rounded-lg p-6 border border-[var(--card-border)] shadow-2xl modal-scale-in">
            <div className="flex items-center gap-4 mb-5">
              <div className="p-3 rounded-lg bg-rose-50 dark:bg-rose-500/10 text-rose-500">
                <ShieldAlert className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-lg font-extrabold text-slate-950 dark:text-white">Delete client?</h3>
                <p className="text-[13px] text-slate-500 dark:text-slate-400">This action cannot be undone.</p>
              </div>
            </div>
            <p className="text-[13px] text-slate-600 dark:text-slate-300 mb-6 leading-relaxed">
              Client <strong className="text-slate-950 dark:text-white font-mono font-bold">{deleteConfirm}</strong> will lose tunnel access immediately.
            </p>
            <div className="flex items-center justify-end gap-3">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="px-4 py-2.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-[13px] font-bold transition"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  onDeleteClient(deleteConfirm);
                  setDeleteConfirm(null);
                }}
                className="px-5 py-2.5 rounded-lg bg-rose-600 hover:bg-rose-700 text-white text-[13px] font-bold shadow-lg shadow-rose-600/20 transition active:scale-95"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
