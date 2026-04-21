"use client";

import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { api, SearchMode } from "@/lib/api";

export function EnhancedSearchToggle() {
  const [isSpreading, setIsSpreading] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleToggle = async (enabled: boolean) => {
    setLoading(true);
    setIsSpreading(enabled);
    setLoading(false);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Enhanced Search</CardTitle>
        <CardDescription>Use proprietary spreading activation for better multi-hop reasoning</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <div>
            <Label htmlFor="spreading-mode" className="flex flex-col cursor-pointer">
              <span className="font-medium">Spreading Activation</span>
              <span className="text-sm text-muted-foreground font-normal">
                Graph-based retrieval (+23% multi-hop reasoning)
              </span>
            </Label>
          </div>
          <Switch
            id="spreading-mode"
            checked={isSpreading}
            onCheckedChange={handleToggle}
            disabled={loading}
          />
        </div>
      </CardContent>
    </Card>
  );
}

interface SearchResultsProps {
  query: string;
  mode?: string;
  onSearchModeChange?: (mode: string) => void;
}

export function EnhancedSearchResults({ query, mode = "vector", onSearchModeChange }: SearchResultsProps) {
  const [results, setResults] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchMode, setSearchMode] = useState(mode);

  useState(() => {
    setSearchMode(mode);
  });

  const handleSearch = async () => {
    if (!query.trim()) return;
    
    setLoading(true);
    try {
      const searchModeValue = searchMode === "spreading" ? SearchMode.SPREADING : SearchMode.VECTOR;
      const data = await api.compression.searchEnhanced(query, searchModeValue);
      setResults(data.results || []);
    } catch (error) {
      console.error("Enhanced search failed:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Search Results</CardTitle>
        <CardDescription>
          {searchMode === SearchMode.SPREADING ? "Spreading Activation" : "Vector Similarity"}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {results.map((result, index) => (
            <div key={index} className="p-3 rounded-lg border">
              <div className="flex items-center justify-between">
                <span className="font-medium">{result.content}</span>
                <span className="text-sm text-muted-foreground">
                  {Math.round(result.score * 100)}%
                </span>
              </div>
              {result.hops && (
                <div className="text-xs text-muted-foreground mt-1">
                  Graph hops: {result.hops}
                </div>
              )}
            </div>
          ))}
          {results.length === 0 && !loading && (
            <div className="text-center text-muted-foreground">No results</div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}