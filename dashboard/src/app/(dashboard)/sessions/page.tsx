"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { sessionsApi, type Session } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { FilterComponent } from "@/components/ui/filter-component";
import { MoreHorizontal, Plus, Bot, MessageSquare, Trash2, Eye, RefreshCw } from "lucide-react";
import { toast } from "sonner";

export default function SessionsPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isViewOpen, setIsViewOpen] = useState(false);
  const [selectedSession, setSelectedSession] = useState<Session | null>(null);
  const [newSession, setNewSession] = useState({ agent_id: "" });
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const queryClient = useQueryClient();

  const { data: sessionsData, isLoading, refetch } = useQuery({
    queryKey: ["sessions"],
    queryFn: () => sessionsApi.list(),
  });

  const { data: messagesData, isLoading: messagesLoading } = useQuery({
    queryKey: ["session-messages", selectedSession?.id],
    queryFn: () => sessionsApi.getMessages(selectedSession!.id, { limit: 50 }),
    enabled: !!selectedSession?.id,
  });

  const createMutation = useMutation({
    mutationFn: (data: { agent_id: string }) => sessionsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions"] });
      setIsCreateOpen(false);
      setNewSession({ agent_id: "" });
      toast.success("Session created successfully");
    },
    onError: () => {
      toast.error("Failed to create session");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => sessionsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sessions"] });
      toast.success("Session deleted");
    },
    onError: () => {
      toast.error("Failed to delete session");
    },
  });

  const handleCreate = () => {
    if (!newSession.agent_id.trim()) {
      toast.error("Agent ID is required");
      return;
    }
    createMutation.mutate({ agent_id: newSession.agent_id });
  };

  const handleDelete = (id: string) => {
    if (!confirm("Are you sure you want to delete this session?")) return;
    setDeletingId(id);
    deleteMutation.mutate(id);
  };

  const handleViewSession = (session: Session) => {
    setSelectedSession(session);
    setIsViewOpen(true);
  };

  const sessions = sessionsData?.sessions || [];
  const messages = messagesData?.messages || [];

  const clearFilters = () => {
    setSearchQuery("");
    setDateFrom(null);
    setDateTo(null);
  };

  const filteredSessions = sessions.filter((session) => {
    const matchesSearch =
      searchQuery === "" ||
      session.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      session.agent_id.toLowerCase().includes(searchQuery.toLowerCase());

    const sessionDate = new Date(session.created_at || Date.now());
    const matchesFrom = !dateFrom || sessionDate >= dateFrom;
    const matchesTo = !dateTo || sessionDate <= dateTo;

    return matchesSearch && matchesFrom && matchesTo;
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Sessions</h1>
          <p className="text-muted-foreground">View and manage agent conversation sessions</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                New Session
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create New Session</DialogTitle>
                <DialogDescription>Start a new conversation session with an agent</DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="agent_id">Agent ID</Label>
                  <Input
                    id="agent_id"
                    placeholder="Enter agent ID..."
                    value={newSession.agent_id}
                    onChange={(e) => setNewSession({ agent_id: e.target.value })}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                <Button onClick={handleCreate} disabled={createMutation.isPending}>
                  {createMutation.isPending ? "Creating..." : "Create Session"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search sessions..."
        typeValue="all"
        onTypeChange={() => {}}
        typeOptions={[]}
        dateFrom={dateFrom}
        onDateFromChange={setDateFrom}
        dateTo={dateTo}
        onDateToChange={setDateTo}
        onClear={clearFilters}
      />

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Session ID</TableHead>
                <TableHead>Agent</TableHead>
                <TableHead>Started</TableHead>
                <TableHead>Last Updated</TableHead>
                <TableHead className="w-[50px]"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-8">
                    <Skeleton className="h-8 w-full" />
                  </TableCell>
                </TableRow>
              ) : filteredSessions.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-8">
                    <Bot className="mx-auto h-12 w-12 text-muted-foreground/50" />
                    <p className="mt-2 text-muted-foreground">No sessions found</p>
                    {searchQuery && (
                      <Button variant="ghost" onClick={clearFilters} className="mt-2">
                        Clear filters
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ) : (
                filteredSessions.map((session) => (
                  <TableRow key={session.id}>
                    <TableCell className="font-mono text-sm">{session.id}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{session.agent_id}</Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDateTime(session.created_at)}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDateTime(session.updated_at)}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => handleViewSession(session)}>
                            <Eye className="mr-2 h-4 w-4" />
                            View Messages
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem 
                            className="text-destructive"
                            onClick={() => handleDelete(session.id)}
                            disabled={deletingId === session.id}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={isViewOpen} onOpenChange={(open) => {
        setIsViewOpen(open);
        if (!open) setSelectedSession(null);
      }}>
        <DialogContent className="sm:max-w-[700px]">
          <DialogHeader>
            <DialogTitle>Session Messages</DialogTitle>
            <DialogDescription className="font-mono text-xs">{selectedSession?.id}</DialogDescription>
          </DialogHeader>
          <div className="max-h-[400px] space-y-4 overflow-y-auto py-4">
            {messagesLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-16 w-full" />
                <Skeleton className="h-16 w-full" />
              </div>
            ) : messages.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                <MessageSquare className="mx-auto h-12 w-12 mb-4" />
                <p>No messages in this session</p>
              </div>
            ) : (
              messages.map((message) => (
                <div
                  key={message.id}
                  className={`flex ${message.role === "user" ? "justify-end" : "justify-start"}`}
                >
                  <div
                    className={`max-w-[80%] rounded-lg p-3 ${
                      message.role === "user"
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted"
                    }`}
                  >
                    <div className="mb-1 flex items-center gap-2">
                      <MessageSquare className="h-3 w-3" />
                      <span className="text-xs font-medium capitalize">{message.role}</span>
                      <span className="text-xs opacity-70">{formatDateTime(message.created_at)}</span>
                    </div>
                    <p className="text-sm">{message.content}</p>
                  </div>
                </div>
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}