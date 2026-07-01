"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Search, Zap, Database, Loader2 } from "lucide-react";

export default function SearchPage() {
  const [query, setQuery] = useState("");
  const [mode, setMode] = useState<string>("vector");
  const [searchResults, setSearchResults] = useState<any[]>([]);
  const [hasSearched, setHasSearched] = useState(false);

  const { isLoading, refetch } = useQuery({
    queryKey: ["search", query, mode],
    queryFn: async () => {
      const data = await api.search.enhanced(query, mode);
      setSearchResults(data.results || []);
      setHasSearched(true);
      return data;
    },
    enabled: false,
  });

  const handleSearch = async () => {
    if (!query.trim()) return;
    refetch();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Search</h1>
          <p className="text-muted-foreground">
            Semantic search with vector, spreading activation, and hybrid modes
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Search Memories</CardTitle>
          <CardDescription>
            Choose a search mode. Spreading activation traverses the knowledge graph for +23% better multi-hop reasoning.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4">
            <div className="flex-1">
              <Input
                placeholder="Search for memories, entities, or concepts..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              />
            </div>
            <div className="w-48">
              <Select value={mode} onValueChange={(v: string | null) => { if (v) setMode(v); }}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="vector">
                    <span className="flex items-center gap-2"><Database className="w-3 h-3" /> Vector</span>
                  </SelectItem>
                  <SelectItem value="spreading">
                    <span className="flex items-center gap-2"><Zap className="w-3 h-3" /> Spreading</span>
                  </SelectItem>
                  <SelectItem value="hybrid">
                    <span className="flex items-center gap-2"><Search className="w-3 h-3" /> Hybrid</span>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <Button onClick={handleSearch} disabled={isLoading || !query.trim()}>
              {isLoading ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : <Search className="w-4 h-4 mr-2" />}
              Search
            </Button>
          </div>
        </CardContent>
      </Card>

      {isLoading && (
        <Card>
          <CardContent className="p-6">
            <div className="space-y-4">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="flex items-start gap-4">
                  <Skeleton className="h-4 w-12" />
                  <Skeleton className="h-4 flex-1" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {!isLoading && searchResults.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>Results ({searchResults.length})</span>
              <Badge variant="outline">{mode} search</Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="divide-y">
              {searchResults.map((result: any, i: number) => (
                <div key={result.id || i} className="px-6 py-4 hover:bg-muted/50 transition-colors">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm line-clamp-3">{result.content}</p>
                      <div className="flex items-center gap-2 mt-2">
                        {result.type && (
                          <Badge variant="outline" className="text-xs">{result.type}</Badge>
                        )}
                        {result.hops !== undefined && (
                          <Badge variant="secondary" className="text-xs">
                            {result.hops} hop{result.hops !== 1 ? "s" : ""}
                          </Badge>
                        )}
                        {result.entity && (
                          <Badge variant="outline" className="text-xs">{result.entity}</Badge>
                        )}
                      </div>
                    </div>
                    <div className="text-right shrink-0">
                      <span className="text-sm font-medium">{(result.score * 100).toFixed(1)}%</span>
                      <p className="text-xs text-muted-foreground">relevance</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {!isLoading && hasSearched && searchResults.length === 0 && (
        <Card>
          <CardContent className="p-12 text-center">
            <Search className="w-12 h-12 mx-auto mb-4 text-muted-foreground opacity-50" />
            <p className="text-lg font-medium">No results found</p>
            <p className="text-muted-foreground mt-1">Try a different query or search mode</p>
          </CardContent>
        </Card>
      )}

      {!hasSearched && (
        <Card>
          <CardContent className="p-12 text-center">
            <Search className="w-12 h-12 mx-auto mb-4 text-muted-foreground opacity-50" />
            <p className="text-lg font-medium">Search your memories</p>
            <p className="text-muted-foreground mt-1">
              Enter a query and choose a search mode to find relevant memories
            </p>
            <div className="flex justify-center gap-4 mt-6">
              <div className="text-center">
                <Badge variant="outline" className="mb-2"><Database className="w-3 h-3 mr-1" /> Vector</Badge>
                <p className="text-xs text-muted-foreground">Semantic similarity</p>
              </div>
              <div className="text-center">
                <Badge variant="outline" className="mb-2"><Zap className="w-3 h-3 mr-1" /> Spreading</Badge>
                <p className="text-xs text-muted-foreground">Graph traversal</p>
              </div>
              <div className="text-center">
                <Badge variant="outline" className="mb-2"><Search className="w-3 h-3 mr-1" /> Hybrid</Badge>
                <p className="text-xs text-muted-foreground">Combined approach</p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}