import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import './index.css'

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    console.error("React ErrorBoundary caught an error:", error, errorInfo);
    this.setState({ errorInfo });
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: '30px', background: '#0F172A', color: '#F8FAFC', minHeight: '100vh', fontFamily: 'monospace' }}>
          <h2 style={{ color: '#EF4444', fontSize: '20px', fontWeight: 'bold' }}>⚠️ Application Error</h2>
          <p style={{ marginTop: '10px', color: '#FCA5A5' }}>{this.state.error?.toString()}</p>
          <pre style={{ marginTop: '15px', padding: '15px', background: '#1E293B', borderRadius: '8px', overflow: 'auto', fontSize: '12px', color: '#94A3B8' }}>
            {this.state.errorInfo?.componentStack || this.state.error?.stack}
          </pre>
          <button 
            onClick={() => window.location.reload()}
            style={{ marginTop: '20px', padding: '10px 20px', background: '#6366F1', color: '#fff', border: 'none', borderRadius: '8px', cursor: 'pointer', fontWeight: 'bold' }}
          >
            Reload Page
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>,
)

