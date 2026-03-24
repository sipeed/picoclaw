import { useState } from "react"
import { useTranslation } from "react-i18next"
import { IconEye, IconEyeOff, IconLoader2, IconCheck, IconX } from "@tabler/icons-react"

import { useAuth } from "@/features/auth"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"

export function SetupForm() {
  const { t } = useTranslation()
  const { setup } = useAuth()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)

  const isNotSecure = typeof window !== "undefined" && window.location.protocol === "http:" && window.location.hostname !== "localhost" && window.location.hostname !== "127.0.0.1"

  const passwordStrength = getPasswordStrength(password)
  const passwordsMatch = password === confirmPassword && confirmPassword !== ""
  const isValid = username.length >= 2 && password.length >= 6 && passwordsMatch

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")

    if (password !== confirmPassword) {
      setError(t("auth.passwordMismatch", "Passwords do not match"))
      return
    }

    if (password.length < 6) {
      setError(t("auth.passwordTooShort", "Password must be at least 6 characters"))
      return
    }

    if (username.length < 2) {
      setError(t("auth.usernameTooShort", "Username must be at least 2 characters"))
      return
    }

    setLoading(true)

    try {
      const result = await setup(username, password)
      if (!result.success) {
        setError(result.message || t("auth.setupFailed", "Setup failed"))
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
        <Label htmlFor="setup-username">{t("auth.username", "Username")}</Label>
        <Input
          id="setup-username"
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
          autoComplete="username"
          autoFocus
          placeholder={t("auth.usernamePlaceholder", "Enter your username")}
          disabled={loading}
        />
        {username.length > 0 && (
          <p className={`text-xs ${username.length >= 2 ? "text-green-600" : "text-muted-foreground"}`}>
            {username.length >= 2 ? (
              <IconCheck className="inline h-3 w-3 mr-1" />
            ) : (
              <IconX className="inline h-3 w-3 mr-1" />
            )}
            {t("auth.usernameRequirement", "At least 2 characters")}
          </p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="setup-password">{t("auth.password", "Password")}</Label>
        <div className="relative">
          <Input
            id="setup-password"
            type={showPassword ? "text" : "password"}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="new-password"
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
        {password.length > 0 && (
          <div className="space-y-1">
            <div className="flex gap-1">
              {[1, 2, 3, 4].map((level) => (
                <div
                  key={level}
                  className={`h-1 flex-1 rounded ${
                    passwordStrength >= level
                      ? passwordStrength <= 1
                        ? "bg-red-500"
                        : passwordStrength === 2
                          ? "bg-yellow-500"
                          : passwordStrength === 3
                            ? "bg-blue-500"
                            : "bg-green-500"
                      : "bg-gray-200"
                  }`}
                />
              ))}
            </div>
            <p className="text-xs text-muted-foreground">
              {password.length < 6 ? (
                <>
                  <IconX className="inline h-3 w-3 mr-1" />
                  {t("auth.passwordRequirement", "At least 6 characters")}
                </>
              ) : (
                <>
                  <IconCheck className="inline h-3 w-3 mr-1 text-green-600" />
                  {t("auth.passwordValid", "Password is valid")}
                </>
              )}
            </p>
          </div>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="confirm-password">{t("auth.confirmPassword", "Confirm Password")}</Label>
        <div className="relative">
          <Input
            id="confirm-password"
            type={showConfirmPassword ? "text" : "password"}
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            required
            autoComplete="new-password"
            placeholder={t("auth.confirmPasswordPlaceholder", "Confirm your password")}
            disabled={loading}
            className="pr-10"
          />
          <button
            type="button"
            onClick={() => setShowConfirmPassword(!showConfirmPassword)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            tabIndex={-1}
          >
            {showConfirmPassword ? <IconEyeOff className="h-4 w-4" /> : <IconEye className="h-4 w-4" />}
          </button>
        </div>
        {confirmPassword.length > 0 && (
          <p className={`text-xs ${passwordsMatch ? "text-green-600" : "text-red-500"}`}>
            {passwordsMatch ? (
              <IconCheck className="inline h-3 w-3 mr-1" />
            ) : (
              <IconX className="inline h-3 w-3 mr-1" />
            )}
            {passwordsMatch
              ? t("auth.passwordsMatch", "Passwords match")
              : t("auth.passwordsDoNotMatch", "Passwords do not match")}
          </p>
        )}
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Button type="submit" className="w-full" disabled={loading || !isValid}>
        {loading ? (
          <>
            <IconLoader2 className="mr-2 h-4 w-4 animate-spin" />
            {t("auth.settingUp", "Setting up...")}
          </>
        ) : (
          t("auth.setup", "Create Account")
        )}
      </Button>
    </form>
  )
}

function getPasswordStrength(password: string): number {
  if (!password) return 0
  let strength = 0
  if (password.length >= 6) strength++
  if (password.length >= 10) strength++
  if (/[A-Z]/.test(password) && /[a-z]/.test(password)) strength++
  if (/[0-9]/.test(password) && /[^A-Za-z0-9]/.test(password)) strength++
  return strength
}
