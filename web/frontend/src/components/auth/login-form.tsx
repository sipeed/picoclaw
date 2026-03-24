import { useState } from "react"
import { useTranslation } from "react-i18next"
import { IconEye, IconEyeOff, IconLoader2 } from "@tabler/icons-react"

import { useAuth } from "@/features/auth"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"

export function LoginForm() {
  const { t } = useTranslation()
  const { login } = useAuth()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  const isNotSecure = typeof window !== "undefined" && window.location.protocol === "http:" && window.location.hostname !== "localhost" && window.location.hostname !== "127.0.0.1"

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setLoading(true)

    try {
      const result = await login(username, password)
      if (!result.success) {
        setError(result.message || t("auth.loginFailed", "Login failed"))
      }
    } catch {
      setError(t("auth.networkError", "Network error"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {isNotSecure && (
        <Alert variant="destructive">
          <AlertDescription>
            {t("auth.insecureConnection", "Warning: Connection is not secure. Passwords will be sent in plain text.")}
          </AlertDescription>
        </Alert>
      )}

      <div className="space-y-2">
        <Label htmlFor="username">{t("auth.username", "Username")}</Label>
        <Input
          id="username"
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
          autoComplete="username"
          autoFocus
          placeholder={t("auth.usernamePlaceholder", "Enter your username")}
          disabled={loading}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="password">{t("auth.password", "Password")}</Label>
        <div className="relative">
          <Input
            id="password"
            type={showPassword ? "text" : "password"}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="current-password"
            placeholder={t("auth.passwordPlaceholder", "Enter your password")}
            disabled={loading}
            className="pr-10"
          />
          <button
            type="button"
            onClick={() => setShowPassword(!showPassword)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            tabIndex={-1}
          >
            {showPassword ? <IconEyeOff className="h-4 w-4" /> : <IconEye className="h-4 w-4" />}
          </button>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Button type="submit" className="w-full" disabled={loading || !username || !password}>
        {loading ? (
          <>
            <IconLoader2 className="mr-2 h-4 w-4 animate-spin" />
            {t("auth.loggingIn", "Logging in...")}
          </>
        ) : (
          t("auth.login", "Login")
        )}
      </Button>
    </form>
  )
}
