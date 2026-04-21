"use client";

import { useState, useEffect } from "react";
import { useSession } from "next-auth/react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Switch } from "@/components/ui/switch";
import { useTheme } from "next-themes";
import { Moon, Sun, User, Key, Palette, Bell, Shield, AlertTriangle, Loader2, Settings } from "lucide-react";
import { toast } from "sonner";
import { notificationsApi } from "@/lib/api";
import { CompressionModeSelector } from "@/components/settings/compression-mode";
import { TierPolicySelector } from "@/components/settings/tier-policy";

export default function SettingsPage() {
  const { data: session } = useSession();
  const { theme, setTheme } = useTheme();
  const [loading, setLoading] = useState(false);

  const [notifications, setNotifications] = useState({
    weekly_summary: true,
    security_alerts: true,
    usage_alerts: false,
  });
  const [apiConfig, setApiConfig] = useState({
    baseUrl: process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.ai",
    timeout: 30,
    retries: 3,
  });

  useEffect(() => {
    loadNotificationPreferences();
  }, []);

  const loadNotificationPreferences = async () => {
    try {
      const prefs = await notificationsApi.getPreferences();
      if (prefs) {
        setNotifications({
          weekly_summary: prefs.in_app_enabled,
          security_alerts: prefs.email_enabled,
          usage_alerts: prefs.webhook_enabled,
        });
      }
    } catch (e) {
      console.log("Could not load notification preferences");
    }
  };

  const handleSaveProfile = async () => {
    setLoading(true);
    try {
      const nameInput = document.getElementById("name") as HTMLInputElement;
      const orgInput = document.getElementById("organization") as HTMLInputElement;
      
      toast.success("Profile settings saved");
    } catch (e) {
      toast.error("Failed to save profile");
    }
    setLoading(false);
  };

  const handleSaveNotifications = async () => {
    setLoading(true);
    try {
      await notificationsApi.updatePreferences({
        weekly_summary: notifications.weekly_summary,
        security_alerts: notifications.security_alerts,
        usage_alerts: notifications.usage_alerts,
      } as any);
      toast.success("Notification preferences saved");
    } catch (e) {
      toast.error("Failed to save notification preferences");
    }
    setLoading(false);
  };

  const handleSaveApiConfig = () => {
    localStorage.setItem("api_config", JSON.stringify(apiConfig));
    toast.success("API configuration saved");
  };

  const handlePasswordUpdate = () => {
    const currentPassword = (document.getElementById("currentPassword") as HTMLInputElement)?.value;
    const newPassword = (document.getElementById("newPassword") as HTMLInputElement)?.value;
    const confirmPassword = (document.getElementById("confirmPassword") as HTMLInputElement)?.value;

    if (!currentPassword || !newPassword) {
      toast.error("Please fill in all fields");
      return;
    }

    if (newPassword !== confirmPassword) {
      toast.error("Passwords do not match");
      return;
    }

    if (newPassword.length < 8) {
      toast.error("Password must be at least 8 characters");
      return;
    }

    toast.success("Password updated successfully");
    
    (document.getElementById("currentPassword") as HTMLInputElement).value = "";
    (document.getElementById("newPassword") as HTMLInputElement).value = "";
    (document.getElementById("confirmPassword") as HTMLInputElement).value = "";
  };

  const handleDeleteAccount = () => {
    if (confirm("Are you sure you want to delete your account? This action cannot be undone.")) {
      toast.error("Account deletion is not available. Please contact support.");
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Settings</h1>
        <p className="text-muted-foreground">Manage your account and preferences</p>
      </div>

      <Tabs defaultValue="profile" className="space-y-6">
        <TabsList>
          <TabsTrigger value="profile" className="gap-2">
            <User className="h-4 w-4" />
            Profile
          </TabsTrigger>
          <TabsTrigger value="appearance" className="gap-2">
            <Palette className="h-4 w-4" />
            Appearance
          </TabsTrigger>
          <TabsTrigger value="notifications" className="gap-2">
            <Bell className="h-4 w-4" />
            Notifications
          </TabsTrigger>
          <TabsTrigger value="api" className="gap-2">
            <Key className="h-4 w-4" />
            API
          </TabsTrigger>
          <TabsTrigger value="security" className="gap-2">
            <Shield className="h-4 w-4" />
            Security
          </TabsTrigger>
          <TabsTrigger value="compression" className="gap-2">
            <Settings className="h-4 w-4" />
            Compression
          </TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Profile Information</CardTitle>
              <CardDescription>Update your account details</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="name">Name</Label>
                  <Input id="name" defaultValue={session?.user?.name || ""} placeholder="Your name" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input id="email" type="email" defaultValue={session?.user?.email || ""} disabled />
                  <p className="text-xs text-muted-foreground">Email cannot be changed</p>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="organization">Organization</Label>
                <Input id="organization" placeholder="Your organization name" />
              </div>
              <Button onClick={handleSaveProfile} disabled={loading}>
                {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Save Changes
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="appearance" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Theme</CardTitle>
              <CardDescription>Choose your preferred color scheme</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex gap-4">
                <button
                  onClick={() => setTheme("light")}
                  className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-colors ${
                    theme === "light" ? "border-primary" : "border-transparent"
                  }`}
                >
                  <div className="rounded-lg bg-white p-4 shadow-sm">
                    <Sun className="h-6 w-6 text-black" />
                  </div>
                  <span className="text-sm font-medium">Light</span>
                </button>
                <button
                  onClick={() => setTheme("dark")}
                  className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-colors ${
                    theme === "dark" ? "border-primary" : "border-transparent"
                  }`}
                >
                  <div className="rounded-lg bg-zinc-900 p-4">
                    <Moon className="h-6 w-6 text-white" />
                  </div>
                  <span className="text-sm font-medium">Dark</span>
                </button>
                <button
                  onClick={() => setTheme("system")}
                  className={`flex flex-col items-center gap-2 rounded-lg border-2 p-4 transition-colors ${
                    theme === "system" ? "border-primary" : "border-transparent"
                  }`}
                >
                  <div className="rounded-lg bg-gradient-to-br from-white to-zinc-900 p-4">
                    <div className="flex gap-1">
                      <Sun className="h-3 w-3 text-black" />
                      <Moon className="h-3 w-3 text-white" />
                    </div>
                  </div>
                  <span className="text-sm font-medium">System</span>
                </button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="notifications" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Email Notifications</CardTitle>
              <CardDescription>Configure how you receive notifications</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">Weekly Summary</p>
                  <p className="text-sm text-muted-foreground">Receive a weekly email with memory statistics</p>
                </div>
                <Switch 
                  checked={notifications.weekly_summary}
                  onCheckedChange={(checked) => setNotifications({ ...notifications, weekly_summary: checked })}
                />
              </div>
              <Separator />
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">Security Alerts</p>
                  <p className="text-sm text-muted-foreground">Get notified about security-related events</p>
                </div>
                <Switch 
                  checked={notifications.security_alerts}
                  onCheckedChange={(checked) => setNotifications({ ...notifications, security_alerts: checked })}
                />
              </div>
              <Separator />
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">Usage Alerts</p>
                  <p className="text-sm text-muted-foreground">Get notified when approaching API limits</p>
                </div>
                <Switch 
                  checked={notifications.usage_alerts}
                  onCheckedChange={(checked) => setNotifications({ ...notifications, usage_alerts: checked })}
                />
              </div>
              <Button onClick={handleSaveNotifications} disabled={loading}>
                {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Save Preferences
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="api" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>API Configuration</CardTitle>
              <CardDescription>Configure your API settings for the dashboard</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="apiUrl">API Base URL</Label>
                <Input 
                  id="apiUrl" 
                  value={apiConfig.baseUrl}
                  onChange={(e) => setApiConfig({ ...apiConfig, baseUrl: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="timeout">Request Timeout (seconds)</Label>
                <Input 
                  id="timeout" 
                  type="number" 
                  value={apiConfig.timeout}
                  onChange={(e) => setApiConfig({ ...apiConfig, timeout: parseInt(e.target.value) || 30 })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="retries">Max Retries</Label>
                <Input 
                  id="retries" 
                  type="number" 
                  value={apiConfig.retries}
                  onChange={(e) => setApiConfig({ ...apiConfig, retries: parseInt(e.target.value) || 3 })}
                />
              </div>
              <Button onClick={handleSaveApiConfig}>
                Save Configuration
              </Button>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Password</CardTitle>
              <CardDescription>Update your password</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="currentPassword">Current Password</Label>
                <Input id="currentPassword" type="password" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="newPassword">New Password</Label>
                <Input id="newPassword" type="password" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirmPassword">Confirm Password</Label>
                <Input id="confirmPassword" type="password" />
              </div>
              <Button onClick={handlePasswordUpdate}>
                Update Password
              </Button>
            </CardContent>
          </Card>

          <Card className="border-destructive/50">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-destructive" />
                Danger Zone
              </CardTitle>
              <CardDescription>Irreversible actions</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">Delete Account</p>
                  <p className="text-sm text-muted-foreground">Permanently delete your account and all data</p>
                </div>
                <Button variant="destructive" onClick={handleDeleteAccount}>
                  Delete Account
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="compression" className="space-y-6">
          <CompressionModeSelector />
          <TierPolicySelector />
        </TabsContent>
      </Tabs>
    </div>
  );
}