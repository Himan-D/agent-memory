"use client";

import { useEffect, useState } from "react";
import { Building2, Check, ChevronsUpDown, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  getActiveTenant,
  setActiveTenant,
  tenantsApi,
  type Tenant,
} from "@/lib/api";
import { cn } from "@/lib/utils";

/**
 * Tenant switcher for multi-tenant dashboard.
 * Lists orgs the user can access; switching calls /session/tenant and
 * persists active tenant for X-Tenant-ID on subsequent API calls.
 */
export function TenantSwitcher() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [switching, setSwitching] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await tenantsApi.list();
        if (cancelled) return;
        const list = res.tenants || [];
        setTenants(list);
        const stored = getActiveTenant();
        const initial =
          (stored && list.find((t) => t.id === stored)?.id) ||
          list[0]?.id ||
          null;
        if (initial) {
          setActiveId(initial);
          setActiveTenant(initial);
        }
      } catch (e) {
        if (!cancelled) {
          setError("Unable to load tenants");
          // Non-fatal: single-tenant deploys may not expose /tenants yet
          console.warn("tenant list failed", e);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const active = tenants.find((t) => t.id === activeId);

  async function onSelect(tenantId: string) {
    if (tenantId === activeId || switching) return;
    setSwitching(true);
    setError(null);
    try {
      await tenantsApi.switch(tenantId);
      setActiveTenant(tenantId);
      setActiveId(tenantId);
      // Reload so all pages refetch under the new tenant scope
      if (typeof window !== "undefined") {
        window.location.reload();
      }
    } catch (e) {
      console.error("switch tenant failed", e);
      // Still set client-side for API-key admin override path
      setActiveTenant(tenantId);
      setActiveId(tenantId);
      setError("Switch may require session membership");
    } finally {
      setSwitching(false);
    }
  }

  if (loading) {
    return (
      <Button variant="outline" size="sm" disabled className="gap-2 min-w-[140px]">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        <span className="text-xs">Tenant…</span>
      </Button>
    );
  }

  if (tenants.length === 0) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className="gap-2 min-w-[140px] max-w-[220px] justify-between"
          disabled={switching}
          aria-label="Switch organization"
        >
          <span className="flex items-center gap-2 truncate">
            <Building2 className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate text-xs font-medium">
              {active?.name || active?.slug || activeId || "Organization"}
            </span>
          </span>
          {switching ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin shrink-0" />
          ) : (
            <ChevronsUpDown className="h-3.5 w-3.5 shrink-0 opacity-50" />
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="text-xs">Organizations</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {tenants.map((t) => (
          <DropdownMenuItem
            key={t.id}
            onClick={() => onSelect(t.id)}
            className="flex items-center justify-between gap-2"
          >
            <span className="truncate">
              <span className="font-medium">{t.name || t.slug}</span>
              {t.plan ? (
                <span className="ml-2 text-xs text-muted-foreground">{t.plan}</span>
              ) : null}
            </span>
            <Check
              className={cn(
                "h-3.5 w-3.5 shrink-0",
                t.id === activeId ? "opacity-100" : "opacity-0"
              )}
            />
          </DropdownMenuItem>
        ))}
        {error ? (
          <>
            <DropdownMenuSeparator />
            <div className="px-2 py-1.5 text-xs text-destructive">{error}</div>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
