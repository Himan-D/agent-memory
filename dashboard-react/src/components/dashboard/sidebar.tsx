import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";
import { useAuth } from "@/contexts/auth-context";
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
  ScrollText,
  ChevronLeft,
  ChevronRight,
  User as UserIcon,
  CreditCard,
} from "lucide-react";

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
      { href: "/audit", label: "Audit Trail", icon: ScrollText },
      { href: "/users", label: "Team", icon: Shield },
      { href: "/notifications", label: "Notifications", icon: Bell },
      { href: "/billing", label: "Billing", icon: CreditCard },
      { href: "/settings", label: "Settings", icon: Settings },
    ],
  },
];

export function Sidebar() {
  const { pathname } = useLocation();
  const [collapsed, setCollapsed] = useState(false);
  const { user } = useAuth();

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
            <Link to="/" className="flex items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
                 <span className="text-primary-foreground font-bold text-xs">H</span>
              </div>
              <span className="font-bold text-xl">Hystersis</span>
            </Link>
          )}
          {collapsed && (
            <Link to="/" className="flex items-center justify-center w-full">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
                 <span className="text-primary-foreground font-bold text-xs">H</span>
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
                const isActive = item.href === "/" ? pathname === "/" : pathname === item.href || pathname.startsWith(item.href + "/");
                return (
                  <Link
                    key={item.href}
                    to={item.href}
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

        {/* Minimal Footer */}
        <div className="border-t border-border p-4">
          <div className="w-full rounded-lg border bg-card p-3 flex items-center gap-3 overflow-hidden">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
              {user ? (
                <span className="text-xs font-medium">{user.name.slice(0, 2).toUpperCase()}</span>
              ) : (
                <UserIcon className="h-4 w-4" />
              )}
            </div>
            {!collapsed && user && (
              <div className="flex flex-col min-w-0 overflow-hidden">
                <span className="truncate text-sm font-medium">{user.name}</span>
                <span className="truncate text-xs text-muted-foreground">{user.email}</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </aside>
  );
}
