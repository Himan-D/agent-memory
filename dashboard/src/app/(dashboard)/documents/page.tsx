"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { api, memoriesApi, sourcesApi } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { FileText, Upload, Loader2, CheckCircle, AlertCircle, Save, Trash2, RefreshCw, Info } from "lucide-react";
import { toast } from "sonner";

interface ExtractionResult {
  content: string;
  title: string;
  mime_type: string;
  source: string;
  metadata: Record<string, string>;
  pages: number;
}

interface Source {
  id: string;
  name: string;
  type: string;
  status: string;
  created_at: string;
}

const SUPPORTED_TYPES = [
  { ext: "PDF", mime: "application/pdf" },
  { ext: "TXT", mime: "text/plain" },
  { ext: "MD", mime: "text/markdown" },
  { ext: "DOCX", mime: "application/vnd.openxmlformats-officedocument.wordprocessingml.document" },
];

export default function DocumentsPage() {
  const [file, setFile] = useState<File | null>(null);
  const [isExtracting, setIsExtracting] = useState(false);
  const [isSavingMemory, setIsSavingMemory] = useState(false);
  const [result, setResult] = useState<ExtractionResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [extractions, setExtractions] = useState<ExtractionResult[]>([]);
  const [sources, setSources] = useState<Source[]>([]);
  const [isLoadingSources, setIsLoadingSources] = useState(false);
  const [deletingSourceId, setDeletingSourceId] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fetchSources = useCallback(async () => {
    try {
      setIsLoadingSources(true);
      const response = await sourcesApi.list();
      setSources(response.sources || []);
    } catch (err) {
      // Sources endpoint may not be available; fail silently
      setSources([]);
    } finally {
      setIsLoadingSources(false);
    }
  }, []);

  useEffect(() => {
    fetchSources();
  }, [fetchSources]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile) {
      setFile(selectedFile);
      setResult(null);
      setError(null);
    }
  };

  const handleExtract = async () => {
    if (!file) return;

    setIsExtracting(true);
    setError(null);
    setResult(null);

    try {
      const data = await api.documents.extract(file);
      setResult(data);
      setExtractions((prev) => [data, ...prev]);
      toast.success("Document extracted successfully");
      // Refresh sources list after extraction
      fetchSources();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Extraction failed";
      setError(message);
      toast.error("Extraction failed: " + message);
    } finally {
      setIsExtracting(false);
    }
  };

  const handleSaveToMemory = async () => {
    if (!result) return;

    setIsSavingMemory(true);
    try {
      await memoriesApi.create({
        content: result.content,
        type: "conversation",
        category: "document",
        tags: ["extracted", file?.name ?? result.source].filter(Boolean),
      });
      toast.success("Saved to memory successfully");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to save";
      toast.error("Failed to save to memory: " + message);
    } finally {
      setIsSavingMemory(false);
    }
  };

  const handleDeleteSource = async (id: string) => {
    if (!confirm("Are you sure you want to delete this source?")) return;

    try {
      setDeletingSourceId(id);
      await sourcesApi.delete(id);
      setSources((prev) => prev.filter((s) => s.id !== id));
      toast.success("Source deleted");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Delete failed";
      toast.error("Failed to delete source: " + message);
    } finally {
      setDeletingSourceId(null);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    const droppedFile = e.dataTransfer.files[0];
    if (droppedFile) {
      setFile(droppedFile);
      setResult(null);
      setError(null);
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Documents</h1>
          <p className="text-muted-foreground">
            Extract memories, entities, and metadata from documents
          </p>
        </div>
      </div>

      {/* Supported file types info */}
      <div className="flex items-start gap-2 p-3 bg-muted rounded-lg text-sm">
        <Info className="h-4 w-4 mt-0.5 shrink-0 text-muted-foreground" />
        <div>
          <span className="font-medium">Supported formats: </span>
          <span className="text-muted-foreground">
            {SUPPORTED_TYPES.map((t) => t.ext).join(", ")}
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Upload Document</CardTitle>
            <CardDescription>
              Supported formats: PDF, TXT, Markdown, DOCX
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div
              className="border-2 border-dashed rounded-lg p-8 text-center hover:border-primary/50 transition-colors cursor-pointer"
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onClick={() => fileInputRef.current?.click()}
            >
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                onChange={handleFileChange}
                accept=".pdf,.txt,.md,.doc,.docx,.csv"
              />
              {file ? (
                <div className="flex flex-col items-center gap-2">
                  <FileText className="w-12 h-12 text-primary" />
                  <p className="font-medium">{file.name}</p>
                  <p className="text-sm text-muted-foreground">
                    {(file.size / 1024).toFixed(1)} KB
                  </p>
                </div>
              ) : (
                <div className="flex flex-col items-center gap-2">
                  <Upload className="w-12 h-12 text-muted-foreground" />
                  <p className="font-medium">Drop a file here or click to browse</p>
                  <p className="text-sm text-muted-foreground">
                    PDF, TXT, Markdown, DOCX
                  </p>
                </div>
              )}
            </div>

            {file && (
              <Button onClick={handleExtract} disabled={isExtracting} className="w-full">
                {isExtracting ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Extracting...
                  </>
                ) : (
                  <>
                    <FileText className="w-4 h-4 mr-2" />
                    Extract from {file.name}
                  </>
                )}
              </Button>
            )}

            {error && (
              <div className="p-3 bg-red-50 text-red-600 rounded-lg text-sm flex items-start gap-2">
                <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
                {error}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Extraction Result</CardTitle>
            <CardDescription>
              Extracted content and metadata from the document
            </CardDescription>
          </CardHeader>
          <CardContent>
            {result ? (
              <div className="space-y-4">
                <div className="flex items-center gap-2 text-green-600">
                  <CheckCircle className="w-4 h-4" />
                  <span className="font-medium">Extraction successful</span>
                </div>

                {result.title && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Title</p>
                    <p className="text-sm">{result.title}</p>
                  </div>
                )}

                <div>
                  <p className="text-sm font-medium text-muted-foreground">Content</p>
                  <div className="mt-1 p-3 bg-muted rounded-lg max-h-48 overflow-y-auto">
                    <p className="text-sm whitespace-pre-wrap">{result.content}</p>
                  </div>
                </div>

                <div className="flex flex-wrap gap-2">
                  {result.pages > 0 && (
                    <Badge variant="outline">{result.pages} pages</Badge>
                  )}
                  {result.mime_type && (
                    <Badge variant="outline">{result.mime_type}</Badge>
                  )}
                  {result.source && (
                    <Badge variant="outline">{result.source}</Badge>
                  )}
                </div>

                {Object.keys(result.metadata).length > 0 && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-2">Metadata</p>
                    <div className="grid grid-cols-2 gap-2">
                      {Object.entries(result.metadata).map(([key, value]) => (
                        <div key={key} className="text-xs p-2 bg-muted rounded">
                          <span className="font-medium">{key}:</span> {value}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                <Button
                  onClick={handleSaveToMemory}
                  disabled={isSavingMemory}
                  className="w-full"
                  variant="outline"
                >
                  {isSavingMemory ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      Saving to Memory...
                    </>
                  ) : (
                    <>
                      <Save className="w-4 h-4 mr-2" />
                      Save to Memory
                    </>
                  )}
                </Button>
              </div>
            ) : extractions.length > 0 ? (
              <div className="space-y-4">
                <p className="text-sm text-muted-foreground">Previous extractions</p>
                {extractions.slice(0, 5).map((ext, i) => (
                  <div key={i} className="p-3 border rounded-lg">
                    <div className="flex items-center justify-between">
                      <p className="font-medium text-sm">{ext.title || ext.source}</p>
                      {ext.pages > 0 && (
                        <Badge variant="outline" className="text-xs">{ext.pages} pages</Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{ext.content}</p>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center text-muted-foreground py-12">
                <FileText className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p>Upload a document and click Extract</p>
                <p className="text-sm mt-1">Results will appear here</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Ingested Sources */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle>Ingested Sources</CardTitle>
            <CardDescription>Previously processed documents and sources</CardDescription>
          </div>
          <Button variant="outline" size="icon" aria-label="Refresh sources" onClick={fetchSources} disabled={isLoadingSources}>
            <RefreshCw className={`h-4 w-4 ${isLoadingSources ? "animate-spin" : ""}`} />
          </Button>
        </CardHeader>
        <CardContent>
          {isLoadingSources ? (
            <div className="space-y-2">
              {[1, 2, 3].map(i => <Skeleton key={i} className="h-12 w-full" />)}
            </div>
          ) : sources.length > 0 ? (
            <div className="space-y-2">
              {sources.map((source) => (
                <div key={source.id} className="flex items-center justify-between p-3 border rounded-lg">
                  <div className="flex items-center gap-3 min-w-0">
                    <FileText className="h-4 w-4 shrink-0 text-muted-foreground" />
                    <div className="min-w-0">
                      <p className="font-medium text-sm truncate">{source.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(source.created_at).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0 ml-2">
                    {source.type && (
                      <Badge variant="outline" className="text-xs uppercase">{source.type}</Badge>
                    )}
                    {source.status && (
                      <Badge
                        variant={source.status === "processed" ? "default" : "secondary"}
                        className="text-xs capitalize"
                      >
                        {source.status}
                      </Badge>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label="Delete source"
                      className="h-8 w-8"
                      onClick={() => handleDeleteSource(source.id)}
                      disabled={deletingSourceId === source.id}
                    >
                      <Trash2 className="h-4 w-4 text-muted-foreground" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <FileText className="h-8 w-8 text-muted-foreground mb-2 opacity-50" />
              <p className="text-sm text-muted-foreground">No ingested sources yet</p>
              <p className="text-xs text-muted-foreground mt-1">
                Extracted documents will appear here
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
