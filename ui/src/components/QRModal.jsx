import React, { useEffect, useState } from 'react';
import { X, QrCode, Download, Copy, Check } from 'lucide-react';

export default function QRModal({ client, onClose }) {
  const [qrCodeData, setQrCodeData] = useState('');
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);

  const token = localStorage.getItem('awg_jwt_token') || '';

  useEffect(() => {
    if (client?.name) {
      fetch(`/api/clients/${client.name}/qr`, {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      })
        .then((res) => {
          if (!res.ok) throw new Error('Failed to fetch QR code');
          return res.json();
        })
        .then((data) => {
          setQrCodeData(data.qr_code_svg || '');
          setLoading(false);
        })
        .catch((err) => {
          console.error(err);
          setLoading(false);
        });
    }
  }, [client, token]);

  const copyConfigText = async () => {
    try {
      const res = await fetch(`/api/clients/${client.name}/download`, {
        headers: {
          Authorization: token ? `Bearer ${token}` : '',
        },
      });
      const text = await res.text();
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error(err);
    }
  };

  const downloadUrl = `/api/clients/${encodeURIComponent(client?.name || '')}/download?token=${encodeURIComponent(token)}`;

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/60 dark:bg-black/75 backdrop-blur-md flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-900 max-w-md w-full rounded-lg p-6 border border-[var(--card-border)] shadow-2xl relative modal-scale-in">
        <div className="flex items-center justify-between border-b border-[var(--divider)] pb-4 mb-5">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-lg bg-teal-50 dark:bg-teal-500/10 text-teal-600 dark:text-teal-300 border border-teal-100 dark:border-teal-500/20">
              <QrCode className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-extrabold text-slate-950 dark:text-white">QR Code: {client?.name}</h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 font-mono">IP: {client?.ip}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="flex flex-col items-center justify-center py-4">
          {loading ? (
            <div className="w-52 h-52 flex items-center justify-center text-slate-500 text-xs font-mono">
              Loading QR code...
            </div>
          ) : qrCodeData ? (
            <div className="p-4 rounded-lg bg-white shadow-2xl border border-slate-200 mb-4">
              <img src={qrCodeData} alt="AmneziaWG QR Code" className="w-56 h-56 object-contain" />
            </div>
          ) : (
            <div className="text-rose-400 text-xs py-8">Failed to render QR code</div>
          )}

          <p className="text-xs text-slate-500 dark:text-slate-400 text-center max-w-xs mb-6 leading-relaxed">
            Open the <strong>AmneziaVPN</strong> or <strong>AmneziaWG</strong> app and scan this code to instantly pair your device.
          </p>

          <div className="flex items-center gap-3 w-full">
            <button
              onClick={copyConfigText}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-semibold transition active:scale-95"
            >
              {copied ? <Check className="w-4 h-4 text-emerald-500" /> : <Copy className="w-4 h-4 text-teal-500" />}
              <span>{copied ? 'Copied' : 'Copy Config'}</span>
            </button>

            <a
              href={downloadUrl}
              download={`${client?.name}.conf`}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-teal-600 hover:bg-teal-700 text-white text-xs font-bold shadow-lg shadow-teal-600/20 transition active:scale-95"
            >
              <Download className="w-4 h-4" />
              <span>Download .conf</span>
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
