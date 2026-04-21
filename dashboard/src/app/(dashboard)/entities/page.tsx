"use client";

import { useState, useCallback, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { entitiesApi, type Entity } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
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
import { CircleDot, Trash2, Edit, Eye, ZoomIn, ZoomOut, RotateCcw, Network, RefreshCw, Plus, MoreHorizontal } from "lucide-react";
import { toast } from "sonner";
import dynamic from "next/dynamic";

const ForceGraph2D = dynamic(() => import("react-force-graph-2d"), {
  ssr: false,
  loading: () => (
    <div className="flex h-[500px] items-center justify-center">
      <Skeleton className="h-[500px] w-full" />
    </div>
  ),
});

const typeColors: Record<string, string> = {
  project: "#3B82F6",
  team: "#10B981",
  service: "#8B5CF6",
  database: "#F59E0B",
  goal: "#EF4444",
  customer: "#EC4899",
  user: "#6366F1",
  agent: "#14B8A6",
};

const entityTypes = ["user", "agent", "project", "team", "service", "database", "goal", "customer"];

export default function EntitiesPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isViewOpen, setIsViewOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [selectedEntity, setSelectedEntity] = useState<Entity | null>(null);
  const [viewMode, setViewMode] = useState<"table" | "graph">("table");
  const [newEntity, setNewEntity] = useState({
    name: "",
    type: "user",
    properties: {} as Record<string, unknown>,
  });
  const [editEntity, setEditEntity] = useState({
    name: "",
    type: "user",
    properties: {} as Record<string, unknown>,
  });

  const graphRef = useRef<any>(null);
  const queryClient = useQueryClient();

  const { data: entitiesData, isLoading, refetch } = useQuery({
    queryKey: ["entities"],
    queryFn: () => entitiesApi.list({ limit: 100 }),
  });

  const { data: relationsData } = useQuery({
    queryKey: ["relations-all"],
    queryFn: async () => {
      const allRelations: { id: string; from_id: string; to_id: string; type: string }[] = [];
      if (!entitiesData?.entities) return allRelations;
      
      for (const entity of entitiesData.entities) {
        try {
          const rels = await entitiesApi.getRelations(entity.id);
          if (rels?.relations) {
            allRelations.push(...rels.relations);
          }
        } catch (e) {}
      }
      return allRelations;
    },
    enabled: !!entitiesData?.entities?.length,
  });

  const graphNodes = entitiesData?.entities?.map((entity) => ({
    id: entity.id,
    name: entity.name,
    type: entity.type,
    val: 30,
    color: typeColors[entity.type] || "#666666",
  })) || [];

  const graphLinks = (relationsData || []).map((rel: any) => ({
    source: rel.from_id,
    target: rel.to_id,
    relation: rel.type,
  }));

  const createMutation = useMutation({
    mutationFn: async (data: Partial<Entity>) => {
      return entitiesApi.create(data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["entities"] });
      setIsCreateOpen(false);
      setNewEntity({ name: "", type: "user", properties: {} });
      toast.success("Entity created successfully");
    },
    onError: (err) => {
      toast.error(`Failed to create entity: ${err}`);
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: string; data: Partial<Entity> }) => {
      return entitiesApi.update(id, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["entities"] });
      setIsEditOpen(false);
      setSelectedEntity(null);
      toast.success("Entity updated successfully");
    },
    onError: (err) => {
      toast.error(`Failed to update entity: ${err}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      return entitiesApi.delete(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["entities"] });
      toast.success("Entity deleted");
    },
    onError: (err) => {
      toast.error(`Failed to delete entity: ${err}`);
    },
  });

  const clearFilters = () => {
    setSearchQuery("");
    setTypeFilter("all");
    setDateFrom(null);
    setDateTo(null);
  };

  const filteredEntities = entitiesData?.entities?.filter((entity) => {
    const matchesSearch =
      searchQuery === "" ||
      entity.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      entity.type.toLowerCase().includes(searchQuery.toLowerCase());

    const matchesType = typeFilter === "all" || entity.type === typeFilter;

    const entityDate = new Date(entity.created_at);
    const matchesFrom = !dateFrom || entityDate >= dateFrom;
    const matchesTo = !dateTo || entityDate <= dateTo;

    return matchesSearch && matchesType && matchesFrom && matchesTo;
  });

  const handleNodeClick = useCallback((node: any) => {
    const entity = entitiesData?.entities?.find((e) => e.id === node.id);
    if (entity) {
      setSelectedEntity(entity);
      setIsViewOpen(true);
    }
  }, [entitiesData]);

  const handleEdit = (entity: Entity) => {
    setSelectedEntity(entity);
    setEditEntity({
      name: entity.name,
      type: entity.type,
      properties: entity.properties || {},
    });
    setIsEditOpen(true);
  };

  const handleUpdate = () => {
    if (!selectedEntity) return;
    updateMutation.mutate({
      id: selectedEntity.id,
      data: {
        name: editEntity.name,
        type: editEntity.type,
        properties: editEntity.properties,
      },
    });
  };

  const handleZoomIn = () => {
    const currentZoom = graphRef.current?.zoom() || 1;
    graphRef.current?.zoom(currentZoom * 1.5, 400);
  };

  const handleZoomOut = () => {
    const currentZoom = graphRef.current?.zoom() || 1;
    graphRef.current?.zoom(currentZoom / 1.5, 400);
  };

  const handleReset = () => {
    graphRef.current?.zoomToFit(400);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Entities</h1>
          <p className="text-muted-foreground">
            {entitiesData?.entities?.length ? `${entitiesData.entities.length} entities in knowledge graph` : "Manage knowledge graph nodes"}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <div className="flex rounded-lg border p-1">
            <Button
              variant={viewMode === "table" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setViewMode("table")}
            >
              Table
            </Button>
            <Button
              variant={viewMode === "graph" ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setViewMode("graph")}
            >
              <Network className="mr-2 h-4 w-4" />
              Graph
            </Button>
          </div>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Create Entity
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[500px]">
              <DialogHeader>
                <DialogTitle>Create New Entity</DialogTitle>
                <DialogDescription>
                  Add a new node to your knowledge graph
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">Name</Label>
                  <Input
                    id="name"
                    placeholder="Enter entity name..."
                    value={newEntity.name}
                    onChange={(e) =>
                      setNewEntity({ ...newEntity, name: e.target.value })
                    }
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="type">Type</Label>
                  <Select
                    value={newEntity.type}
                    onValueChange={(value) =>
                      setNewEntity({ ...newEntity, type: value || "user" })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {entityTypes.map(type => (
                        <SelectItem key={type} value={type}>{type}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="properties">Properties (JSON)</Label>
                  <Textarea
                    id="properties"
                    placeholder='{"key": "value"}'
                    onChange={(e) => {
                      try {
                        setNewEntity({ 
                          ...newEntity, 
                          properties: JSON.parse(e.target.value || "{}") 
                        });
                      } catch {
                        // Invalid JSON, ignore
                      }
                    }}
                    className="min-h-[80px]"
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                  Cancel
                </Button>
                <Button
                  onClick={() => createMutation.mutate(newEntity)}
                  disabled={!newEntity.name.trim() || createMutation.isPending}
                >
                  {createMutation.isPending ? "Creating..." : "Create Entity"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search entities..."
        typeValue={typeFilter}
        onTypeChange={setTypeFilter}
        typeOptions={[
          { label: "All Types", value: "all" },
          { label: "User", value: "user" },
          { label: "Agent", value: "agent" },
          { label: "Project", value: "project" },
          { label: "Team", value: "team" },
          { label: "Service", value: "service" },
          { label: "Database", value: "database" },
          { label: "Goal", value: "goal" },
          { label: "Customer", value: "customer" },
        ]}
        dateFrom={dateFrom}
        onDateFromChange={setDateFrom}
        dateTo={dateTo}
        onDateToChange={setDateTo}
        onClear={clearFilters}
      />

      {viewMode === "table" ? (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="w-[50px]"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell><Skeleton className="h-4 w-full" /></TableCell>
                      <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                      <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                      <TableCell><Skeleton className="h-8 w-8" /></TableCell>
                    </TableRow>
                  ))
                ) : filteredEntities?.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8">
                      <CircleDot className="mx-auto h-12 w-12 text-muted-foreground/50" />
                      <p className="mt-2 font-medium">No entities found</p>
                      <p className="text-sm text-muted-foreground">
                        {searchQuery ? "Try a different search" : "Create your first entity"}
                      </p>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredEntities?.map((entity) => (
                    <TableRow key={entity.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <div
                            className="h-3 w-3 rounded-full"
                            style={{ backgroundColor: typeColors[entity.type] || "#666666" }}
                          />
                          <span className="font-medium">{entity.name}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="capitalize">
                          {entity.type}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDateTime(entity.created_at)}
                      </TableCell>
                      <TableCell>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => {
                              setSelectedEntity(entity);
                              setIsViewOpen(true);
                            }}>
                              <Eye className="mr-2 h-4 w-4" />
                              View
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleEdit(entity)}>
                              <Edit className="mr-2 h-4 w-4" />
                              Edit
                            </DropdownMenuItem>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              className="text-destructive"
                              onClick={() => deleteMutation.mutate(entity.id)}
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
      ) : (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-lg font-semibold">Knowledge Graph</CardTitle>
            <div className="flex gap-1">
              <Button variant="outline" size="icon" onClick={handleZoomIn}>
                <ZoomIn className="h-4 w-4" />
              </Button>
              <Button variant="outline" size="icon" onClick={handleZoomOut}>
                <ZoomOut className="h-4 w-4" />
              </Button>
              <Button variant="outline" size="icon" onClick={handleReset}>
                <RotateCcw className="h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="relative h-[500px] overflow-hidden rounded-lg border bg-accent/50">
              {graphNodes.length > 0 ? (
                <ForceGraph2D
                  ref={graphRef}
                  graphData={{ nodes: graphNodes, links: graphLinks }}
                  nodeLabel="name"
                  nodeColor="color"
                  nodeRelSize={8}
                  linkColor={() => "var(--border)"}
                  linkWidth={2}
                  linkDirectionalArrowLength={6}
                  linkDirectionalArrowRelPos={1}
                  onNodeClick={handleNodeClick}
                  backgroundColor="transparent"
                  warmupTicks={50}
                  cooldownTicks={100}
                />
              ) : (
                <div className="flex h-full items-center justify-center text-muted-foreground">
                  <div className="text-center">
                    <Network className="mx-auto h-12 w-12 opacity-50" />
                    <p className="mt-2">No entities to visualize</p>
                  </div>
                </div>
              )}
              <div className="absolute bottom-4 left-4 rounded-lg border bg-background/95 p-3 backdrop-blur">
                <p className="mb-2 text-xs font-medium text-muted-foreground">Entity Types</p>
                <div className="space-y-1">
                  {Object.entries(typeColors).map(([type, color]) => (
                    <div key={type} className="flex items-center gap-2">
                      <div className="h-2 w-2 rounded-full" style={{ backgroundColor: color }} />
                      <span className="text-xs capitalize">{type}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* View Entity Dialog */}
      <Dialog open={isViewOpen} onOpenChange={setIsViewOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Entity Details</DialogTitle>
          </DialogHeader>
          {selectedEntity && (
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <div
                  className="h-4 w-4 rounded-full"
                  style={{ backgroundColor: typeColors[selectedEntity.type] || "#666666" }}
                />
                <div>
                  <p className="font-semibold text-lg">{selectedEntity.name}</p>
                  <Badge variant="outline" className="capitalize mt-1">
                    {selectedEntity.type}
                  </Badge>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-muted-foreground">Created</Label>
                  <p className="mt-1">{formatDateTime(selectedEntity.created_at)}</p>
                </div>
                <div>
                  <Label className="text-muted-foreground">Updated</Label>
                  <p className="mt-1">{formatDateTime(selectedEntity.updated_at)}</p>
                </div>
              </div>
              <div>
                <Label className="text-muted-foreground">Entity ID</Label>
                <p className="mt-1 font-mono text-sm break-all">{selectedEntity.id}</p>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Edit Entity Dialog */}
      <Dialog open={isEditOpen} onOpenChange={(open) => {
        setIsEditOpen(open);
        if (!open) setSelectedEntity(null);
      }}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Edit Entity</DialogTitle>
            <DialogDescription>Update entity information</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input
                id="edit-name"
                value={editEntity.name}
                onChange={(e) => setEditEntity({ ...editEntity, name: e.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-type">Type</Label>
              <Select
                value={editEntity.type}
                onValueChange={(value) => setEditEntity({ ...editEntity, type: value || "user" })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {entityTypes.map(type => (
                    <SelectItem key={type} value={type}>{type}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-properties">Properties (JSON)</Label>
              <Textarea
                id="edit-properties"
                value={JSON.stringify(editEntity.properties, null, 2)}
                onChange={(e) => {
                  try {
                    setEditEntity({ 
                      ...editEntity, 
                      properties: JSON.parse(e.target.value || "{}") 
                    });
                  } catch {
                    // Invalid JSON, ignore
                  }
                }}
                className="min-h-[80px]"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsEditOpen(false)}>Cancel</Button>
            <Button onClick={handleUpdate} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}