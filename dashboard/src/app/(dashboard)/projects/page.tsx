"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
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
import { FilterComponent } from "@/components/ui/filter-component";
import { FolderKanban, Plus, Settings, Trash2, Users, RefreshCw, Eye } from "lucide-react";
import { toast } from "sonner";
import { projectsApi, memoriesApi, type Project } from "@/lib/api";

export default function ProjectsPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isViewOpen, setIsViewOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [projectMemories, setProjectMemories] = useState<any[]>([]);
  const [newProject, setNewProject] = useState({ name: "", description: "" });
  const [editProject, setEditProject] = useState({ name: "", description: "" });
  const [isCreating, setIsCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const fetchProjects = useCallback(async () => {
    try {
      setIsLoading(true);
      const response = await projectsApi.list();
      setProjects(response.projects || []);
    } catch (error) {
      console.error("Failed to fetch projects:", error);
      toast.error("Failed to load projects");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const clearFilters = () => {
    setSearchQuery("");
    setDateFrom(null);
    setDateTo(null);
  };

  const filteredProjects = projects.filter((project) => {
    const matchesSearch =
      searchQuery === "" ||
      project.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (project.description?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false);

    const projectDate = new Date(project.created_at || Date.now());
    const matchesFrom = !dateFrom || projectDate >= dateFrom;
    const matchesTo = !dateTo || projectDate <= dateTo;

    return matchesSearch && matchesFrom && matchesTo;
  });

  const handleCreate = async () => {
    if (!newProject.name.trim()) {
      toast.error("Project name is required");
      return;
    }

    try {
      setIsCreating(true);
      const created = await projectsApi.create({
        name: newProject.name,
        description: newProject.description,
      });
      setProjects(prev => [...prev, created]);
      setIsCreateOpen(false);
      setNewProject({ name: "", description: "" });
      toast.success("Project created successfully");
    } catch (error) {
      console.error("Failed to create project:", error);
      toast.error("Failed to create project");
    } finally {
      setIsCreating(false);
    }
  };

  const handleEdit = async () => {
    if (!selectedProject || !editProject.name.trim()) {
      toast.error("Project name is required");
      return;
    }

    try {
      setIsCreating(true);
      await projectsApi.update(selectedProject.id, {
        name: editProject.name,
        description: editProject.description,
      });
      setProjects(prev => prev.map(p => 
        p.id === selectedProject.id ? { ...p, name: editProject.name, description: editProject.description } : p
      ));
      setIsSettingsOpen(false);
      setSelectedProject(null);
      toast.success("Project updated successfully");
    } catch (error) {
      console.error("Failed to update project:", error);
      toast.error("Failed to update project");
    } finally {
      setIsCreating(false);
    }
  };

  const handleView = async (project: Project) => {
    setSelectedProject(project);
    setIsViewOpen(true);
    
    try {
      const response = await memoriesApi.list({ limit: 50 });
      const filtered = (response.memories || []).filter((m: any) => m.project_id === project.id);
      setProjectMemories(filtered);
    } catch (error) {
      console.log("Could not load project memories");
      setProjectMemories([]);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this project?")) return;

    try {
      setDeletingId(id);
      await projectsApi.delete(id);
      setProjects(prev => prev.filter(p => p.id !== id));
      toast.success("Project deleted");
    } catch (error) {
      console.error("Failed to delete project:", error);
      toast.error("Failed to delete project");
    } finally {
      setDeletingId(null);
    }
  };

  const openSettingsDialog = (project: Project) => {
    setSelectedProject(project);
    setEditProject({ name: project.name, description: project.description || "" });
    setIsSettingsOpen(true);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Projects</h1>
          <p className="text-muted-foreground">Organize memories and agents by project</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={fetchProjects}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Create Project
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create New Project</DialogTitle>
                <DialogDescription>Create a new project to organize your work</DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">Project Name</Label>
                  <Input
                    id="name"
                    placeholder="Enter project name..."
                    value={newProject.name}
                    onChange={(e) => setNewProject({ ...newProject, name: e.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="description">Description</Label>
                  <Input
                    id="description"
                    placeholder="Brief description..."
                    value={newProject.description}
                    onChange={(e) => setNewProject({ ...newProject, description: e.target.value })}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                <Button onClick={handleCreate} disabled={isCreating}>
                  {isCreating ? "Creating..." : "Create Project"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search projects..."
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
      ) : filteredProjects.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <FolderKanban className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">No projects found</p>
            {searchQuery && (
              <Button variant="ghost" onClick={clearFilters} className="mt-2">
                Clear filters
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredProjects.map((project) => (
            <Card key={project.id} className="card-hover">
              <CardHeader className="space-y-0 pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-primary/10 p-2">
                      <FolderKanban className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                      <CardTitle className="text-lg">{project.name}</CardTitle>
                      {project.description && (
                        <p className="text-sm text-muted-foreground line-clamp-2">{project.description}</p>
                      )}
                    </div>
                  </div>
                  <Button 
                    variant="ghost" 
                    size="icon"
                    onClick={() => handleDelete(project.id)}
                    disabled={deletingId === project.id}
                  >
                    <Trash2 className="h-4 w-4 text-muted-foreground" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex gap-4 mb-4">
                  <div className="flex items-center gap-1 text-sm text-muted-foreground">
                    <Settings className="h-4 w-4" />
                    <span>{project.memory_count ?? 0} memories</span>
                  </div>
                  <div className="flex items-center gap-1 text-sm text-muted-foreground">
                    <Users className="h-4 w-4" />
                    <span>{project.agent_count ?? 0} agents</span>
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" className="flex-1" onClick={() => handleView(project)}>
                    <Eye className="mr-1 h-4 w-4" />
                    View
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => openSettingsDialog(project)}>
                    <Settings className="h-4 w-4" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={isViewOpen} onOpenChange={setIsViewOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{selectedProject?.name}</DialogTitle>
            <DialogDescription>
              {selectedProject?.description || "No description"}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="p-4 bg-muted rounded-lg">
                <div className="text-sm text-muted-foreground">Memories</div>
                <div className="text-2xl font-bold">{selectedProject?.memory_count ?? 0}</div>
              </div>
              <div className="p-4 bg-muted rounded-lg">
                <div className="text-sm text-muted-foreground">Agents</div>
                <div className="text-2xl font-bold">{selectedProject?.agent_count ?? 0}</div>
              </div>
            </div>
            <div>
              <Label>Recent Memories</Label>
              <div className="mt-2 space-y-2 max-h-60 overflow-y-auto">
                {projectMemories.length > 0 ? (
                  projectMemories.slice(0, 5).map((mem: any) => (
                    <div key={mem.id} className="p-2 border rounded text-sm">
                      {mem.content?.substring(0, 100)}...
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-muted-foreground">No memories found</p>
                )}
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={isSettingsOpen} onOpenChange={setIsSettingsOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Project Settings</DialogTitle>
            <DialogDescription>
              Update project details
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Project Name</Label>
              <Input
                id="edit-name"
                value={editProject.name}
                onChange={(e) => setEditProject({ ...editProject, name: e.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-description">Description</Label>
              <Input
                id="edit-description"
                value={editProject.description}
                onChange={(e) => setEditProject({ ...editProject, description: e.target.value })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsSettingsOpen(false)}>Cancel</Button>
            <Button onClick={handleEdit} disabled={isCreating}>
              {isCreating ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}