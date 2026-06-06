"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useSession, signOut } from "next-auth/react";
import { cn } from "@/lib/utils";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  LayoutDashboard,
  Database,
  CircleDot,
  MessageSquare,
  Search,
  FileText,
  FlaskConical,
  Bot,
  Users,
  Sparkles,
  Workflow,
  FolderKanban,
  Webhook,
  Key,
  AlertTriangle,
  BarChart3,
  Shield,
  Bell,
  Settings,
  ChevronLeft,
  ChevronRight,
  LogOut,
  User,
  CreditCard,
} from "lucide-react";
import { useState } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface SidebarItem {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}

interface SidebarGroup {
  label: string;
  items: SidebarItem[];
}

const sidebarGroups: SidebarGroup[] = [
  {
    label: "Core",
    items: [
      { href: "/", label: "Dashboard", icon: LayoutDashboard },
      { href: "/memories", label: "Memories", icon: Database },
      { href: "/entities", label: "Entities", icon: CircleDot },
      { href: "/sessions", label: "Sessions", icon: MessageSquare },
    ],
  },
  {
    label: "Intelligence",
    items: [
      { href: "/search", label: "Search", icon: Search },
      { href: "/documents", label: "Documents", icon: FileText },
      { href: "/playground", label: "Playground", icon: FlaskConical },
    ],
  },
  {
    label: "Agent Infra",
    items: [
      { href: "/agents", label: "Agents", icon: Bot },
      { href: "/groups", label: "Groups", icon: Users },
      { href: "/skills", label: "Skills", icon: Sparkles },
      { href: "/chains", label: "Chains", icon: Workflow },
    ],
  },
  {
    label: "Integration",
    items: [
      { href: "/projects", label: "Projects", icon: FolderKanban },
      { href: "/webhooks", label: "Webhooks", icon: Webhook },
      { href: "/api-keys", label: "API Keys", icon: Key },
    ],
  },
  {
    label: "Operations",
    items: [
      { href: "/alerts", label: "Alerts", icon: AlertTriangle },
      { href: "/analytics", label: "Analytics", icon: BarChart3 },
      { href: "/users", label: "Team", icon: Shield },
      { href: "/notifications", label: "Notifications", icon: Bell },
      { href: "/billing", label: "Billing", icon: CreditCard },
      { href: "/settings", label: "Settings", icon: Settings },
    ],
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const { data: session } = useSession();

  const userInitials = session?.user?.name
    ? session.user.name.split(" ").map((n: string) => n[0]).join("").toUpperCase().slice(0, 2)
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
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
                <svg viewBox="0 0 128 128" className="h-5 w-5" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <defs>
                    <linearGradient id="logoGrad" x1="0%" y1="0%" x2="100%" y2="100%">
                      <stop offset="0%" stopColor="#6366f1"/>
                      <stop offset="100%" stopColor="#8b5cf6"/>
                    </linearGradient>
                  </defs>
                  <circle cx="64" cy="64" r="12" fill="url(#logoGrad)"/>
                  <circle cx="32" cy="40" r="7" fill="url(#logoGrad)" opacity="0.9"/>
                  <circle cx="96" cy="40" r="7" fill="url(#logoGrad)" opacity="0.9"/>
                  <circle cx="32" cy="88" r="7" fill="url(#logoGrad)" opacity="0.9"/>
                  <circle cx="96" cy="88" r="7" fill="url(#logoGrad)" opacity="0.9"/>
                  <circle cx="64" cy="24" r="5" fill="url(#logoGrad)" opacity="0.7"/>
                  <circle cx="64" cy="104" r="5" fill="url(#logoGrad)" opacity="0.7"/>
                  <line x1="64" y1="52" x2="32" y2="40" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
                  <line x1="64" y1="52" x2="96" y2="40" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
                  <line x1="64" y1="52" x2="32" y2="88" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
                  <line x1="64" y1="52" x2="96" y2="88" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
                  <line x1="64" y1="52" x2="64" y2="24" stroke="#6366f1" strokeWidth="2" opacity="0.4"/>
                  <line x1="64" y1="76" x2="64" y2="104" stroke="#6366f1" strokeWidth="2" opacity="0.4"/>
                </svg>
              </div>
              <span className="font-bold text-xl">Hystersis</span>
            </Link>
          )}
          {collapsed && (
            <Link href="/" className="flex items-center justify-center w-full">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
                <svg viewBox="0 0 128 128" className="h-5 w-5" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <defs>
                    <linearGradient id="logoGradCollapsed" x1="0%" y1="0%" x2="100%" y2="100%">
                      <stop offset="0%" stopColor="#6366f1"/>
                      <stop offset="100%" stopColor="#8b5cf6"/>
                    </linearGradient>
                  </defs>
                  <circle cx="64" cy="64" r="12" fill="url(#logoGradCollapsed)"/>
                  <circle cx="32" cy="40" r="7" fill="url(#logoGradCollapsed)" opacity="0.9"/>
                  <circle cx="96" cy="40" r="7" fill="url(#logoGradCollapsed)" opacity="0.9"/>
                  <circle cx="32" cy="88" r="7" fill="url(#logoGradCollapsed)" opacity="0.9"/>
                  <circle cx="96" cy="88" r="7" fill="url(#logoGradCollapsed)" opacity="0.9"/>
                  <circle cx="64" cy="24" r="5" fill="url(#logoGradCollapsed)" opacity="0.7"/>
                  <circle cx="64" cy="104" r="5" fill="url(#logoGradCollapsed)" opacity="0.7"/>
                  <line x1="64" y1="52" x2="32" y2="40" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
                  <line x1="64" y1="52" x2="96" y2="40" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
                  <line x1="64" y1="52" x2="32" y2="88" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
                  <line x1="64" y1="52" x2="96" y2="88" stroke="#6366f1" strokeWidth="2.5" opacity="0.6"/>
                  <line x1="64" y1="52" x2="64" y2="24" stroke="#6366f1" strokeWidth="2" opacity="0.4"/>
                  <line x1="64" y1="76" x2="64" y2="104" stroke="#6366f1" strokeWidth="2" opacity="0.4"/>
                </svg>
              </div>
            </Link>
          )}
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="flex h-8 w-8 items-center justify-center rounded-md hover:bg-accent"
            aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto p-2">
          {sidebarGroups.map((group) => (
            <div key={group.label} className="mb-1">
              {!collapsed && (
                <div className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">
                  {group.label}
                </div>
              )}
              {group.items.map((item) => {
                const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
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
            </div>
          ))}
        </nav>

        {/* Footer with user profile dropdown */}
        {!collapsed && (
          <div className="border-t border-border p-4">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button className="w-full rounded-lg border bg-card p-3 hover:bg-accent transition-colors cursor-pointer">
                  <div className="flex items-center gap-3">
                    <Avatar className="h-9 w-9">
                      <AvatarImage src={session?.user?.image || undefined} alt={session?.user?.name || "User"} />
                      <AvatarFallback className="text-xs">{userInitials}</AvatarFallback>
                    </Avatar>
                    <div className="flex-1 min-w-0 text-left">
                      <p className="text-sm font-medium truncate">
                        {session?.user?.name || "User"}
                      </p>
                      <p className="text-xs text-muted-foreground truncate">
                        {session?.user?.email || "user@example.com"}
                      </p>
                    </div>
                  </div>
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuLabel>My Account</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem asChild>
                  <Link href="/settings" className="flex items-center cursor-pointer">
                    <User className="mr-2 h-4 w-4" />
                    <span>Profile Settings</span>
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link href="/settings" className="flex items-center cursor-pointer">
                    <CreditCard className="mr-2 h-4 w-4" />
                    <span>Billing & Plans</span>
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => signOut({ callbackUrl: "/auth/signin" })}
                  className="text-destructive focus:text-destructive cursor-pointer"
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  <span>Sign out</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}

        {/* Collapsed footer with dropdown */}
        {collapsed && (
          <div className="border-t border-border p-2">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button className="w-full cursor-pointer rounded-md p-1 hover:bg-accent transition-colors">
                  <Avatar className="h-8 w-8 mx-auto">
                    <AvatarImage src={session?.user?.image || undefined} alt={session?.user?.name || "User"} />
                    <AvatarFallback className="text-xs">{userInitials}</AvatarFallback>
                  </Avatar>
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuLabel>{session?.user?.name || "User"}</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem asChild>
                  <Link href="/settings" className="flex items-center cursor-pointer">
                    <User className="mr-2 h-4 w-4" />
                    <span>Profile Settings</span>
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link href="/settings" className="flex items-center cursor-pointer">
                    <CreditCard className="mr-2 h-4 w-4" />
                    <span>Billing & Plans</span>
                  </Link>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => signOut({ callbackUrl: "/auth/signin" })}
                  className="text-destructive focus:text-destructive cursor-pointer"
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  <span>Sign out</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}
      </div>
    </aside>
  );
}