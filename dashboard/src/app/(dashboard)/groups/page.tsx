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
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { FilterComponent } from "@/components/ui/filter-component";
import { Users, Plus, Bot, Trash2, RefreshCw, Brain, BookOpen } from "lucide-react";
import { toast } from "sonner";
import { groupsApi, agentsApi, type AgentGroup, type Skill, type Memory } from "@/lib/api";

const GROUP_SIZE_OPTIONS = [
  { value: "all", label: "All" },
  { value: "small", label: "Small (1-5)" },
  { value: "large", label: "Large (5+)" },
];

export default function GroupsPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [sizeFilter, setSizeFilter] = useState("all");
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
  const [isSaving, setIsSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [selectedAgentId, setSelectedAgentId] = useState<string>("");
  const [selectedRole, setSelectedRole] = useState<string>("member");

  // Group detail tab data
  const [groupSkills, setGroupSkills] = useState<Skill[]>([]);
  const [groupMemories, setGroupMemories] = useState<Memory[]>([]);
  const [isLoadingSkills, setIsLoadingSkills] = useState(false);
  const [isLoadingMemories, setIsLoadingMemories] = useState(false);

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

  const fetchGroupSkills = useCallback(async (groupId: string) => {
    try {
      setIsLoadingSkills(true);
      const response = await groupsApi.getSkills(groupId);
      setGroupSkills(response.skills || []);
    } catch (error) {
      console.error("Failed to fetch group skills:", error);
      setGroupSkills([]);
    } finally {
      setIsLoadingSkills(false);
    }
  }, []);

  const fetchGroupMemories = useCallback(async (groupId: string) => {
    try {
      setIsLoadingMemories(true);
      const response = await groupsApi.getMemories(groupId);
      setGroupMemories(response.memories || []);
    } catch (error) {
      console.error("Failed to fetch group memories:", error);
      setGroupMemories([]);
    } finally {
      setIsLoadingMemories(false);
    }
  }, []);

  const handleCreate = async () => {
    if (!newGroup.name.trim()) {
      toast.error("Group name is required");
      return;
    }

    try {
      setIsSaving(true);
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
      setIsSaving(false);
    }
  };

  const handleEdit = async () => {
    if (!selectedGroup || !editGroup.name.trim()) {
      toast.error("Group name is required");
      return;
    }

    try {
      setIsSaving(true);
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
      setIsSaving(false);
    }
  };

  const handleAddMember = async () => {
    if (!selectedGroup || !selectedAgentId) {
      toast.error("Please select an agent");
      return;
    }

    try {
      await groupsApi.addMember(selectedGroup.id, selectedAgentId, selectedRole);
      const refreshed = await groupsApi.get(selectedGroup.id);
      setSelectedGroup(refreshed);
      toast.success("Agent added to group");
      setSelectedAgentId("");
      setSelectedRole("member");
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
      const refreshed = await groupsApi.get(selectedGroup.id);
      setSelectedGroup(refreshed);
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
    setSizeFilter("all");
    setDateFrom(null);
    setDateTo(null);
  };

  const filteredGroups = groups.filter((group) => {
    const matchesSearch =
      searchQuery === "" ||
      group.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (group.description?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false);

    const memberCount = group.member_count ?? group.members?.length ?? 0;
    const matchesSize =
      sizeFilter === "all" ||
      (sizeFilter === "small" && memberCount >= 1 && memberCount <= 5) ||
      (sizeFilter === "large" && memberCount > 5);

    const groupDate = new Date(group.created_at || Date.now());
    const matchesFrom = !dateFrom || groupDate >= dateFrom;
    const matchesTo = !dateTo || groupDate <= dateTo;

    return matchesSearch && matchesSize && matchesFrom && matchesTo;
  });

  const openEditDialog = (group: AgentGroup) => {
    setSelectedGroup(group);
    setEditGroup({ name: group.name, description: group.description || "" });
    setIsEditOpen(true);
  };

  const openMembersDialog = async (group: AgentGroup) => {
    setSelectedGroup(group);
    setGroupSkills([]);
    setGroupMemories([]);
    setIsMembersOpen(true);
    try {
      const detailed = await groupsApi.get(group.id);
      setSelectedGroup(detailed);
    } catch (error) {
      console.error("Failed to fetch group details:", error);
      toast.error("Failed to load group details");
    }
    // Pre-fetch skills and memories when opening
    fetchGroupSkills(group.id);
    fetchGroupMemories(group.id);
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
                <Button onClick={handleCreate} disabled={isSaving}>
                  {isSaving ? "Creating..." : "Create Group"}
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
        typeValue={sizeFilter}
        onTypeChange={setSizeFilter}
        typeOptions={GROUP_SIZE_OPTIONS}
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
            {(searchQuery || sizeFilter !== "all") && (
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
                          {agent.name || agent.agent_id || agent.id}
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

      {/* Members / Skills / Memories Dialog */}
      <Dialog open={isMembersOpen} onOpenChange={setIsMembersOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Group Details</DialogTitle>
            <DialogDescription>
              {selectedGroup?.name} — Manage members, view skills and memories
            </DialogDescription>
          </DialogHeader>

          <Tabs defaultValue="members" className="mt-2">
            <TabsList className="w-full">
              <TabsTrigger value="members" className="flex-1">
                <Users className="mr-1 h-4 w-4" />
                Members
              </TabsTrigger>
              <TabsTrigger value="skills" className="flex-1">
                <Brain className="mr-1 h-4 w-4" />
                Skills
              </TabsTrigger>
              <TabsTrigger value="memories" className="flex-1">
                <BookOpen className="mr-1 h-4 w-4" />
                Memories
              </TabsTrigger>
            </TabsList>

            {/* Members Tab */}
            <TabsContent value="members" className="space-y-4 py-2">
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
                <Select
                  value={selectedRole}
                  onValueChange={(v) => setSelectedRole(v ?? "member")}
                >
                  <SelectTrigger className="w-32">
                    <SelectValue placeholder="Role" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="admin">Admin</SelectItem>
                    <SelectItem value="leader">Leader</SelectItem>
                    <SelectItem value="member">Member</SelectItem>
                    <SelectItem value="contributor">Contributor</SelectItem>
                    <SelectItem value="viewer">Viewer</SelectItem>
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
                    {selectedGroup.members.map((member) => {
                      const agent = agents.find((a) => a.id === member.agent_id);
                      return (
                      <div key={member.agent_id} className="flex items-center justify-between p-2 border rounded">
                        <div className="flex items-center gap-2">
                          <Bot className="h-4 w-4" />
                          <span>{agent?.name || member.agent_id}</span>
                          <Badge variant="secondary" className="text-xs capitalize">
                            {member.role}
                          </Badge>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveMember(member.agent_id)}
                        >
                          Remove
                        </Button>
                      </div>
                    )})}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No members yet</p>
                )}
              </div>
            </TabsContent>

            {/* Skills Tab */}
            <TabsContent value="skills" className="py-2">
              {isLoadingSkills ? (
                <div className="space-y-2">
                  {[1, 2, 3].map(i => <Skeleton key={i} className="h-12 w-full" />)}
                </div>
              ) : groupSkills.length > 0 ? (
                <div className="space-y-2 max-h-72 overflow-y-auto">
                  {groupSkills.map((skill) => (
                    <div key={skill.id} className="flex items-center justify-between p-3 border rounded">
                      <div>
                        <p className="font-medium text-sm">{skill.name}</p>
                        <p className="text-xs text-muted-foreground">{skill.description}</p>
                      </div>
                      <div className="flex items-center gap-2 shrink-0 ml-2">
                        {skill.domain && (
                          <Badge variant="outline" className="text-xs">{skill.domain}</Badge>
                        )}
                        {skill.usage_count !== undefined && (
                          <span className="text-xs text-muted-foreground tabular-nums">
                            {(skill.usage_count * 100).toFixed(0)}%
                          </span>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-8 text-center">
                  <Brain className="h-8 w-8 text-muted-foreground mb-2" />
                  <p className="text-sm text-muted-foreground">No skills shared with this group</p>
                </div>
              )}
            </TabsContent>

            {/* Memories Tab */}
            <TabsContent value="memories" className="py-2">
              {isLoadingMemories ? (
                <div className="space-y-2">
                  {[1, 2, 3].map(i => <Skeleton key={i} className="h-14 w-full" />)}
                </div>
              ) : groupMemories.length > 0 ? (
                <div className="space-y-2 max-h-72 overflow-y-auto">
                  {groupMemories.map((memory) => (
                    <div key={memory.id} className="p-3 border rounded space-y-1">
                      <p className="text-sm line-clamp-2">{memory.content}</p>
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="text-xs capitalize">{memory.type}</Badge>
                        {memory.category && (
                          <Badge variant="secondary" className="text-xs">{memory.category}</Badge>
                        )}
                        <span className="text-xs text-muted-foreground ml-auto">
                          {new Date(memory.created_at).toLocaleDateString()}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-8 text-center">
                  <BookOpen className="h-8 w-8 text-muted-foreground mb-2" />
                  <p className="text-sm text-muted-foreground">No shared memories found for this group</p>
                </div>
              )}
            </TabsContent>
          </Tabs>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
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
            <Button onClick={handleEdit} disabled={isSaving}>
              {isSaving ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
