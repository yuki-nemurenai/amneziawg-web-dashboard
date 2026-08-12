import React, { useEffect, useState, useCallback } from 'react';
import Sidebar from './components/Sidebar';
import Topbar from './components/Topbar';
import Dashboard from './components/Dashboard';
import ClientList from './components/ClientList';
import AddClientModal from './components/AddClientModal';
import QRModal from './components/QRModal';
import NetworkSettingsModal from './components/NetworkSettingsModal';
import AmneziaConfigModal from './components/AmneziaConfigModal';
import ChangePasswordModal from './components/ChangePasswordModal';
import LoginModal from './components/LoginModal';

export default function App() {
  const [authStatus, setAuthStatus] = useState(null);
  const [authToken, setAuthToken] = useState(localStorage.getItem('awg_jwt_token') || '');
  const [currentUser, setCurrentUser] = useState(null);

  // Default theme is 'light' as requested by user
  const [theme, setTheme] = useState(localStorage.getItem('awg_theme') || 'light');

  const [activeTab, setActiveTab] = useState('dashboard');
  const [searchTerm, setSearchTerm] = useState('');

  const [status, setStatus] = useState(null);
  const [serverConfig, setServerConfig] = useState(null);
  const [clients, setClients] = useState([]);
  const [loading, setLoading] = useState(true);
  const [events, setEvents] = useState([]);

  const logEvent = useCallback((message, type = 'info') => {
    setEvents(prev => [{ id: Date.now(), time: new Date().toLocaleTimeString(), message, type }, ...prev].slice(0, 50));
  }, []);

  // Modals
  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedQRClient, setSelectedQRClient] = useState(null);
  const [showNetworkModal, setShowNetworkModal] = useState(false);
  const [showAmneziaModal, setShowAmneziaModal] = useState(false);
  const [showChangePasswordModal, setShowChangePasswordModal] = useState(false);

  // Manage Light / Dark class on HTML root element
  useEffect(() => {
    const root = document.documentElement;
    if (theme === 'dark') {
      root.classList.add('dark');
      root.classList.remove('light');
    } else {
      root.classList.add('light');
      root.classList.remove('dark');
    }
    localStorage.setItem('awg_theme', theme);
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prev) => (prev === 'dark' ? 'light' : 'dark'));
  };

  // Check auth status
  const checkAuthStatus = async () => {
    try {
      const res = await fetch('/api/auth/status');
      const data = await res.json();
      setAuthStatus(data);
    } catch (err) {
      console.error('Error checking auth status:', err);
    }
  };

  const authenticatedFetch = useCallback(
    async (url, options = {}) => {
      const headers = {
        ...options.headers,
        Authorization: authToken ? `Bearer ${authToken}` : '',
      };

      const res = await fetch(url, { ...options, headers });
      if (res.status === 401) {
        localStorage.removeItem('awg_jwt_token');
        setAuthToken('');
        setCurrentUser(null);
        checkAuthStatus();
        throw new Error('Unauthorized');
      }
      return res;
    },
    [authToken]
  );

  const fetchUserData = useCallback(async () => {
    if (!authToken) return;
    try {
      const res = await authenticatedFetch('/api/auth/me');
      if (res.ok) {
        const data = await res.json();
        setCurrentUser(data);
      }
    } catch (err) {
      console.error('Failed to fetch user data:', err);
    }
  }, [authToken, authenticatedFetch]);

  const fetchData = useCallback(async () => {
    if (!authToken) {
      setLoading(false);
      return;
    }

    try {
      const [statusRes, serverRes, clientsRes] = await Promise.all([
        authenticatedFetch('/api/status').then((r) => r.json()),
        authenticatedFetch('/api/server').then((r) => r.json()),
        authenticatedFetch('/api/clients').then((r) => r.json()),
      ]);

      setStatus(statusRes);
      setServerConfig(serverRes);
      setClients(Array.isArray(clientsRes) ? clientsRes : []);
    } catch (err) {
      console.error('Error fetching server data:', err);
    } finally {
      setLoading(false);
    }
  }, [authToken, authenticatedFetch]);

  useEffect(() => {
    checkAuthStatus();
  }, []);

  useEffect(() => {
    if (authToken) {
      fetchUserData();
      fetchData();
      const interval = setInterval(fetchData, 5000);
      return () => clearInterval(interval);
    } else {
      setLoading(false);
    }
  }, [authToken, fetchUserData, fetchData]);

  const handleLogin = async (credentials) => {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Login failed');
    }
    localStorage.setItem('awg_jwt_token', data.token);
    setAuthToken(data.token);
    setCurrentUser(data.user);
    logEvent(`User ${data.user.username} logged in`, 'success');
  };

  const handleSetup = async (credentials) => {
    const res = await fetch('/api/auth/setup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Setup failed');
    }
    localStorage.setItem('awg_jwt_token', data.token);
    setAuthToken(data.token);
    setCurrentUser(data.user);
    setAuthStatus({ needs_setup: false });
  };

  const handleLogout = () => {
    localStorage.removeItem('awg_jwt_token');
    setAuthToken('');
    setCurrentUser(null);
    checkAuthStatus();
  };

  const handleCreateClient = async (payload) => {
    const res = await authenticatedFetch('/api/clients', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Failed to create client');
    }
    fetchData();
    logEvent(`Client ${payload.name} created successfully`, 'success');
    return data;
  };

  const handleDeleteClient = async (name) => {
    try {
      const res = await authenticatedFetch(`/api/clients/${encodeURIComponent(name)}`, {
        method: 'DELETE',
      });
      if (!res.ok) {
        const data = await res.json();
        alert(data.error || 'Failed to delete client');
      } else {
        logEvent(`Client ${name} deleted`, 'info');
      }
      fetchData();
    } catch (err) {
      console.error(err);
    }
  };

  const handleDownloadConfig = (name) => {
    window.location.href = `/api/clients/${encodeURIComponent(name)}/download?token=${encodeURIComponent(authToken)}`;
  };

  const handleSaveSettings = async (newConfig) => {
    const res = await authenticatedFetch('/api/server', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newConfig),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Failed to update settings');
    }
    logEvent('Server settings updated', 'success');
    fetchData();
  };

  const handleChangePassword = async (payload) => {
    const res = await authenticatedFetch('/api/auth/change-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || 'Failed to change password');
    }
    logEvent('Administrator password updated successfully', 'success');
  };

  // Render Login or Setup Wizard if unauthenticated
  if (!authToken || authStatus?.needs_setup) {
    return (
      <LoginModal
        needsSetup={Boolean(authStatus?.needs_setup)}
        onLogin={handleLogin}
        onSetup={handleSetup}
      />
    );
  }

  return (
    <div className={`min-h-screen flex bg-[var(--bg-main)] relative font-sans text-slate-900 dark:text-slate-100 transition-colors duration-200 ${theme}`}>
      <Sidebar
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        clientsCount={clients.length}
        currentUser={currentUser}
        onLogout={handleLogout}
        onChangePassword={() => setShowChangePasswordModal(true)}
        onOpenNetworkSettings={() => setShowNetworkModal(true)}
        onOpenAmneziaSettings={() => setShowAmneziaModal(true)}
      />

      <div className="flex-1 flex flex-col min-w-0 h-screen overflow-y-auto relative z-10">
        <Topbar
          status={status}
          activeTab={activeTab}
          setActiveTab={setActiveTab}
          onOpenAddClient={() => setShowAddModal(true)}
          onOpenSettings={() => setShowNetworkModal(true)}
          onChangePassword={() => setShowChangePasswordModal(true)}
          searchTerm={searchTerm}
          setSearchTerm={setSearchTerm}
          theme={theme}
          toggleTheme={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
          events={events}
          currentUser={currentUser}
        />

        <main className="px-5 sm:px-8 pb-16 flex-1 max-w-[1480px] w-full">
          {loading ? (
            <div className="py-24 text-center text-slate-400 dark:text-slate-500 font-medium text-sm flex flex-col items-center gap-3">
              <div className="w-8 h-8 rounded-full border-2 border-indigo-500 border-t-transparent animate-spin" />
              <span>Loading AmneziaWG control center...</span>
            </div>
          ) : (
            <div className="space-y-6">
              {activeTab !== 'users' && (
                <Dashboard status={status} serverConfig={serverConfig} />
              )}

              <ClientList
                clients={clients}
                searchTerm={searchTerm}
                onShowQR={(client) => setSelectedQRClient(client)}
                onDeleteClient={handleDeleteClient}
                onDownloadConfig={handleDownloadConfig}
              />
            </div>
          )}
        </main>
      </div>

      {showAddModal && (
        <AddClientModal
          onClose={() => setShowAddModal(false)}
          onSubmit={handleCreateClient}
        />
      )}

      {selectedQRClient && (
        <QRModal
          client={selectedQRClient}
          onClose={() => setSelectedQRClient(null)}
        />
      )}

      {showNetworkModal && (
        <NetworkSettingsModal
          serverConfig={serverConfig}
          status={status}
          onClose={() => setShowNetworkModal(false)}
          onSave={handleSaveSettings}
        />
      )}

      {showAmneziaModal && (
        <AmneziaConfigModal
          serverConfig={serverConfig}
          onClose={() => setShowAmneziaModal(false)}
          onSave={handleSaveSettings}
        />
      )}

      <ChangePasswordModal
        isOpen={showChangePasswordModal}
        onClose={() => setShowChangePasswordModal(false)}
        onPasswordChanged={handleChangePassword}
      />
    </div>
  );
}
