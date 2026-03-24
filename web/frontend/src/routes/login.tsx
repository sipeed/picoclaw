import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { useEffect } from "react"

import { useAuth } from "@/features/auth"
import { LoginForm, SetupForm } from "@/components/auth"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

function LoginPage() {
  const { t } = useTranslation()
  const { needsSetup, needsLogin, isAuthenticated } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (isAuthenticated) {
      navigate({ to: "/" })
    }
  }, [isAuthenticated, navigate])

  useEffect(() => {
    if (!needsSetup && !needsLogin) {
      navigate({ to: "/" })
    }
  }, [needsSetup, needsLogin, navigate])

  if (isAuthenticated || (!needsSetup && !needsLogin)) {
    return null
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">PicoClaw</CardTitle>
          <CardDescription>
            {needsSetup
              ? t("auth.setupDescription", "Create an admin account to secure your instance")
              : t("auth.loginDescription", "Enter your credentials to access the console")}
          </CardDescription>
        </CardHeader>
        <CardContent>{needsSetup ? <SetupForm /> : <LoginForm />}</CardContent>
      </Card>
    </div>
  )
}

export const Route = createFileRoute("/login")({
  component: LoginPage,
})
