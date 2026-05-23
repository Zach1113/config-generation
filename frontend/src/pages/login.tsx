import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/lib/auth"
import {
  getAuthConfig,
  login as apiLogin,
  register as apiRegister,
  type AuthConfig,
} from "@/api/auth"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { AxiosError } from "axios"

function getErrorMessage(err: unknown): string {
  if (err instanceof AxiosError && err.response?.data?.error) {
    return err.response.data.error
  }
  return "Something went wrong"
}

export default function LoginPage() {
  const { login, user, loading } = useAuth()
  const navigate = useNavigate()
  const [authConfig, setAuthConfig] = useState<AuthConfig | null>(null)

  // Login form state
  const [loginUsername, setLoginUsername] = useState("")
  const [loginPassword, setLoginPassword] = useState("")
  const [loginError, setLoginError] = useState("")
  const [loginLoading, setLoginLoading] = useState(false)

  // Register form state
  const [regUsername, setRegUsername] = useState("")
  const [regDisplayName, setRegDisplayName] = useState("")
  const [regPassword, setRegPassword] = useState("")
  const [regConfirm, setRegConfirm] = useState("")
  const [regError, setRegError] = useState("")
  const [regLoading, setRegLoading] = useState(false)

  useEffect(() => {
    getAuthConfig()
      .then(setAuthConfig)
      .catch(() => {
        setAuthConfig({
          oidc_enabled: false,
          oidc_provider_name: "SSO",
          password_login_enabled: true,
          registration_enabled: true,
        })
      })
  }, [])

  useEffect(() => {
    if (!loading && user) {
      navigate("/projects", { replace: true })
    }
  }, [loading, navigate, user])

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setLoginError("")
    if (!loginUsername.trim() || !loginPassword) {
      setLoginError("Username and password are required")
      return
    }
    setLoginLoading(true)
    try {
      const res = await apiLogin(loginUsername.trim(), loginPassword)
      login(res)
      navigate("/projects", { replace: true })
    } catch (err) {
      setLoginError(getErrorMessage(err))
    } finally {
      setLoginLoading(false)
    }
  }

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault()
    setRegError("")
    if (!regUsername.trim() || !regPassword) {
      setRegError("Username and password are required")
      return
    }
    if (regPassword.length < 8) {
      setRegError("Password must be at least 8 characters")
      return
    }
    if (regPassword !== regConfirm) {
      setRegError("Passwords do not match")
      return
    }
    setRegLoading(true)
    try {
      const res = await apiRegister(
        regUsername.trim(),
        regPassword,
        regDisplayName.trim() || undefined,
      )
      login(res)
      navigate("/projects", { replace: true })
    } catch (err) {
      setRegError(getErrorMessage(err))
    } finally {
      setRegLoading(false)
    }
  }

  function handleSSOLogin() {
    const returnTo = encodeURIComponent("/projects")
    window.location.href = `/api/auth/oidc/login?return_to=${returnTo}`
  }

  const showPasswordLogin = authConfig?.password_login_enabled ?? true
  const showRegistration = authConfig?.registration_enabled ?? true
  const showSSO = authConfig?.oidc_enabled ?? false
  const defaultTab = showPasswordLogin ? "login" : "register"

  return (
    <div className="flex min-h-screen items-center justify-center">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Config Generation</CardTitle>
        </CardHeader>
        <CardContent>
          {showSSO && (
            <div className="mb-4">
              <Button type="button" className="w-full" onClick={handleSSOLogin}>
                Sign in with {authConfig?.oidc_provider_name ?? "SSO"}
              </Button>
            </div>
          )}
          {(showPasswordLogin || showRegistration) && (
            <Tabs defaultValue={defaultTab}>
              {showPasswordLogin && showRegistration && (
                <TabsList className="w-full">
                  <TabsTrigger value="login" className="flex-1">
                    Login
                  </TabsTrigger>
                  <TabsTrigger value="register" className="flex-1">
                    Register
                  </TabsTrigger>
                </TabsList>
              )}

              {showPasswordLogin && (
                <TabsContent value="login">
                  <form onSubmit={handleLogin} className="space-y-4 pt-4">
                    <div className="space-y-2">
                      <Label htmlFor="login-username">Username</Label>
                      <Input
                        id="login-username"
                        type="text"
                        autoComplete="username"
                        placeholder="Enter your username"
                        value={loginUsername}
                        onChange={(e) => setLoginUsername(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="login-password">Password</Label>
                      <Input
                        id="login-password"
                        type="password"
                        autoComplete="current-password"
                        placeholder="Enter your password"
                        value={loginPassword}
                        onChange={(e) => setLoginPassword(e.target.value)}
                      />
                    </div>
                    {loginError && (
                      <p className="text-sm text-destructive">{loginError}</p>
                    )}
                    <Button type="submit" className="w-full" disabled={loginLoading}>
                      {loginLoading ? "Signing in..." : "Sign In"}
                    </Button>
                  </form>
                </TabsContent>
              )}

              {showRegistration && (
                <TabsContent value="register">
                  <form onSubmit={handleRegister} className="space-y-4 pt-4">
                    <div className="space-y-2">
                      <Label htmlFor="reg-username">Username</Label>
                      <Input
                        id="reg-username"
                        type="text"
                        autoComplete="username"
                        placeholder="Choose a username"
                        value={regUsername}
                        onChange={(e) => setRegUsername(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="reg-display-name">Display Name</Label>
                      <Input
                        id="reg-display-name"
                        type="text"
                        autoComplete="name"
                        placeholder="Optional"
                        value={regDisplayName}
                        onChange={(e) => setRegDisplayName(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="reg-password">Password</Label>
                      <Input
                        id="reg-password"
                        type="password"
                        autoComplete="new-password"
                        placeholder="At least 8 characters"
                        value={regPassword}
                        onChange={(e) => setRegPassword(e.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="reg-confirm">Confirm Password</Label>
                      <Input
                        id="reg-confirm"
                        type="password"
                        autoComplete="new-password"
                        placeholder="Repeat your password"
                        value={regConfirm}
                        onChange={(e) => setRegConfirm(e.target.value)}
                      />
                    </div>
                    {regError && (
                      <p className="text-sm text-destructive">{regError}</p>
                    )}
                    <Button type="submit" className="w-full" disabled={regLoading}>
                      {regLoading ? "Creating account..." : "Create Account"}
                    </Button>
                  </form>
                </TabsContent>
              )}
            </Tabs>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
