const SSO_API = 'http://127.0.0.1:8080'

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
    maxWidth: 360,
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
  error: {
    color: '#dc2626',
    fontSize: 14,
    margin: '0 0 16px',
    padding: '10px 12px',
    background: '#fef2f2',
    borderRadius: 8,
  },
  hint: {
    color: '#64748b',
    fontSize: 15,
    margin: 0,
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: 16,
  },
  field: {
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
  },
  label: {
    fontSize: 14,
    fontWeight: 500,
    color: '#334155',
  },
  input: {
    width: '100%',
    padding: '10px 12px',
    fontSize: 15,
    border: '1px solid #e2e8f0',
    borderRadius: 8,
    outline: 'none',
  },
  submit: {
    padding: '10px 20px',
    fontSize: 14,
    fontWeight: 500,
    background: '#2563eb',
    color: '#fff',
    border: 'none',
    borderRadius: 8,
    cursor: 'pointer',
    marginTop: 8,
  },
}

function App() {
  const params = new URLSearchParams(window.location.search)
  const service = params.get('service') || ''
  const error = params.get('error')

  return (
    <div style={styles.page}>
      <div style={styles.card}>
        <h1 style={styles.title}>SSO 登录</h1>
        {error && (
          <p style={styles.error}>
            {error === 'invalid_credentials' ? '用户名或密码错误' : error}
          </p>
        )}
        {!service ? (
          <p style={styles.hint}>缺少 service 参数</p>
        ) : (
          <form method="POST" action={`${SSO_API}/login`} style={styles.form}>
            <input type="hidden" name="service" value={service} />
            <div style={styles.field}>
              <label style={styles.label} htmlFor="username">用户名</label>
              <input
                id="username"
                name="username"
                type="text"
                required
                autoComplete="username"
                style={styles.input}
              />
            </div>
            <div style={styles.field}>
              <label style={styles.label} htmlFor="password">密码</label>
              <input
                id="password"
                name="password"
                type="password"
                required
                autoComplete="current-password"
                style={styles.input}
              />
            </div>
            <button type="submit" style={styles.submit}>登录</button>
          </form>
        )}
      </div>
    </div>
  )
}

export default App
