import { useState, useEffect } from 'react'

const APP_API = 'http://127.0.0.1:8081'
const APP_ORIGIN = 'http://127.0.0.1:3001'

const styles = {
  page: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 24,
  },
  card: {
    width: '100%',
    maxWidth: 400,
    background: '#fff',
    borderRadius: 12,
    boxShadow: '0 4px 24px rgba(0,0,0,0.08)',
    padding: 32,
  },
  title: {
    margin: '0 0 24px',
    fontSize: 22,
    fontWeight: 600,
    color: '#0f172a',
  },
  loading: {
    color: '#64748b',
    fontSize: 15,
    margin: 0,
  },
  error: {
    color: '#dc2626',
    fontSize: 14,
    margin: '0 0 16px',
    padding: '10px 12px',
    background: '#fef2f2',
    borderRadius: 8,
  },
  userRow: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    flexWrap: 'wrap',
    gap: 16,
    marginBottom: 24,
  },
  userText: {
    margin: 0,
    fontSize: 15,
    color: '#334155',
  },
  userTextStrong: {
    color: '#0f172a',
    fontWeight: 500,
  },
  btn: {
    padding: '10px 20px',
    fontSize: 14,
    fontWeight: 500,
    border: 'none',
    borderRadius: 8,
    cursor: 'pointer',
    transition: 'background 0.15s, transform 0.05s',
  },
  btnPrimary: {
    background: '#2563eb',
    color: '#fff',
  },
  btnSecondary: {
    background: '#f1f5f9',
    color: '#475569',
  },
  guestBlock: {
    textAlign: 'center',
    padding: '24px 0 8px',
  },
  guestText: {
    margin: '0 0 20px',
    fontSize: 15,
    color: '#64748b',
  },
}

function App() {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const fetchMe = () =>
    fetch(`${APP_API}/api/me`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : Promise.reject(res)))
      .then((data) => {
        setUser(data.user)
        setError(null)
      })
      .catch((res) => {
        if (res && res.status === 401) {
          setUser(null)
          return
        }
        setError('请求失败')
        setUser(null)
      })

  useEffect(() => {
    fetchMe().finally(() => setLoading(false))
  }, [])

  const handleLogin = () => {
    window.location.href = `${APP_API}/login`
  }

  const handleLogout = () => {
    fetch(`${APP_API}/api/logout`, { method: 'POST', credentials: 'include' })
      .then(() => {
        setUser(null)
      })
      .catch(() => setUser(null))
  }

  if (loading) {
    return (
      <div style={styles.page}>
        <div style={styles.card}>
          <h1 style={styles.title}>App A</h1>
          <p style={styles.loading}>加载中...</p>
        </div>
      </div>
    )
  }

  return (
    <div style={styles.page}>
      <div style={styles.card}>
        <h1 style={styles.title}>App A</h1>
        {error && <p style={styles.error}>{error}</p>}
        {user ? (
          <>
            <div style={styles.userRow}>
              <p style={styles.userText}>
                <span style={styles.userTextStrong}>{user.name}</span>
                <span>（{user.username}）</span>
              </p>
              <button
                type="button"
                onClick={handleLogout}
                style={{ ...styles.btn, ...styles.btnSecondary }}
                onMouseDown={(e) => (e.currentTarget.style.transform = 'scale(0.98)')}
                onMouseUp={(e) => (e.currentTarget.style.transform = '')}
                onMouseLeave={(e) => (e.currentTarget.style.transform = '')}
              >
                退出
              </button>
            </div>
          </>
        ) : (
          <div style={styles.guestBlock}>
            <p style={styles.guestText}>未登录</p>
            <button
              type="button"
              onClick={handleLogin}
              style={{ ...styles.btn, ...styles.btnPrimary }}
              onMouseDown={(e) => (e.currentTarget.style.transform = 'scale(0.98)')}
              onMouseUp={(e) => (e.currentTarget.style.transform = '')}
              onMouseLeave={(e) => (e.currentTarget.style.transform = '')}
            >
              登录
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default App
