import { useState, useEffect } from 'react'
import './App.css'

const API_BASE = 'http://127.0.0.1:8081'
const LOGIN_URL = `${API_BASE}/auth/login`

function App() {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const err = params.get('error')
    if (err) {
      setError(err)
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [])

  useEffect(() => {
    fetch(`${API_BASE}/api/me`, { credentials: 'include' })
      .then((res) => {
        if (res.ok) return res.json()
        if (res.status === 401) return null
        throw new Error('Failed to fetch user')
      })
      .then((data) => {
        setUser(data)
      })
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="app">
        <p>加载中...</p>
      </div>
    )
  }

  return (
    <div className="app">
      <h1>Google OAuth Demo</h1>
      {error && (
        <p className="error">登录异常: {error}</p>
      )}
      {user ? (
        <div className="user-card">
          <img src={user.picture} alt={user.name} className="avatar" />
          <h2>{user.name}</h2>
          <p className="email">{user.email}</p>
          {user.given_name && <p>Given name: {user.given_name}</p>}
          {user.family_name && <p>Family name: {user.family_name}</p>}
          {user.locale && <p>Locale: {user.locale}</p>}
          <p className="id">ID: {user.id}</p>
        </div>
      ) : (
        <div className="login">
          <a href={LOGIN_URL} className="login-btn">使用 Google 登录</a>
        </div>
      )}
    </div>
  )
}

export default App
