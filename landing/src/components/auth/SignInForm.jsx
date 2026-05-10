import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Sparkles, Loader2, Eye, EyeOff } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useRouter } from 'react-router-dom'
import { trackEvent } from '../lib/amplitude'

function SignInForm() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const { signIn } = useAuth()

  async function handleSubmit(e) {
    e.preventDefault()
    setIsLoading(true)
    setError("")

    const formData = new FormData(e.currentTarget)
    const email = formData.get("email")
    const password = formData.get("password")

    trackEvent("sign_in_attempt", { email })

    const result = await signIn(email, password)

    if (result.success) {
      trackEvent("sign_in_success", { email })
      router.push("/dashboard")
    } else {
      trackEvent("sign_in_error", { email, error: result.error })
      setError(result.error)
      setIsLoading(false)
    }
  }

  return (
    <Card className="shadow-2xl">
      <CardHeader className="space-y-4">
        <CardTitle className="text-3xl font-bold tracking-tight">Sign in</CardTitle>
        <CardDescription className="text-base">
          Welcome back! Enter your email below to sign in to your account
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-6">
        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-3">
            <Label htmlFor="email" className="text-base font-medium">Email</Label>
            <Input
              id="email"
              name="email"
              type="email"
              placeholder="you@example.com"
              required
              className="h-12 text-lg"
            />
          </div>
          <div className="space-y-3">
            <Label htmlFor="password" className="text-base font-medium">Password</Label>
            <div className="relative">
              <Input
                id="password"
                name="password"
                type={showPassword ? "text" : "password"}
                placeholder="••••••••"
                required
                className="h-12 text-lg pr-12"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-0 top-0 h-full px-3"
                onClick={() => setShowPassword(!showPassword)}
              >
                {showPassword ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
          {error && (
            <div className="rounded-md bg-destructive/15 p-4 text-sm text-destructive">
              {error}
            </div>
          )}
          <Button type="submit" className="w-full h-12 text-lg" disabled={isLoading}>
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                Signing in...
              </>
            ) : (
              <>
                <Sparkles className="mr-2 h-4 w-4" />
                Sign in
              </>
            )}
          </Button>
        </form>
        <div className="text-center text-sm mt-4">
          Don&apos;t have an account?{" "}
          <Button
            variant="link"
            className="p-0 h-auto font-semibold"
            onClick={() => router.push("/auth/signup")}
          >
            Sign up
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export default SignInForm