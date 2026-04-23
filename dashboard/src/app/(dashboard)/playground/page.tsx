"use client";

import { useState } from "react";
import { 
  FileArchive, 
  Search, 
  Network, 
  Zap, 
  Clock, 
  TrendingUp,
  Play,
  Loader2,
  Copy,
  Check
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import {
  playgroundApi,
  PlaygroundCompressionRequest,
  PlaygroundCompressionResult,
  PlaygroundSearchRequest,
  PlaygroundSearchResult,
  CompressionPlaygroundMode,
  SearchMode,
} from "@/lib/api";

export default function PlaygroundPage() {
  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Playground</h1>
          <p className="text-muted-foreground">
            Test the proprietary compression engine and search algorithms
          </p>
        </div>
        <Badge variant="outline" className="text-green-500 border-green-500">
          <Zap className="w-3 h-3 mr-1" />
          Pro Mode
        </Badge>
      </div>

      <Tabs defaultValue="compression" className="space-y-4">
        <TabsList>
          <TabsTrigger value="compression" className="gap-2">
            <FileArchive className="w-4 h-4" />
            Compression
          </TabsTrigger>
          <TabsTrigger value="search" className="gap-2">
            <Search className="w-4 h-4" />
            Search
          </TabsTrigger>
          <TabsTrigger value="graph" className="gap-2">
            <Network className="w-4 h-4" />
            Graph
          </TabsTrigger>
        </TabsList>

        <TabsContent value="compression">
          <CompressionPlayground />
        </TabsContent>

        <TabsContent value="search">
          <SearchPlayground />
        </TabsContent>

        <TabsContent value="graph">
          <GraphPlayground />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function CompressionPlayground() {
  const [input, setInput] = useState("");
  const [modes, setModes] = useState<string[]>(["extraction", "relational", "radix", "hybrid"]);
  const [showEntities, setShowEntities] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<PlaygroundCompressionResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleTest = async () => {
    if (!input.trim()) return;
    
    setIsLoading(true);
    setError(null);
    setResult(null);

    try {
      const req: PlaygroundCompressionRequest = {
        text: input,
        modes: modes,
        show_entities: showEntities,
      };
      const res = await playgroundApi.testCompression(req);
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setIsLoading(false);
    }
  };

  const bestMode = result?.best_mode || "";

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Input</CardTitle>
          <CardDescription>
            Enter text to compress and compare different compression modes
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Text to compress</Label>
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Enter your text here... (e.g., 'Machine learning is a subset of artificial intelligence...')"
              className="min-h-[200px]"
            />
          </div>

          <div className="space-y-2">
            <Label>Compression Modes</Label>
            <div className="flex flex-wrap gap-2">
              {Object.values(CompressionPlaygroundMode).map((mode) => (
                <Badge
                  key={mode}
                  variant={modes.includes(mode) ? "default" : "outline"}
                  className="cursor-pointer"
                  onClick={() => {
                    setModes(prev => 
                      prev.includes(mode) 
                        ? prev.filter(m => m !== mode)
                        : [...prev, mode]
                    );
                  }}
                >
                  {mode.charAt(0).toUpperCase() + mode.slice(1)}
                </Badge>
              ))}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Switch
              id="showEntities"
              checked={showEntities}
              onCheckedChange={setShowEntities}
            />
            <Label htmlFor="showEntities">Show entities</Label>
          </div>

          <Button onClick={handleTest} disabled={isLoading || !input.trim()} className="w-full">
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Compressing...
              </>
            ) : (
              <>
                <Play className="w-4 h-4 mr-2" />
                Test Compression
              </>
            )}
          </Button>

          {error && (
            <div className="p-3 bg-red-50 text-red-600 rounded-lg text-sm">
              {error}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Results</CardTitle>
          <CardDescription>
            {bestMode && (
              <Badge className="mt-2 bg-green-500">
                Best: {bestMode} ({result?.results?.[bestMode]?.reduction_percent?.toFixed(1)}% reduction)
              </Badge>
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!result && !error && (
            <div className="text-center text-muted-foreground py-12">
              <FileArchive className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Enter text and click Test Compression to see results</p>
            </div>
          )}

          {result && (
            <div className="space-y-4">
              {Object.entries(result.results).map(([mode, data]) => (
                <div key={mode} className="p-4 border rounded-lg">
                  <div className="flex items-center justify-between mb-2">
                    <div className="flex items-center gap-2">
                      <Badge variant={mode === bestMode ? "default" : "outline"}>
                        {mode}
                      </Badge>
                      {mode === bestMode && <Zap className="w-4 h-4 text-yellow-500" />}
                    </div>
                    <div className="flex items-center gap-4 text-sm">
                      <span className="text-green-500">
                        {(data.reduction_percent * 100).toFixed(1)}% reduction
                      </span>
                      <span className="text-muted-foreground flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {data.latency_ms.toFixed(0)}ms
                      </span>
                    </div>
                  </div>
                  <pre className="text-xs bg-muted p-2 rounded overflow-x-auto max-h-[150px]">
                    {data.compressed}
                  </pre>
                  {data.token_savings > 0 && (
                    <p className="text-xs text-muted-foreground mt-2">
                      Saved ~{data.token_savings} tokens
                    </p>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function SearchPlayground() {
  const [query, setQuery] = useState("");
  const [modes, setModes] = useState<string[]>(["vector", "spreading"]);
  const [limit, setLimit] = useState(10);
  const [compareModes, setCompareModes] = useState(true);
  const [showGraph, setShowGraph] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<PlaygroundSearchResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleTest = async () => {
    if (!query.trim()) return;
    
    setIsLoading(true);
    setError(null);
    setResult(null);

    try {
      const req: PlaygroundSearchRequest = {
        query: query,
        modes: modes,
        limit: limit,
        compare_modes: compareModes,
        show_graph: showGraph,
      };
      const res = await playgroundApi.testSearch(req);
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Search Query</CardTitle>
          <CardDescription>
            Test different search modes and compare results
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Search Query</Label>
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Enter your search query..."
            />
          </div>

          <div className="space-y-2">
            <Label>Search Modes</Label>
            <div className="flex flex-wrap gap-2">
              {Object.values(SearchMode).map((mode) => (
                <Badge
                  key={mode}
                  variant={modes.includes(mode) ? "default" : "outline"}
                  className="cursor-pointer"
                  onClick={() => {
                    setModes(prev => 
                      prev.includes(mode) 
                        ? prev.filter(m => m !== mode)
                        : [...prev, mode]
                    );
                  }}
                >
                  {mode.charAt(0).toUpperCase() + mode.slice(1)}
                </Badge>
              ))}
            </div>
          </div>

          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <Switch
                id="compareModes"
                checked={compareModes}
                onCheckedChange={setCompareModes}
              />
              <Label htmlFor="compareModes">Compare modes</Label>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                id="showGraph"
                checked={showGraph}
                onCheckedChange={setShowGraph}
              />
              <Label htmlFor="showGraph">Show graph</Label>
            </div>
          </div>

          <Button onClick={handleTest} disabled={isLoading || !query.trim()} className="w-full">
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Searching...
              </>
            ) : (
              <>
                <Search className="w-4 h-4 mr-2" />
                Test Search
              </>
            )}
          </Button>

          {error && (
            <div className="p-3 bg-red-50 text-red-600 rounded-lg text-sm">
              {error}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Results</CardTitle>
          <CardDescription>
            {result && (
              <div className="flex gap-4 mt-2">
                <span className="flex items-center gap-1">
                  <Clock className="w-3 h-3" />
                  Vector: {result.stats.vector_latency_ms.toFixed(0)}ms
                </span>
                <span className="flex items-center gap-1">
                  <Network className="w-3 h-3" />
                  Spreading: {result.stats.spreading_latency_ms.toFixed(0)}ms
                </span>
                <Badge>{result.stats.total_results} results</Badge>
              </div>
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!result && !error && (
            <div className="text-center text-muted-foreground py-12">
              <Search className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Enter a query and click Test Search</p>
            </div>
          )}

          {result && (
            <div className="space-y-4">
              {result.comparison && (
                <div className="p-4 bg-muted rounded-lg">
                  <h4 className="font-medium mb-2">Comparison</h4>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div>
                      <span className="text-muted-foreground">Overlap:</span>{" "}
                      {result.comparison.overlap_count}
                    </div>
                    <div>
                      <span className="text-muted-foreground">Best:</span>{" "}
                      <Badge>{result.comparison.best_mode}</Badge>
                    </div>
                  </div>
                </div>
              )}

              {Object.entries(result.results).map(([mode, hits]) => (
                <div key={mode} className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">{mode}</Badge>
                    <span className="text-sm text-muted-foreground">
                      {hits.length} results
                    </span>
                  </div>
                  {hits.slice(0, 3).map((hit, i) => (
                    <div key={i} className="p-3 border rounded text-sm">
                      <div className="flex items-center justify-between mb-1">
                        <span className="font-mono text-xs text-muted-foreground">
                          {hit.id?.slice(0, 8)}...
                        </span>
                        <span className="text-green-500">
                          {(hit.score * 100).toFixed(1)}%
                        </span>
                      </div>
                      <p className="line-clamp-2">{hit.content}</p>
                      {hit.hops !== undefined && (
                        <Badge variant="outline" className="mt-1 text-xs">
                          {hit.hops} hop{hit.hops !== 1 ? "s" : ""}
                        </Badge>
                      )}
                    </div>
                  ))}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function GraphPlayground() {
  const [query, setQuery] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [graphData, setGraphData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  const handleGenerate = async () => {
    if (!query.trim()) return;
    
    setIsLoading(true);
    setError(null);

    try {
      const res = await playgroundApi.testSearch({
        query: query,
        modes: ["spreading"],
        limit: 10,
        show_graph: true,
      });
      setGraphData(res.graph);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <Card className="lg:col-span-1">
        <CardHeader>
          <CardTitle>Graph Explorer</CardTitle>
          <CardDescription>
            Visualize knowledge graph from your query
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Enter query to visualize..."
          />
          <Button onClick={handleGenerate} disabled={isLoading || !query.trim()} className="w-full">
            {isLoading ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <>
                <Network className="w-4 h-4 mr-2" />
                Generate Graph
              </>
            )}
          </Button>
        </CardContent>
      </Card>

      <Card className="lg:col-span-2">
        <CardHeader>
          <CardTitle>Visualization</CardTitle>
        </CardHeader>
        <CardContent>
          {!graphData && !error && (
            <div className="text-center text-muted-foreground py-12">
              <Network className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Generate a graph to see visualization</p>
            </div>
          )}

          {graphData && (
            <div className="space-y-4">
              <div className="flex gap-4 text-sm">
                <Badge variant="outline">
                  {graphData.nodes?.length || 0} nodes
                </Badge>
                <Badge variant="outline">
                  {graphData.edges?.length || 0} edges
                </Badge>
              </div>
              
              <div className="border rounded-lg p-4 max-h-[400px] overflow-auto">
                <div className="space-y-2">
                  {graphData.nodes?.map((node: any) => (
                    <div key={node.id} className="flex items-center gap-2 text-sm">
                      <Badge 
                        variant={node.type === "query" ? "default" : "outline"}
                        className="text-xs"
                      >
                        {node.type}
                      </Badge>
                      <span>{node.label}</span>
                      {node.score && (
                        <span className="text-muted-foreground">
                          ({(node.score * 100).toFixed(0)}%)
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              <div className="border rounded-lg p-4 max-h-[200px] overflow-auto">
                <h4 className="text-sm font-medium mb-2">Edges</h4>
                <div className="space-y-1 text-xs font-mono">
                  {graphData.edges?.map((edge: any, i: number) => (
                    <div key={i}>
                      {edge.from} → {edge.to} ({edge.type})
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {error && (
            <div className="p-3 bg-red-50 text-red-600 rounded-lg text-sm">
              {error}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}