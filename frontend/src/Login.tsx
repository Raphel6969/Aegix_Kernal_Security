import React, { useState, useEffect } from 'react';
import { useAuth } from './AuthContext';
import {
  ShieldCheck, Loader2, ArrowRight, Mail, Lock,
  Cpu, Brain, CheckCircle2, AlertTriangle, Terminal
} from 'lucide-react';
import aegixLogo from './assets/aegix-logo.png';
import { API_URL } from './config';

export const Login: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isRegistering, setIsRegistering] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [isCheckingOAuth, setIsCheckingOAuth] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { login } = useAuth();

  useEffect(() => {
    // Attempt to silently log in if returning from OAuth (valid HTTP-only cookie)
    fetch(`${API_URL}/auth/refresh`, { method: 'POST', credentials: 'include' })
      .then(r => r.ok ? r.json() : Promise.reject())
      .then(data => {
        if (data.user && data.access_token) {
          login(data.user, data.access_token);
        }
      })
      .catch(() => {
        setIsCheckingOAuth(false);
      });
  }, [login]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setIsSubmitting(true);
    
    try {
      const endpoint = isRegistering ? `${API_URL}/auth/register` : `${API_URL}/auth/login`;
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
        credentials: 'include'
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || (isRegistering ? 'Registration failed' : 'Invalid credentials'));
      }

      if (isRegistering) {
        setSuccess('Registration successful! You can now log in.');
        setIsRegistering(false);
        setPassword('');
      } else {
        const data = await res.json();
        login(data.user, data.access_token);
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isCheckingOAuth) {
    return (
      <div style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '16px',
        background: 'var(--bg-main)',
      }}>
        <Loader2 className="animate-spin" size={38} color="var(--neon-cyan)" />
        <span style={{
          color: 'var(--text-secondary)',
          fontFamily: 'var(--font-tech)',
          fontSize: '13px',
          letterSpacing: '1px'
        }}>
          VERIFYING OPERATOR CREDENTIALS...
        </span>
      </div>
    );
  }

  return (
    <div className="login-page">
      {/* Ambient glowing atmosphere matching Hero section */}
      <div className="login-glow login-glow--green" aria-hidden />
      <div className="login-glow login-glow--cyan" aria-hidden />
      <div className="login-glow login-glow--center" aria-hidden />

      <div className="login-container">
        <div className="login-hero-grid">
          {/* Left Hero Section: Brand, Mission & Security Architecture */}
          <div className="login-hero-info">
            <div className="home-hero__pill" style={{ width: 'fit-content' }}>
              <span className="home-hero__pill-dot" />
              AEGIX GATEWAY · V2.4.1 · ACTIVE DEFENSE
            </div>

            <h1 className="login-hero__headline">
              Autonomous Defense at{' '}
              <span className="home-hero__glow">Kernel Velocity</span>
            </h1>

            <p className="login-hero__desc">
              The world's first cascading AI security system. Intercepting unauthorized process
              executions at the kernel boundary with sub-millisecond precision, behavioral machine
              learning, and autonomous remediation.
            </p>

            <div className="login-features-list">
              <div className="login-feature-item">
                <div className="login-feature-item__icon login-feature-item__icon--cyan">
                  <Cpu size={18} />
                </div>
                <div>
                  <div className="login-feature-item__title">Tier A — Deterministic Rule Engine</div>
                  <div className="login-feature-item__desc">
                    Pattern matching, Shannon entropy calculation, and memory profiling in under 1ms.
                  </div>
                </div>
              </div>

              <div className="login-feature-item">
                <div className="login-feature-item__icon login-feature-item__icon--orange">
                  <Brain size={18} />
                </div>
                <div>
                  <div className="login-feature-item__title">Tier B — Pure Go ML Scorer</div>
                  <div className="login-feature-item__desc">
                    TF-IDF n-gram vectorization and Logistic Regression inference with 94.2% accuracy.
                  </div>
                </div>
              </div>

              <div className="login-feature-item">
                <div className="login-feature-item__icon login-feature-item__icon--green">
                  <ShieldCheck size={18} />
                </div>
                <div>
                  <div className="login-feature-item__title">Layer 1 — eBPF Kernel Guard</div>
                  <div className="login-feature-item__desc">
                    Tracepoint hook on sys_enter_execve with zero-copy ringbuffer streaming to edge cache.
                  </div>
                </div>
              </div>
            </div>

            <div className="login-telemetry-badge">
              <span className="login-telemetry-dot" />
              SEC_NODE_ID // 0x7F0001 · ENCRYPTED JWT · ZERO-TRUST
            </div>
          </div>

          {/* Right Hero Section: Glassmorphic Terminal Card */}
          <div className="glass-card login-terminal-card home-pipeline--glow">
            <div style={{ display: 'flex', alignItems: 'center', gap: '14px', marginBottom: '20px' }}>
              <img
                src={aegixLogo}
                alt="Aegix"
                style={{
                  width: '42px',
                  height: '42px',
                  borderRadius: '10px',
                  border: '1px solid rgba(0, 210, 255, 0.3)',
                  boxShadow: '0 0 16px rgba(0, 210, 255, 0.25)',
                  padding: '2px',
                  background: 'rgba(0, 0, 0, 0.4)'
                }}
              />
              <div>
                <div style={{
                  fontSize: '11px',
                  fontFamily: 'var(--font-tech)',
                  color: 'var(--neon-cyan)',
                  letterSpacing: '1.2px'
                }}>
                  TERMINAL ACCESS NODE
                </div>
                <h2 style={{
                  margin: '2px 0 0 0',
                  fontSize: '18px',
                  fontWeight: 700,
                  color: 'var(--text-primary)'
                }}>
                  {isRegistering ? 'Register Operator' : 'Operator Authentication'}
                </h2>
              </div>
            </div>

            {/* Toggle Tabs */}
            <div className="login-tabs">
              <button
                type="button"
                className={`login-tab-btn ${!isRegistering ? 'login-tab-btn--active' : ''}`}
                onClick={() => {
                  setIsRegistering(false);
                  setError('');
                  setSuccess('');
                }}
              >
                SIGN IN
              </button>
              <button
                type="button"
                className={`login-tab-btn ${isRegistering ? 'login-tab-btn--active' : ''}`}
                onClick={() => {
                  setIsRegistering(true);
                  setError('');
                  setSuccess('');
                }}
              >
                CREATE ACCOUNT
              </button>
            </div>

            {/* Form */}
            <form onSubmit={handleSubmit}>
              <div className="login-input-group">
                <label className="login-input-label">
                  <span>Operator Identity</span>
                  <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>EMAIL</span>
                </label>
                <div className="login-input-wrapper">
                  <Mail size={16} className="login-input-icon" />
                  <input
                    type="email"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                    className="login-input"
                    placeholder="operator@aegix.internal"
                    required
                    autoComplete="email"
                  />
                </div>
              </div>

              <div className="login-input-group">
                <label className="login-input-label">
                  <span>Passphrase</span>
                  <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>KEY</span>
                </label>
                <div className="login-input-wrapper">
                  <Lock size={16} className="login-input-icon" />
                  <input
                    type="password"
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                    className="login-input"
                    placeholder="••••••••••••"
                    required
                    autoComplete={isRegistering ? 'new-password' : 'current-password'}
                  />
                </div>
              </div>

              {/* Status messages */}
              {error && (
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '10px 14px',
                  borderRadius: 'var(--radius-sm)',
                  background: 'rgba(255, 77, 77, 0.1)',
                  border: '1px solid rgba(255, 77, 77, 0.3)',
                  color: 'var(--neon-red)',
                  fontSize: '12px',
                  marginBottom: '14px'
                }}>
                  <AlertTriangle size={15} style={{ flexShrink: 0 }} />
                  <span>{error}</span>
                </div>
              )}

              {success && (
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '10px 14px',
                  borderRadius: 'var(--radius-sm)',
                  background: 'rgba(0, 255, 157, 0.1)',
                  border: '1px solid rgba(0, 255, 157, 0.3)',
                  color: 'var(--neon-green)',
                  fontSize: '12px',
                  marginBottom: '14px'
                }}>
                  <CheckCircle2 size={15} style={{ flexShrink: 0 }} />
                  <span>{success}</span>
                </div>
              )}

              <button
                type="submit"
                disabled={isSubmitting}
                className="btn-hero-primary login-submit-btn"
              >
                {isSubmitting ? (
                  <>
                    <Loader2 className="animate-spin" size={16} />
                    <span>AUTHENTICATING...</span>
                  </>
                ) : (
                  <>
                    <span>{isRegistering ? 'INITIALIZE CREDENTIALS' : 'ENTER COMMAND CENTER'}</span>
                    <ArrowRight size={16} />
                  </>
                )}
              </button>
            </form>

            {/* SSO Providers */}
            <div style={{ display: 'flex', alignItems: 'center', margin: '24px 0 16px 0' }}>
              <div style={{ flex: 1, height: '1px', backgroundColor: 'var(--border-color)' }} />
              <span style={{
                padding: '0 12px',
                color: 'var(--text-muted)',
                fontSize: '10px',
                fontFamily: 'var(--font-tech)',
                letterSpacing: '1px'
              }}>
                OR CONNECT WITH SSO
              </span>
              <div style={{ flex: 1, height: '1px', backgroundColor: 'var(--border-color)' }} />
            </div>

            <div style={{ display: 'flex', gap: '12px' }}>
              <button
                type="button"
                className="login-oauth-btn login-oauth-btn--github"
                onClick={() => window.location.href = `${API_URL}/auth/github/login`}
                title="Authenticate with GitHub"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <path fillRule="evenodd" clipRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"/>
                </svg>
                <span>GitHub SSO</span>
              </button>

              <button
                type="button"
                className="login-oauth-btn login-oauth-btn--google"
                onClick={() => window.location.href = `${API_URL}/auth/google/login`}
                title="Authenticate with Google"
              >
                <svg width="18" height="18" viewBox="0 0 24 24">
                  <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
                  <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
                  <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"/>
                  <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"/>
                </svg>
                <span>Google</span>
              </button>
            </div>

            <div style={{ marginTop: '20px', textAlign: 'center' }}>
              <button
                type="button"
                style={{
                  fontSize: '12px',
                  border: 'none',
                  background: 'none',
                  color: 'var(--text-muted)',
                  cursor: 'pointer',
                  transition: 'color 0.2s',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '6px'
                }}
                onMouseEnter={e => (e.currentTarget.style.color = 'var(--neon-cyan)')}
                onMouseLeave={e => (e.currentTarget.style.color = 'var(--text-muted)')}
                onClick={() => {
                  setIsRegistering(!isRegistering);
                  setError('');
                  setSuccess('');
                }}
              >
                <Terminal size={13} />
                {isRegistering
                  ? 'Already registered? Return to operator sign-in'
                  : 'Need new access credentials? Register administrator'}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
