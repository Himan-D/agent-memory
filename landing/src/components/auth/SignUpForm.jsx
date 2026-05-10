import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Sparkles, Loader2, Eye, EyeOff } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useRouter } from 'react-router-dom'
import { trackEvent } from '../lib/amplitude'

function SignUpForm() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const { signUp } = useAuth()

  async function handleSubmit(e) {
    e.preventDefault()
    setIsLoading(true)
    setError("")

    const formData = new FormData(e.currentTarget)
    const name = formData.get("name")
    const email = formData.get("email")
    const password = formData.get("password")
    const confirmPassword = formData.get("confirmPassword")

    if (password !== confirmPassword) {
      setError("Passwords do not match")
      setIsLoading(false)
      return
    }

    if (password.length < 6) {
      setError("Password must be at least 6 characters")
      setIsLoading(false)
      return
    }

    trackEvent("sign_up_attempt", { email })

    const result = await signUp(name, email, password)

    if (result.success) {
      trackEvent("sign_up_success", { email })
      router.push("/dashboard")
    } else {
      trackEvent("sign_up_error", { email, error: result.error })
      setError(result.error)
      setIsLoading(false)
    }
  }

  return (
    <Card className="shadow-2xl">
      <CardHeader className="space-y-4">
        <CardTitle className="text-3xl font-bold tracking-tight">Create an account</CardTitle>
        <CardDescription className="text-base">
          Enter your details below to create your account
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-6">
        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-3">
            <Label htmlFor="name" className="text-base font-medium">Name</Label>
            <Input
              id="name"
              name="name"
              type="text"
              placeholder="Your name"
              required
              className="h-12 text-lg"
            />
          </div>
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
          <div className="space-y-3">
            <Label htmlFor="confirmPassword" className="text-base font-medium">Confirm Password</Label>
            <div className="relative">
              <Input
                id="confirmPassword"
                name="confirmPassword"
                type={showConfirmPassword ? "text" : "password"}
                placeholder="••••••••"
                required
                className="h-12 text-lg pr-12"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-0 top-0 h-full px-3"
                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
              >
                {showConfirmPassword ? (
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
                Creating account...
              </>
            ) : (
              <>
                <Sparkles className="mr-2 h-4 w-4" />
                Create Account
              </>
            )}
          </Button>
        </form>
        <div className="text-center text-sm mt-4">
          Already have an account?{" "}
          <Button
            variant="link"
            className="p-0 h-auto font-semibold"
            onClick={() => router.push("/auth/signin")}
          >
            Sign in
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export default SignUpForm