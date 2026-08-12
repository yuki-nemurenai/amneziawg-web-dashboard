import React, { useState } from 'react';
import { X, Save, Check, Shield } from 'lucide-react';
import { generateQUICPayload } from '../utils/quicGenerator';

export default function AmneziaConfigModal({ serverConfig, onClose, onSave }) {
  const [formData, setFormData] = useState({
    obfuscation: {
      jc: serverConfig?.obfuscation?.jc || '6',
      jmin: serverConfig?.obfuscation?.jmin || '10',
      jmax: serverConfig?.obfuscation?.jmax || '50',
      s1: serverConfig?.obfuscation?.s1 || '136',
      s2: serverConfig?.obfuscation?.s2 || '20',
      s3: serverConfig?.obfuscation?.s3 || '36',
      s4: serverConfig?.obfuscation?.s4 || '10',
      h1: serverConfig?.obfuscation?.h1 || '1027326130-2124574311',
      h2: serverConfig?.obfuscation?.h2 || '2128283030-2131527662',
      h3: serverConfig?.obfuscation?.h3 || '2139330923-2144622857',
      h4: serverConfig?.obfuscation?.h4 || '2145845262-2147466530',
      i1: serverConfig?.obfuscation?.i1 || '<r 2><b 0x858000010001000000000669636c6f756403636f6d0000010001c00c000100010000105a00044d583737>',
      i2: serverConfig?.obfuscation?.i2 || '',
      i3: serverConfig?.obfuscation?.i3 || '',
      i4: serverConfig?.obfuscation?.i4 || '',
      i5: serverConfig?.obfuscation?.i5 || '',
    },
  });

  const [domainInputs, setDomainInputs] = useState({
    i1: '', i2: '', i3: '', i4: '', i5: ''
  });

  const [loading, setLoading] = useState(false);
  const [savedSuccess, setSavedSuccess] = useState(false);
  const [error, setError] = useState('');

  const handleObfChange = (field, val) => {
    setFormData((prev) => ({
      ...prev,
      obfuscation: { ...prev.obfuscation, [field]: val },
    }));
  };

  const handleDomainChange = async (field, domain) => {
    setDomainInputs((prev) => ({ ...prev, [field]: domain }));
    if (domain.trim()) {
      try {
        const payload = await generateQUICPayload(domain.trim(), 4);
        handleObfChange(field, payload);
      } catch (err) {
        console.error('Failed to generate QUIC payload:', err);
      }
    } else {
      handleObfChange(field, '');
    }
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
      setError(err.message || 'Failed to save Amnezia configuration');
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
            <div className="p-2.5 rounded-lg bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-300 border border-amber-100 dark:border-amber-500/20">
              <Shield className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-lg font-extrabold text-slate-900 dark:text-white">AmneziaWG Configuration</h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">Configure anti-censorship parameters and headers</p>
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
          <div className="space-y-4">
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">Jc</label>
                <input
                  type="text"
                  value={formData.obfuscation.jc}
                  onChange={(e) => handleObfChange('jc', e.target.value)}
                  className="w-full px-3.5 py-2 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">Jmin</label>
                <input
                  type="text"
                  value={formData.obfuscation.jmin}
                  onChange={(e) => handleObfChange('jmin', e.target.value)}
                  className="w-full px-3.5 py-2 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">Jmax</label>
                <input
                  type="text"
                  value={formData.obfuscation.jmax}
                  onChange={(e) => handleObfChange('jmax', e.target.value)}
                  className="w-full px-3.5 py-2 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
                />
              </div>
            </div>

            <div className="grid grid-cols-4 gap-3">
              {['s1', 's2', 's3', 's4'].map((sKey) => (
                <div key={sKey}>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1 uppercase">{sKey}</label>
                  <input
                    type="text"
                    value={formData.obfuscation[sKey]}
                    onChange={(e) => handleObfChange(sKey, e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
                  />
                </div>
              ))}
            </div>

            <div className="space-y-2.5">
              {['h1', 'h2', 'h3', 'h4'].map((hKey) => (
                <div key={hKey}>
                  <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1 uppercase">{hKey}</label>
                  <input
                    type="text"
                    value={formData.obfuscation[hKey]}
                    onChange={(e) => handleObfChange(hKey, e.target.value)}
                    className="w-full px-3.5 py-2 rounded-lg finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
                  />
                </div>
              ))}
            </div>

            <div className="space-y-4">
              {['i1', 'i2', 'i3', 'i4', 'i5'].map((iKey) => (
                <div key={iKey} className="p-3 bg-slate-50 dark:bg-slate-800/30 rounded-lg border border-[var(--divider)]">
                  <div className="flex items-center justify-between mb-2">
                    <label className="text-xs font-semibold text-slate-700 dark:text-slate-300 uppercase">{iKey} (Junk Magic Payload)</label>
                  </div>
                  
                  <div className="space-y-3">
                    <div>
                      <label className="block text-[10px] font-semibold text-slate-500 dark:text-slate-400 mb-1">
                        Generate from Domain (QUIC SNI)
                      </label>
                      <input
                        type="text"
                        value={domainInputs[iKey]}
                        onChange={(e) => handleDomainChange(iKey, e.target.value)}
                        placeholder="e.g. example.com"
                        className="w-full px-3 py-1.5 rounded-md finebank-input text-slate-900 dark:text-slate-100 text-xs font-mono"
                      />
                    </div>
                    
                    <div>
                      <label className="block text-[10px] font-semibold text-slate-500 dark:text-slate-400 mb-1">
                        Raw Payload Code
                      </label>
                      <textarea
                        rows={2}
                        value={formData.obfuscation[iKey]}
                        onChange={(e) => handleObfChange(iKey, e.target.value)}
                        className="w-full p-2.5 rounded-md finebank-input text-slate-900 dark:text-slate-100 text-[10px] font-mono shadow-inner"
                      />
                    </div>
                  </div>
                </div>
              ))}
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
