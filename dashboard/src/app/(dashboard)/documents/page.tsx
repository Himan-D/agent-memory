"use client";

import { useState, useRef } from "react";
import { api } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { FileText, Upload, Loader2, CheckCircle, AlertCircle } from "lucide-react";
import { toast } from "sonner";

interface ExtractionResult {
  content: string;
  title: string;
  mime_type: string;
  source: string;
  metadata: Record<string, string>;
  pages: number;
}

export default function DocumentsPage() {
  const [file, setFile] = useState<File | null>(null);
  const [isExtracting, setIsExtracting] = useState(false);
  const [result, setResult] = useState<ExtractionResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [extractions, setExtractions] = useState<ExtractionResult[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);

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
    } catch (err) {
      const message = err instanceof Error ? err.message : "Extraction failed";
      setError(message);
      toast.error("Extraction failed: " + message);
    } finally {
      setIsExtracting(false);
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

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Upload Document</CardTitle>
            <CardDescription>
              Supported formats: PDF, text, Markdown, and other document formats
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
                    PDF, TXT, Markdown, DOC, CSV
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
                  <div className="mt-1 p-3 bg-muted rounded-lg max-h-64 overflow-y-auto">
                    <p className="text-sm whitespace-pre-wrap line-clamp-20">{result.content}</p>
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
    </div>
  );
}