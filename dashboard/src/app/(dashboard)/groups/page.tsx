"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { FilterComponent } from "@/components/ui/filter-component";
import { Users, Plus, Crown, Bot, Trash2, RefreshCw, Settings } from "lucide-react";
import { toast } from "sonner";
import { groupsApi, agentsApi, type AgentGroup } from "@/lib/api";

export default function GroupsPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [groups, setGroups] = useState<AgentGroup[]>([]);
  const [agents, setAgents] = useState<any[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isMembersOpen, setIsMembersOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [selectedGroup, setSelectedGroup] = useState<AgentGroup | null>(null);
  const [newGroup, setNewGroup] = useState({ name: "", description: "" });
  const [editGroup, setEditGroup] = useState({ name: "", description: "" });
  const [isCreating, setIsCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [selectedAgentId, setSelectedAgentId] = useState<string>("");

  const fetchGroups = useCallback(async () => {
    try {
      setIsLoading(true);
      const response = await groupsApi.list();
      setGroups(response.groups || []);
    } catch (error) {
      console.error("Failed to fetch groups:", error);
      toast.error("Failed to load groups");
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchAgents = useCallback(async () => {
    try {
      const response = await agentsApi.list();
      setAgents(response.agents || []);
    } catch (error) {
      console.error("Failed to fetch agents:", error);
    }
  }, []);

  useEffect(() => {
    fetchGroups();
    fetchAgents();
  }, [fetchGroups, fetchAgents]);

  const handleCreate = async () => {
    if (!newGroup.name.trim()) {
      toast.error("Group name is required");
      return;
    }

    try {
      setIsCreating(true);
      const created = await groupsApi.create({
        name: newGroup.name,
        description: newGroup.description,
      });
      setGroups(prev => [...prev, created]);
      setIsCreateOpen(false);
      setNewGroup({ name: "", description: "" });
      toast.success("Group created successfully");
    } catch (error) {
      console.error("Failed to create group:", error);
      toast.error("Failed to create group");
    } finally {
      setIsCreating(false);
    }
  };

  const handleEdit = async () => {
    if (!selectedGroup || !editGroup.name.trim()) {
      toast.error("Group name is required");
      return;
    }

    try {
      setIsCreating(true);
      await groupsApi.update(selectedGroup.id, {
        name: editGroup.name,
        description: editGroup.description,
      });
      setGroups(prev => prev.map(g => 
        g.id === selectedGroup.id ? { ...g, name: editGroup.name, description: editGroup.description } : g
      ));
      setIsEditOpen(false);
      setSelectedGroup(null);
      toast.success("Group updated successfully");
    } catch (error) {
      console.error("Failed to update group:", error);
      toast.error("Failed to update group");
    } finally {
      setIsCreating(false);
    }
  };

  const handleAddMember = async () => {
    if (!selectedGroup || !selectedAgentId) {
      toast.error("Please select an agent");
      return;
    }

    try {
      await groupsApi.addMember(selectedGroup.id, selectedAgentId, "member");
      toast.success("Agent added to group");
      setSelectedAgentId("");
      fetchGroups();
    } catch (error) {
      console.error("Failed to add member:", error);
      toast.error("Failed to add member");
    }
  };

  const handleRemoveMember = async (agentId: string) => {
    if (!selectedGroup) return;
    
    try {
      await groupsApi.removeMember(selectedGroup.id, agentId);
      toast.success("Agent removed from group");
      fetchGroups();
    } catch (error) {
      console.error("Failed to remove member:", error);
      toast.error("Failed to remove member");
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this group?")) return;

    try {
      setDeletingId(id);
      await groupsApi.delete(id);
      setGroups(prev => prev.filter(g => g.id !== id));
      toast.success("Group deleted");
    } catch (error) {
      console.error("Failed to delete group:", error);
      toast.error("Failed to delete group");
    } finally {
      setDeletingId(null);
    }
  };

  const clearFilters = () => {
    setSearchQuery("");
    setDateFrom(null);
    setDateTo(null);
  };

  const filteredGroups = groups.filter((group) => {
    const matchesSearch =
      searchQuery === "" ||
      group.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (group.description?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false);

    const groupDate = new Date(group.created_at || Date.now());
    const matchesFrom = !dateFrom || groupDate >= dateFrom;
    const matchesTo = !dateTo || groupDate <= dateTo;

    return matchesSearch && matchesFrom && matchesTo;
  });

  const openEditDialog = (group: AgentGroup) => {
    setSelectedGroup(group);
    setEditGroup({ name: group.name, description: group.description || "" });
    setIsEditOpen(true);
  };

  const openMembersDialog = (group: AgentGroup) => {
    setSelectedGroup(group);
    setIsMembersOpen(true);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Groups</h1>
          <p className="text-muted-foreground">Organize agents into teams and divisions</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={fetchGroups}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Create Group
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create New Group</DialogTitle>
                <DialogDescription>Create a new agent group for collaboration</DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">Group Name</Label>
                  <Input
                    id="name"
                    placeholder="Enter group name..."
                    value={newGroup.name}
                    onChange={(e) => setNewGroup({ ...newGroup, name: e.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="description">Description (optional)</Label>
                  <Input
                    id="description"
                    placeholder="Enter description..."
                    value={newGroup.description}
                    onChange={(e) => setNewGroup({ ...newGroup, description: e.target.value })}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                <Button onClick={handleCreate} disabled={isCreating}>
                  {isCreating ? "Creating..." : "Create Group"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search groups..."
        typeValue="all"
        onTypeChange={() => {}}
        typeOptions={[]}
        dateFrom={dateFrom}
        onDateFromChange={setDateFrom}
        dateTo={dateTo}
        onDateToChange={setDateTo}
        onClear={clearFilters}
      />

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map(i => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-5 w-32" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-4 w-full mb-2" />
                <Skeleton className="h-8 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : filteredGroups.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Users className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">No groups found</p>
            {searchQuery && (
              <Button variant="ghost" onClick={clearFilters} className="mt-2">
                Clear filters
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredGroups.map((group) => (
            <Card key={group.id} className="card-hover">
              <CardHeader className="flex flex-row items-center justify-between space-y-0">
                <div className="flex items-center gap-3">
                  <div className="rounded-lg bg-primary/10 p-2">
                    <Users className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <CardTitle className="text-lg">{group.name}</CardTitle>
                    <p className="text-sm text-muted-foreground">
                      {group.member_count ?? group.members?.length ?? 0} members
                    </p>
                  </div>
                </div>
                <Button 
                  variant="ghost" 
                  size="icon"
                  onClick={() => handleDelete(group.id)}
                  disabled={deletingId === group.id}
                >
                  <Trash2 className="h-4 w-4 text-muted-foreground" />
                </Button>
              </CardHeader>
              <CardContent>
                {group.description && (
                  <p className="text-sm text-muted-foreground mb-3 line-clamp-2">
                    {group.description}
                  </p>
                )}
                <div className="space-y-3">
                  {group.members && group.members.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {group.members.slice(0, 4).map((agent, i) => (
                        <Badge key={i} variant="outline" className="text-xs">
                          <Bot className="mr-1 h-3 w-3" />
                          {agent.name || agent.id}
                        </Badge>
                      ))}
                      {group.members.length > 4 && (
                        <Badge variant="outline" className="text-xs">
                          +{group.members.length - 4} more
                        </Badge>
                      )}
                    </div>
                  )}
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" className="flex-1" onClick={() => openMembersDialog(group)}>
                      View Members
                    </Button>
                    <Button variant="outline" size="sm" className="flex-1" onClick={() => openEditDialog(group)}>
                      Edit
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={isMembersOpen} onOpenChange={setIsMembersOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Group Members</DialogTitle>
            <DialogDescription>
              {selectedGroup?.name} - Manage members
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="flex gap-2">
              <Select value={selectedAgentId} onValueChange={(v) => v && setSelectedAgentId(v)}>
                <SelectTrigger className="flex-1">
                  <SelectValue placeholder="Select an agent to add" />
                </SelectTrigger>
                <SelectContent>
                  {agents.map(agent => (
                    <SelectItem key={agent.id} value={agent.id}>
                      {agent.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button onClick={handleAddMember} disabled={!selectedAgentId}>
                Add
              </Button>
            </div>
            <div className="space-y-2">
              <Label>Current Members ({selectedGroup?.members?.length || 0})</Label>
              {selectedGroup?.members && selectedGroup.members.length > 0 ? (
                <div className="space-y-2 max-h-60 overflow-y-auto">
                  {selectedGroup.members.map((agent: any) => (
                    <div key={agent.id} className="flex items-center justify-between p-2 border rounded">
                      <div className="flex items-center gap-2">
                        <Bot className="h-4 w-4" />
                        <span>{agent.name || agent.id}</span>
                      </div>
                      <Button 
                        variant="ghost" 
                        size="sm"
                        onClick={() => handleRemoveMember(agent.id)}
                      >
                        Remove
                      </Button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No members yet</p>
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Group</DialogTitle>
            <DialogDescription>
              Update group settings
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Group Name</Label>
              <Input
                id="edit-name"
                value={editGroup.name}
                onChange={(e) => setEditGroup({ ...editGroup, name: e.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-description">Description</Label>
              <Input
                id="edit-description"
                value={editGroup.description}
                onChange={(e) => setEditGroup({ ...editGroup, description: e.target.value })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsEditOpen(false)}>Cancel</Button>
            <Button onClick={handleEdit} disabled={isCreating}>
              {isCreating ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}