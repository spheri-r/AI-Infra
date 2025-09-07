import { useEffect, useState } from 'react'
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from '@/components/ui/sonner'
import { Login } from '@/components/auth/Login'
import { Register } from '@/components/auth/Register'
import { Dashboard } from '@/components/dashboard/Dashboard'
import { Layout } from '@/components/layout/Layout'
import { usersAPI, User } from '@/lib/api'

function App() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('auth-token')
    if (token) {
      usersAPI.getCurrentUser()
        .then(setUser)
        .catch(() => {
          localStorage.removeItem('auth-token')
        })
        .finally(() => setLoading(false))
    } else {
      setLoading(false)
    }
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-primary"></div>
      </div>
    )
  }

  return (
    <Router>
      <div className="min-h-screen bg-background">
        <Routes>
          <Route
            path="/login"
            element={!user ? <Login onLogin={setUser} /> : <Navigate to="/dashboard" />}
          />
          <Route
            path="/register"
            element={!user ? <Register onRegister={setUser} /> : <Navigate to="/dashboard" />}
          />
          <Route
            path="/dashboard/*"
            element={
              user ? (
                <Layout>
                  <Dashboard user={user} />
                </Layout>
              ) : (
                <Navigate to="/login" />
              )
            }
          />
          <Route path="/" element={<Navigate to="/dashboard" />} />
        </Routes>
        <Toaster />
      </div>
    </Router>
  )
}

export default App 