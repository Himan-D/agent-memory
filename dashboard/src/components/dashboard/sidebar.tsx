"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession } from "next-auth/react";
import { cn } from "@/lib/utils";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  LayoutDashboard,
  Database,
  CircleDot,
  Bot,
  Users,
  Sparkles,
  Key,
  BarChart3,
  FolderKanban,
  Webhook,
  Settings,
  ChevronLeft,
  ChevronRight,
  Workflow,
  Bell,
  Shield,
  AlertTriangle,
} from "lucide-react";
import { useState } from "react";

const sidebarItems = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/memories", label: "Memories", icon: Database },
  { href: "/entities", label: "Entities", icon: CircleDot },
  { href: "/sessions", label: "Sessions", icon: Bot },
  { href: "/agents", label: "Agents", icon: Bot },
  { href: "/groups", label: "Groups", icon: Users },
  { href: "/projects", label: "Projects", icon: FolderKanban },
  { href: "/skills", label: "Skills", icon: Sparkles },
  { href: "/chains", label: "Chains", icon: Workflow },
  { href: "/webhooks", label: "Webhooks", icon: Webhook },
  { href: "/api-keys", label: "API Keys", icon: Key },
  { href: "/alerts", label: "Alerts", icon: AlertTriangle },
  { href: "/users", label: "Team", icon: Shield },
  { href: "/analytics", label: "Analytics", icon: BarChart3 },
  { href: "/notifications", label: "Notifications", icon: Bell },
  { href: "/settings", label: "Settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const { data: session } = useSession();

  const userInitials = session?.user?.name
    ? session.user.name.split(" ").map(n => n[0]).join("").toUpperCase().slice(0, 2)
    : session?.user?.email?.[0]?.toUpperCase() || "U";

  return (
    <aside
      className={cn(
        "fixed left-0 top-0 z-40 h-screen border-r border-border bg-sidebar transition-all duration-300",
        collapsed ? "w-16" : "w-64"
      )}
    >
      <div className="flex h-full flex-col">
        {/* Logo */}
        <div className="flex h-16 items-center justify-between border-b border-border px-4">
          {!collapsed && (
            <Link href="/" className="flex items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Sparkles className="h-4 w-4" />
              </div>
              <span className="font-bold text-xl">Hystersis</span>
            </Link>
          )}
          {collapsed && (
            <Link href="/" className="flex items-center justify-center w-full">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <Sparkles className="h-4 w-4" />
              </div>
            </Link>
          )}
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-accent"
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 space-y-1 p-2">
          {sidebarItems.map((item) => {
            const isActive = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-accent hover:text-foreground",
                  collapsed && "justify-center px-2"
                )}
                title={collapsed ? item.label : undefined}
              >
                <item.icon className="h-5 w-5 flex-shrink-0" />
                {!collapsed && <span>{item.label}</span>}
              </Link>
            );
          })}
        </nav>

        {/* Footer */}
        {!collapsed && (
          <div className="border-t border-border p-4">
            <div className="rounded-lg border bg-card p-3">
              <div className="flex items-center gap-3">
                <Avatar className="h-9 w-9">
                  <AvatarImage src={session?.user?.image || undefined} alt={session?.user?.name || "User"} />
                  <AvatarFallback className="text-xs">{userInitials}</AvatarFallback>
                </Avatar>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">
                    {session?.user?.name || "User"}
                  </p>
                  <p className="text-xs text-muted-foreground truncate">
                    {session?.user?.email || "user@example.com"}
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Collapsed footer */}
        {collapsed && (
          <div className="border-t border-border p-2">
            <Avatar className="h-8 w-8 mx-auto">
              <AvatarImage src={session?.user?.image || undefined} alt={session?.user?.name || "User"} />
              <AvatarFallback className="text-xs">{userInitials}</AvatarFallback>
            </Avatar>
          </div>
        )}
      </div>
    </aside>
  );
}
