"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { auditApi, type AuditEvent } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ScrollText,
  Search,
  Download,
  RefreshCw,
  Shield,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  Filter,
} from "lucide-react";

const EVENT_TYPE_GROUPS = [
  { value: "memory", label: "Memory", types: "memory.create,memory.read,memory.update,memory.delete,memory.search" },
  { value: "skill", label: "Skill", types: "skill.create,skill.update,skill.delete,skill.use,skill.execute" },
  { value: "agent", label: "Agent", types: "agent.create,agent.update,agent.delete" },
  { value: "group", label: "Group", types: "group.create,group.add_member,group.remove_member" },
  { value: "auth", label: "Auth", types: "auth.login,auth.logout,auth.sso" },
  { value: "api", label: "API", types: "api.access" },
];

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHr = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHr / 24);

  if (diffSec < 60) return "just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHr < 24) return `${diffHr}h ago`;
  if (diffDay < 7) return `${diffDay}d ago`;
  return date.toLocaleDateString();
}

function StatusBadge({ status }: { status: string }) {
  if (status === "success") {
    return (
      <Badge variant="outline" className="gap-1 text-green-600 border-green-200 bg-green-50">
        <CheckCircle className="h-3 w-3" />
        Success
      </Badge>
    );
  }
  return (
    <Badge variant="outline" className="gap-1 text-red-600 border-red-200 bg-red-50">
      <XCircle className="h-3 w-3" />
      Failed
    </Badge>
  );
}

function EventTypeBadge({ type }: { type: string }) {
  const category = type.split(".")[0];
  const colorMap: Record<string, string> = {
    memory: "bg-blue-100 text-blue-700 border-blue-200",
    skill: "bg-purple-100 text-purple-700 border-purple-200",
    agent: "bg-green-100 text-green-700 border-green-200",
    group: "bg-orange-100 text-orange-700 border-orange-200",
    auth: "bg-yellow-100 text-yellow-700 border-yellow-200",
    chain: "bg-pink-100 text-pink-700 border-pink-200",
    review: "bg-indigo-100 text-indigo-700 border-indigo-200",
    license: "bg-cyan-100 text-cyan-700 border-cyan-200",
    api: "bg-gray-100 text-gray-700 border-gray-200",
  };

  return (
    <Badge variant="outline" className={colorMap[category] || colorMap.api}>
      {type}
    </Badge>
  );
}

export default function AuditPage() {
  const [searchActor, setSearchActor] = useState("");
  const [eventTypeFilter, setEventTypeFilter] = useState<string>("all");
  const [page, setPage] = useState(0);
  const pageSize = 50;

  const typesParam =
    eventTypeFilter !== "all"
      ? EVENT_TYPE_GROUPS.find((g) => g.value === eventTypeFilter)?.types
      : undefined;

  const { data: events, isLoading, refetch } = useQuery({
    queryKey: ["audit-events", eventTypeFilter, searchActor, page],
    queryFn: () =>
      auditApi.query({
        types: typesParam,
        actor_id: searchActor || undefined,
        limit: pageSize,
        offset: page * pageSize,
      }),
  });

  const auditEvents: AuditEvent[] = Array.isArray(events) ? events : [];

  const stats = {
    total: auditEvents.length,
    success: auditEvents.filter((e) => e.status === "success").length,
    failed: auditEvents.filter((e) => e.status === "failure").length,
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <ScrollText className="h-8 w-8" />
            Audit Trail
          </h1>
          <p className="text-muted-foreground">
            Track all actions across your memory infrastructure
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              const url = `${process.env.NEXT_PUBLIC_API_URL || ""}/audit/export?format=csv`;
              window.open(url, "_blank");
            }}
          >
            <Download className="h-4 w-4 mr-2" />
            Export CSV
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-primary/10 p-2">
                <Shield className="h-5 w-5 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold tabular-nums">{stats.total}</p>
                <p className="text-sm text-muted-foreground">Total Events</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-green-100 p-2">
                <CheckCircle className="h-5 w-5 text-green-600" />
              </div>
              <div>
                <p className="text-2xl font-bold tabular-nums">{stats.success}</p>
                <p className="text-sm text-muted-foreground">Successful</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-3">
              <div className="rounded-lg bg-red-100 p-2">
                <AlertTriangle className="h-5 w-5 text-red-600" />
              </div>
              <div>
                <p className="text-2xl font-bold tabular-nums">{stats.failed}</p>
                <p className="text-sm text-muted-foreground">Failed</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <CardTitle className="flex items-center gap-2">
              <Filter className="h-5 w-5" />
              Events
            </CardTitle>
            <div className="flex gap-2">
              <div className="relative w-64">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Filter by actor ID..."
                  value={searchActor}
                  onChange={(e) => {
                    setSearchActor(e.target.value);
                    setPage(0);
                  }}
                  className="pl-9"
                />
              </div>
              <Select
                value={eventTypeFilter}
                onValueChange={(v) => {
                  setEventTypeFilter(v);
                  setPage(0);
                }}
              >
                <SelectTrigger className="w-40">
                  <SelectValue placeholder="Event type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Events</SelectItem>
                  {EVENT_TYPE_GROUPS.map((g) => (
                    <SelectItem key={g.value} value={g.value}>
                      {g.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 8 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : auditEvents.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <ScrollText className="h-12 w-12 mb-4 opacity-50" />
              <p className="font-medium">No audit events found</p>
              <p className="text-sm">Events will appear as actions are performed</p>
            </div>
          ) : (
            <>
              <div className="rounded-lg border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Time</TableHead>
                      <TableHead>Event</TableHead>
                      <TableHead>Action</TableHead>
                      <TableHead>Actor</TableHead>
                      <TableHead>Resource</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className="text-right">Duration</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {auditEvents.map((event) => (
                      <TableRow key={event.id}>
                        <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                          <div className="flex items-center gap-1">
                            <Clock className="h-3 w-3" />
                            {formatRelativeTime(event.timestamp)}
                          </div>
                        </TableCell>
                        <TableCell>
                          <EventTypeBadge type={event.type} />
                        </TableCell>
                        <TableCell className="text-sm">{event.action}</TableCell>
                        <TableCell className="text-sm font-mono text-muted-foreground">
                          {event.actor_id
                            ? event.actor_id.length > 12
                              ? event.actor_id.slice(0, 12) + "..."
                              : event.actor_id
                            : "-"}
                        </TableCell>
                        <TableCell className="text-sm">
                          {event.resource_type && event.resource_id ? (
                            <span className="font-mono text-xs">
                              {event.resource_type}/{event.resource_id.slice(0, 8)}
                            </span>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                        <TableCell>
                          <StatusBadge status={event.status} />
                        </TableCell>
                        <TableCell className="text-right text-sm tabular-nums text-muted-foreground">
                          {event.duration_ms ? `${event.duration_ms}ms` : "-"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <div className="flex items-center justify-between mt-4">
                <p className="text-sm text-muted-foreground">
                  Showing {auditEvents.length} events
                  {page > 0 && ` (page ${page + 1})`}
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page === 0}
                    onClick={() => setPage((p) => p - 1)}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={auditEvents.length < pageSize}
                    onClick={() => setPage((p) => p + 1)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
