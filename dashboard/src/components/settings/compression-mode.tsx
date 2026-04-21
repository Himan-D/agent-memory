"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { api, CompressionMode } from "@/lib/api";

const MODES = [
  { value: CompressionMode.EXTRACT, label: "Extract", description: "97%+ accuracy, 80-85% reduction", recommended: true },
  { value: CompressionMode.BALANCED, label: "Balanced", description: "95%+ accuracy, 85-90% reduction", recommended: false },
  { value: CompressionMode.AGGRESSIVE, label: "Aggressive", description: "92%+ accuracy, 90-93% reduction", recommended: false },
];

export function CompressionModeSelector() {
  const [mode, setMode] = useState<string>(CompressionMode.EXTRACT);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    async function fetchMode() {
      try {
        const data = await api.compression.getMode();
        setMode(data.mode);
      } catch (error) {
        console.error("Failed to fetch compression mode:", error);
      } finally {
        setLoading(false);
      }
    }

    fetchMode();
  }, []);

  const handleModeChange = async (newMode: string) => {
    setSaving(true);
    try {
      await api.compression.setMode(newMode);
      setMode(newMode);
    } catch (error) {
      console.error("Failed to set compression mode:", error);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Compression Mode</CardTitle>
          <CardDescription>Loading...</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Compression Mode</CardTitle>
        <CardDescription>Configure proprietary memory compression</CardDescription>
      </CardHeader>
      <CardContent>
        <RadioGroup
          value={mode}
          onValueChange={handleModeChange}
          disabled={saving}
          className="space-y-3"
        >
          {MODES.map((m) => (
            <div key={m.value} className="flex items-center space-x-3 space-y-2">
              <RadioGroupItem value={m.value} id={m.value} />
              <Label htmlFor={m.value} className="flex flex-col cursor-pointer">
                <span className="flex items-center gap-2">
                  {m.label}
                  {m.recommended && (
                    <span className="text-xs bg-green-500/20 text-green-600 px-2 py-0.5 rounded">
                      Recommended
                    </span>
                  )}
                </span>
                <span className="text-sm text-muted-foreground font-normal">
                  {m.description}
                </span>
              </Label>
            </div>
          ))}
        </RadioGroup>
      </CardContent>
    </Card>
  );
}