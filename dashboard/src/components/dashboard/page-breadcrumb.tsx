"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Fragment } from "react";

const LABELS: Record<string, string> = {
  "": "Dashboard",
  memories: "Memories",
  entities: "Entities",
  sessions: "Sessions",
  search: "Search",
  documents: "Documents",
  playground: "Playground",
  agents: "Agents",
  groups: "Groups",
  skills: "Skills",
  chains: "Chains",
  projects: "Projects",
  webhooks: "Webhooks",
  "api-keys": "API Keys",
  alerts: "Alerts",
  analytics: "Analytics",
  audit: "Audit Trail",
  users: "Team",
  notifications: "Notifications",
  billing: "Billing",
  settings: "Settings",
};

export function PageBreadcrumb() {
  const pathname = usePathname() || "/";
  const segments = pathname.split("/").filter(Boolean);

  if (segments.length === 0) {
    return (
      <Breadcrumb className="mb-4">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbPage>Dashboard</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
    );
  }

  return (
    <Breadcrumb className="mb-4">
      <BreadcrumbList>
        <BreadcrumbItem>
          <Link
            href="/"
            className="transition-colors hover:text-foreground text-muted-foreground"
          >
            Dashboard
          </Link>
        </BreadcrumbItem>
        {segments.map((seg, i) => {
          const href = "/" + segments.slice(0, i + 1).join("/");
          const label =
            LABELS[seg] ||
            seg.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
          const isLast = i === segments.length - 1;
          return (
            <Fragment key={href}>
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {isLast ? (
                  <BreadcrumbPage>{label}</BreadcrumbPage>
                ) : (
                  <Link
                    href={href}
                    className="transition-colors hover:text-foreground text-muted-foreground"
                  >
                    {label}
                  </Link>
                )}
              </BreadcrumbItem>
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
