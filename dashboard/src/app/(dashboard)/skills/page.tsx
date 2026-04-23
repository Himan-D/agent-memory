"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { skillsApi, type Skill } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { FilterComponent } from "@/components/ui/filter-component";
import { Sparkles, Plus, Trash2, Edit, Play, RefreshCw } from "lucide-react";
import { toast } from "sonner";

const domainColors: Record<string, string> = {
  database: "bg-blue-500/10 text-blue-600 border-blue-500/20",
  development: "bg-green-500/10 text-green-600 border-green-500/20",
  infrastructure: "bg-purple-500/10 text-purple-600 border-purple-500/20",
  quality: "bg-orange-500/10 text-orange-600 border-orange-500/20",
  security: "bg-red-500/10 text-red-600 border-red-500/20",
  ai: "bg-pink-500/10 text-pink-600 border-pink-500/20",
};

const domains = ["database", "development", "infrastructure", "quality", "security", "ai"];

export default function SkillsPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [domainFilter, setDomainFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [editingSkill, setEditingSkill] = useState<Skill | null>(null);
  const [testingSkill, setTestingSkill] = useState<Skill | null>(null);
  const [testInput, setTestInput] = useState("");
  const [testResult, setTestResult] = useState("");
  const [isTesting, setIsTesting] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [newSkill, setNewSkill] = useState({
    name: "",
    description: "",
    trigger: "",
    domain: "",
    prompt: "",
  });

  const queryClient = useQueryClient();

  const { data: skillsData, isLoading, refetch } = useQuery({
    queryKey: ["skills"],
    queryFn: () => skillsApi.list(),
  });

  const createMutation = useMutation({
    mutationFn: (data: { name: string; description: string; trigger: string; domain: string; prompt: string }) =>
      skillsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["skills"] });
      setIsCreateOpen(false);
      setNewSkill({ name: "", description: "", trigger: "", domain: "", prompt: "" });
      toast.success("Skill created successfully");
    },
    onError: () => {
      toast.error("Failed to create skill");
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Skill> }) => skillsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["skills"] });
      setEditingSkill(null);
      toast.success("Skill updated successfully");
    },
    onError: () => {
      toast.error("Failed to update skill");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => skillsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["skills"] });
      toast.success("Skill deleted");
    },
    onError: () => {
      toast.error("Failed to delete skill");
    },
  });

  const handleCreate = () => {
    if (!newSkill.name.trim()) {
      toast.error("Skill name is required");
      return;
    }
    if (!newSkill.trigger.trim()) {
      toast.error("Trigger is required");
      return;
    }
    if (!newSkill.domain) {
      toast.error("Domain is required");
      return;
    }
    createMutation.mutate(newSkill);
  };

  const handleUpdate = () => {
    if (!editingSkill) return;
    if (!editingSkill.name.trim()) {
      toast.error("Skill name is required");
      return;
    }
    updateMutation.mutate({
      id: editingSkill.id,
      data: {
        name: editingSkill.name,
        description: editingSkill.description || editingSkill.name,
        trigger: editingSkill.trigger,
        domain: editingSkill.domain,
      },
    });
  };

  const handleDelete = (id: string) => {
    if (!confirm("Are you sure you want to delete this skill?")) return;
    setDeletingId(id);
    deleteMutation.mutate(id);
  };

  const handleTest = async () => {
    if (!testingSkill || !testInput.trim()) return;
    
    setIsTesting(true);
    setTestResult("");
    
    try {
      const result = await skillsApi.use(testingSkill.id, { input: testInput });
      setTestResult(JSON.stringify(result, null, 2));
      toast.success("Skill executed successfully");
    } catch (error) {
      toast.error("Failed to execute skill");
      setTestResult("Error: Failed to execute skill");
    }
    
    setIsTesting(false);
  };

  const openTestDialog = (skill: Skill) => {
    setTestingSkill(skill);
    setTestInput("");
    setTestResult("");
  };

  const skills = skillsData?.skills || [];

  const clearFilters = () => {
    setSearchQuery("");
    setDomainFilter("all");
    setDateFrom(null);
    setDateTo(null);
  };

  const filteredSkills = skills.filter((skill) => {
    const matchesSearch =
      searchQuery === "" ||
      skill.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      skill.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      skill.trigger?.toLowerCase().includes(searchQuery.toLowerCase());

    const matchesDomain = domainFilter === "all" || skill.domain === domainFilter;

    const skillDate = new Date(skill.created_at || Date.now());
    const matchesFrom = !dateFrom || skillDate >= dateFrom;
    const matchesTo = !dateTo || skillDate <= dateTo;

    return matchesSearch && matchesDomain && matchesFrom && matchesTo;
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Skills</h1>
          <p className="text-muted-foreground">Built-in and custom agent skills</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Create Skill
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[600px]">
              <DialogHeader>
                <DialogTitle>Create New Skill</DialogTitle>
                <DialogDescription>Create a custom skill for your agents</DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">Skill Name</Label>
                  <Input
                    id="name"
                    placeholder="e.g., Code Reviewer"
                    value={newSkill.name}
                    onChange={(e) => setNewSkill({ ...newSkill, name: e.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="description">Description</Label>
                  <Input
                    id="description"
                    placeholder="Brief description of the skill"
                    value={newSkill.description}
                    onChange={(e) => setNewSkill({ ...newSkill, description: e.target.value })}
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="trigger">Trigger</Label>
                    <Input
                      id="trigger"
                      placeholder="e.g., code_review"
                      value={newSkill.trigger}
                      onChange={(e) => setNewSkill({ ...newSkill, trigger: e.target.value })}
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="domain">Domain</Label>
                    <Select onValueChange={(v) => setNewSkill({ ...newSkill, domain: v as string })}>
                      <SelectTrigger>
                        <SelectValue placeholder="Select domain" />
                      </SelectTrigger>
                      <SelectContent>
                        {domains.map(d => (
                          <SelectItem key={d} value={d}>{d}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="prompt">System Prompt</Label>
                  <Textarea
                    id="prompt"
                    placeholder="Enter the system prompt for this skill..."
                    value={newSkill.prompt}
                    onChange={(e) => setNewSkill({ ...newSkill, prompt: e.target.value })}
                    className="min-h-[100px]"
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                <Button onClick={handleCreate} disabled={createMutation.isPending}>
                  {createMutation.isPending ? "Creating..." : "Create Skill"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search skills..."
        typeValue={domainFilter}
        onTypeChange={setDomainFilter}
        typeOptions={[
          { label: "All Domains", value: "all" },
          { label: "Database", value: "database" },
          { label: "Development", value: "development" },
          { label: "Infrastructure", value: "infrastructure" },
          { label: "Quality", value: "quality" },
          { label: "Security", value: "security" },
          { label: "AI", value: "ai" },
        ]}
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
      ) : filteredSkills.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Sparkles className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">No skills found</p>
            {searchQuery && (
              <Button variant="ghost" onClick={clearFilters} className="mt-2">
                Clear filters
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredSkills.map((skill) => (
            <Card key={skill.id} className="card-hover">
              <CardHeader className="space-y-0 pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-primary/10 p-2">
                      <Sparkles className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                      <CardTitle className="text-lg">{skill.name}</CardTitle>
                      {skill.is_builtin && (
                        <Badge variant="secondary" className="mt-1 text-xs">
                          Built-in
                        </Badge>
                      )}
                    </div>
                  </div>
                  <Button 
                    variant="ghost" 
                    size="icon"
                    onClick={() => handleDelete(skill.id)}
                    disabled={deletingId === skill.id}
                  >
                    <Trash2 className="h-4 w-4 text-muted-foreground" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground mb-3 line-clamp-2">{skill.description || skill.name}</p>
                <div className="flex flex-wrap gap-2 mb-3">
                  <Badge variant="outline" className={domainColors[skill.domain] || ""}>
                    {skill.domain}
                  </Badge>
                  <Badge variant="outline" className="text-xs">
                    Trigger: {skill.trigger}
                  </Badge>
                </div>
                <div className="flex items-center justify-between text-sm text-muted-foreground mb-3">
                  <span>Used {skill.usage_count || 0} times</span>
                  <span>{formatDateTime(skill.created_at)}</span>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" className="flex-1" onClick={() => setEditingSkill(skill)}>
                    <Edit className="mr-1 h-3 w-3" />
                    Edit
                  </Button>
                  <Button variant="outline" size="sm" className="flex-1" onClick={() => openTestDialog(skill)}>
                    <Play className="mr-1 h-3 w-3" />
                    Test
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Edit Skill Dialog */}
      <Dialog open={!!editingSkill} onOpenChange={() => setEditingSkill(null)}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Edit Skill</DialogTitle>
            <DialogDescription>Update skill configuration</DialogDescription>
          </DialogHeader>
          {editingSkill && (
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="edit-name">Skill Name</Label>
                <Input
                  id="edit-name"
                  value={editingSkill.name}
                  onChange={(e) => setEditingSkill({ ...editingSkill, name: e.target.value })}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-description">Description</Label>
                <Input
                  id="edit-description"
                  value={editingSkill.description || ""}
                  onChange={(e) => setEditingSkill({ ...editingSkill, description: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="edit-trigger">Trigger</Label>
                  <Input
                    id="edit-trigger"
                    value={editingSkill.trigger}
                    onChange={(e) => setEditingSkill({ ...editingSkill, trigger: e.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="edit-domain">Domain</Label>
                  <Select value={(editingSkill.domain || "") as string} onValueChange={(v) => setEditingSkill({ ...editingSkill, domain: v as string })}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {domains.map(d => (
                        <SelectItem key={d} value={d}>{d}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditingSkill(null)}>Cancel</Button>
            <Button onClick={handleUpdate} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!testingSkill} onOpenChange={() => setTestingSkill(null)}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Test Skill</DialogTitle>
            <DialogDescription>
              {testingSkill?.name} - Enter input to test the skill
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="test-input">Input</Label>
              <Textarea
                id="test-input"
                placeholder="Enter test input..."
                value={testInput}
                onChange={(e) => setTestInput(e.target.value)}
                rows={4}
              />
            </div>
            <Button onClick={handleTest} disabled={isTesting || !testInput.trim()}>
              {isTesting ? "Testing..." : "Run Skill"}
            </Button>
            {testResult && (
              <div className="grid gap-2">
                <Label>Result</Label>
                <pre className="p-4 bg-muted rounded-lg text-sm overflow-auto max-h-60">
                  {testResult}
                </pre>
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}