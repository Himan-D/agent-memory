"use client";

import { useCallback, useEffect, useState } from "react";
import { Building2, Loader2, UserPlus, Trash2, Copy, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  getActiveTenant,
  setActiveTenant,
  tenantsApi,
  type Tenant,
  type TenantMember,
} from "@/lib/api";
export default function OrganizationSettingsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [active, setActive] = useState<Tenant | null>(null);
  const [members, setMembers] = useState<TenantMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState("");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("member");
  const [inviteToken, setInviteToken] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { tenants: list } = await tenantsApi.list();
      setTenants(list || []);
      const stored = getActiveTenant();
      const current =
        list.find((t) => t.id === stored) || list[0] || null;
      setActive(current);
      if (current) {
        setActiveTenant(current.id);
        setName(current.name || "");
        const mem = await tenantsApi.listMembers(current.id);
        setMembers(mem.members || []);
      }
    } catch (e) {
      console.error(e);
      setError("Failed to load organization");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function onSave() {
    if (!active) return;
    setSaving(true);
    setMessage(null);
    setError(null);
    try {
      const updated = await tenantsApi.update(active.id, { name });
      setActive(updated);
      setMessage("Organization updated");
      await load();
    } catch (e) {
      setError("Update failed");
    } finally {
      setSaving(false);
    }
  }

  async function onInvite() {
    if (!active || !inviteEmail.trim()) return;
    setSaving(true);
    setError(null);
    setInviteToken(null);
    try {
      const inv = await tenantsApi.invite(active.id, {
        email: inviteEmail.trim(),
        role: inviteRole,
      });
      setInviteToken(inv.token);
      setInviteEmail("");
      setMessage(`Invite created for ${inv.email}`);
    } catch (e) {
      setError("Invite failed");
    } finally {
      setSaving(false);
    }
  }

  async function onRemove(userId: string) {
    if (!active) return;
    if (!confirm("Remove this member?")) return;
    try {
      await tenantsApi.removeMember(active.id, userId);
      setMembers((m) => m.filter((x) => x.user_id !== userId));
    } catch {
      setError("Could not remove member");
    }
  }

  async function onSwitch(id: string) {
    try {
      await tenantsApi.switch(id);
      setActiveTenant(id);
      window.location.reload();
    } catch {
      setActiveTenant(id);
      window.location.reload();
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12 gap-2 text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" />
        Loading organization…
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6 max-w-3xl">
      <div>
        <p className="text-xs text-muted-foreground mb-1">
          <a href="/settings" className="hover:underline">
            Settings
          </a>{" "}
          / Organization
        </p>
        <h1 className="text-2xl font-semibold tracking-tight flex items-center gap-2">
          <Building2 className="h-6 w-6" />
          Organization
        </h1>
        <p className="text-sm text-muted-foreground mt-1">
          Manage your multi-tenant workspace, members, and invites.
        </p>
      </div>

      {error ? (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      ) : null}
      {message ? (
        <div className="rounded-md border border-green-500/30 bg-green-500/10 px-3 py-2 text-sm text-green-700 dark:text-green-400">
          {message}
        </div>
      ) : null}

      {tenants.length > 1 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Switch organization</CardTitle>
            <CardDescription>You belong to multiple workspaces.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            {tenants.map((t) => (
              <Button
                key={t.id}
                size="sm"
                variant={t.id === active?.id ? "default" : "outline"}
                onClick={() => onSwitch(t.id)}
              >
                {t.name || t.slug}
              </Button>
            ))}
          </CardContent>
        </Card>
      ) : null}

      {active ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Details</CardTitle>
              <CardDescription>
                ID: <code className="text-xs">{active.id}</code>
                {active.plan ? ` · Plan: ${active.plan}` : null}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="org-name">Name</Label>
                <Input
                  id="org-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>Slug</Label>
                <Input value={active.slug || ""} disabled />
              </div>
              <Button onClick={onSave} disabled={saving}>
                {saving ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
                Save changes
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Members</CardTitle>
              <CardDescription>{members.length} member(s)</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {members.length === 0 ? (
                <p className="text-sm text-muted-foreground">No members listed.</p>
              ) : (
                <ul className="divide-y rounded-md border">
                  {members.map((m) => (
                    <li
                      key={m.user_id}
                      className="flex items-center justify-between gap-3 px-3 py-2 text-sm"
                    >
                      <div>
                        <div className="font-medium">{m.email || m.user_id}</div>
                        <div className="text-xs text-muted-foreground">{m.role}</div>
                      </div>
                      {m.role !== "owner" ? (
                        <Button
                          size="icon"
                          variant="ghost"
                          aria-label="Remove member"
                          onClick={() => onRemove(m.user_id)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      ) : null}
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Invite member</CardTitle>
              <CardDescription>
                Creates an invite token the invitee accepts via API or login flow.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex flex-col sm:flex-row gap-2">
                <Input
                  type="email"
                  placeholder="email@company.com"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                />
                <Select value={inviteRole} onValueChange={setInviteRole}>
                  <SelectTrigger className="w-[140px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="member">Member</SelectItem>
                    <SelectItem value="admin">Admin</SelectItem>
                    <SelectItem value="viewer">Viewer</SelectItem>
                  </SelectContent>
                </Select>
                <Button onClick={onInvite} disabled={saving || !inviteEmail.trim()}>
                  <UserPlus className="h-4 w-4 mr-2" />
                  Invite
                </Button>
              </div>
              {inviteToken ? (
                <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-2 text-xs font-mono break-all">
                  <span className="flex-1">token: {inviteToken}</span>
                  <Button
                    size="icon"
                    variant="ghost"
                    onClick={async () => {
                      await navigator.clipboard.writeText(inviteToken);
                      setCopied(true);
                      setTimeout(() => setCopied(false), 1500);
                    }}
                  >
                    {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                  </Button>
                </div>
              ) : null}
            </CardContent>
          </Card>
        </>
      ) : (
        <Card>
          <CardContent className="py-8 text-center text-sm text-muted-foreground">
            No organization found. Create one via{" "}
            <code className="text-xs">POST /tenants</code> or register a new account.
          </CardContent>
        </Card>
      )}
    </div>
  );
}
