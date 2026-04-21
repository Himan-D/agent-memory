"use client";

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Calendar } from "@/components/ui/calendar";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { Search, Filter, X, CalendarIcon } from "lucide-react";
import { format } from "date-fns";

export interface FilterOption {
  label: string;
  value: string;
}

export interface FilterComponentProps {
  searchValue: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder?: string;
  typeValue: string;
  onTypeChange: (value: string) => void;
  typeOptions: FilterOption[];
  statusValue?: string;
  onStatusChange?: (value: string) => void;
  statusOptions?: FilterOption[];
  dateFrom: Date | null;
  onDateFromChange: (date: Date | null) => void;
  dateTo: Date | null;
  onDateToChange: (date: Date | null) => void;
  onClear: () => void;
  className?: string;
}

export function FilterComponent({
  searchValue,
  onSearchChange,
  searchPlaceholder = "Search...",
  typeValue,
  onTypeChange,
  typeOptions,
  statusValue,
  onStatusChange,
  statusOptions,
  dateFrom,
  onDateFromChange,
  dateTo,
  onDateToChange,
  onClear,
  className,
}: FilterComponentProps) {
  const [showFilters, setShowFilters] = useState(false);

  const activeFilterCount =
    (searchValue !== "" ? 1 : 0) +
    (typeValue !== "all" ? 1 : 0) +
    (statusValue && statusValue !== "all" ? 1 : 0) +
    (dateFrom ? 1 : 0) +
    (dateTo ? 1 : 0);

  const hasActiveFilters = activeFilterCount > 0;

  return (
    <div className={cn("space-y-4", className)}>
      <div className="flex items-center gap-4 flex-wrap">
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            placeholder={searchPlaceholder}
            className="pl-10"
            value={searchValue}
            onChange={(e) => onSearchChange(e.target.value)}
          />
        </div>
        <Button
          variant={showFilters ? "secondary" : "outline"}
          onClick={() => setShowFilters(!showFilters)}
        >
          <Filter className="mr-2 h-4 w-4" />
          Filters
          {hasActiveFilters && (
            <Badge
              variant="secondary"
              className="ml-2 h-5 w-5 rounded-full p-0 text-xs flex items-center justify-center"
            >
              {activeFilterCount}
            </Badge>
          )}
        </Button>
        {hasActiveFilters && (
          <Button variant="ghost" onClick={onClear}>
            <X className="mr-2 h-4 w-4" />
            Clear Filters
          </Button>
        )}
      </div>

      {showFilters && (
        <Card className="p-4">
          <div className="flex items-center gap-4 flex-wrap">
            {typeOptions && typeOptions.length > 0 && (
              <div className="space-y-1">
                <Label className="text-xs">Type</Label>
                <Select
                  value={typeValue}
                  onValueChange={(v) => v && onTypeChange(v)}
                >
                  <SelectTrigger className="w-[160px]">
                    <SelectValue placeholder="Filter by type" />
                  </SelectTrigger>
                  <SelectContent>
                    {typeOptions.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            {statusOptions && statusOptions.length > 0 && onStatusChange && (
              <div className="space-y-1">
                <Label className="text-xs">Status</Label>
                <Select
                  value={statusValue || "all"}
                  onValueChange={(v) => v && onStatusChange(v)}
                >
                  <SelectTrigger className="w-[160px]">
                    <SelectValue placeholder="Filter by status" />
                  </SelectTrigger>
                  <SelectContent>
                    {statusOptions.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <div className="space-y-1">
              <Label className="text-xs">From Date</Label>
              <Popover>
                <PopoverTrigger asChild>
                  <button className={cn(
                    "inline-flex items-center justify-center whitespace-nowrap rounded-md border border-input bg-background text-sm ring-offset-background font-normal px-3 py-2 w-[150px] justify-start text-left",
                    !dateFrom && "text-muted-foreground"
                  )}>
                    <CalendarIcon className="mr-2 h-4 w-4" />
                    {dateFrom ? format(dateFrom, "MMM d, yyyy") : "From date"}
                  </button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={dateFrom || undefined}
                    onSelect={(date) => {
                      onDateFromChange(date || null);
                    }}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>

            <div className="space-y-1">
              <Label className="text-xs">To Date</Label>
              <Popover>
                <PopoverTrigger asChild>
                  <button className={cn(
                    "inline-flex items-center justify-center whitespace-nowrap rounded-md border border-input bg-background text-sm ring-offset-background font-normal px-3 py-2 w-[150px] justify-start text-left",
                    !dateTo && "text-muted-foreground"
                  )}>
                    <CalendarIcon className="mr-2 h-4 w-4" />
                    {dateTo ? format(dateTo, "MMM d, yyyy") : "To date"}
                  </button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start">
                  <Calendar
                    mode="single"
                    selected={dateTo || undefined}
                    onSelect={(date) => {
                      onDateToChange(date || null);
                    }}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}