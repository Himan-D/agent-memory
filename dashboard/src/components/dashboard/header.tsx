"use client";

import { useState, useRef, useEffect } from "react";
import { useTheme } from "next-themes";
import { Moon, Sun, Search, Bell, Check, Zap, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { useSession, signOut } from "next-auth/react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useNotifications } from "@/contexts/notification-context";
import { cn } from "@/lib/utils";
import { alertsApi, SearchMode, EnhancedSearchResult } from "@/lib/api";

export function Header() {
  const { theme, setTheme } = useTheme();
  const { data: session, status } = useSession();
  const { notifications, unreadCount, markAsRead, markAllAsRead } = useNotifications();
  const [isOpen, setIsOpen] = useState(false);
  const [mounted, setMounted] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchMode, setSearchMode] = useState<"vector" | "spreading">("vector");
  const [searchResults, setSearchResults] = useState<EnhancedSearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showResults, setShowResults] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const searchTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  useState(() => {
    setMounted(true);
  });

  useEffect(() => {
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }

    if (!searchQuery.trim()) {
      setSearchResults([]);
      setShowResults(false);
      return;
    }

    searchTimeoutRef.current = setTimeout(async () => {
      setIsSearching(true);
      try {
        const response = await alertsApi.compression.searchEnhanced(searchQuery, searchMode);
        setSearchResults(response.results || []);
        setShowResults(true);
      } catch (error) {
        console.error("Search failed:", error);
        setSearchResults([]);
      } finally {
        setIsSearching(false);
      }
    }, 300);

    return () => {
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
    };
  }, [searchQuery, searchMode]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (searchInputRef.current && !searchInputRef.current.contains(event.target as Node)) {
        setTimeout(() => setShowResults(false), 200);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const getTypeStyles = (type: string) => {
    switch (type) {
      case "success":
        return "bg-green-500/10 text-green-500 border-green-500/20";
      case "warning":
        return "bg-yellow-500/10 text-yellow-500 border-yellow-500/20";
      case "error":
        return "bg-red-500/10 text-red-500 border-red-500/20";
      default:
        return "bg-blue-500/10 text-blue-500 border-blue-500/20";
    }
  };

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-background/95 px-6 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex flex-1 items-center gap-4">
        <div className="relative max-w-md flex-1" ref={searchInputRef}>
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            placeholder="Search memories, entities, agents..."
            className="pl-10 pr-20"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onFocus={() => searchResults.length > 0 && setShowResults(true)}
          />
          <div className="absolute right-1 top-1/2 -translate-y-1/2">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant={searchMode === "spreading" ? "default" : "ghost"}
                  size="sm"
                  className={cn(
                    "h-7 px-2 text-xs gap-1",
                    searchMode === "spreading" && "bg-primary/90 hover:bg-primary"
                  )}
                >
                  <Zap className="h-3 w-3" />
                  {searchMode === "spreading" ? "AI" : "Vec"}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setSearchMode("vector")}>
                  <Search className="mr-2 h-3 w-3" />
                  Vector Search
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSearchMode("spreading")}>
                  <Zap className="mr-2 h-3 w-3" />
                  Spreading Activation
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          {showResults && searchResults.length > 0 && (
            <div className="absolute top-full left-0 right-0 mt-2 bg-background border rounded-lg shadow-lg max-h-96 overflow-y-auto z-50">
              <div className="flex items-center justify-between px-3 py-2 border-b text-xs text-muted-foreground">
                <span>{searchResults.length} results ({searchMode === "spreading" ? "Spreading Activation" : "Vector"})</span>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 px-2"
                  onClick={() => {
                    setSearchQuery("");
                    setShowResults(false);
                  }}
                >
                  <X className="h-3 w-3" />
                </Button>
              </div>
              {searchResults.map((result) => (
                <a
                  key={result.id}
                  href={`/memories/${result.id}`}
                  className="block px-3 py-2 hover:bg-muted border-b last:border-0"
                >
                  <p className="text-sm line-clamp-2">{result.content}</p>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="text-xs text-muted-foreground">
                      Score: {(result.score * 100).toFixed(1)}%
                    </span>
                    {result.hops !== undefined && (
                      <Badge variant="outline" className="text-xs">
                        {result.hops} hop{result.hops !== 1 ? "s" : ""}
                      </Badge>
                    )}
                  </div>
                </a>
              ))}
            </div>
          )}
          {showResults && searchQuery && !isSearching && searchResults.length === 0 && (
            <div className="absolute top-full left-0 right-0 mt-2 bg-background border rounded-lg shadow-lg p-4 z-50">
              <p className="text-sm text-muted-foreground text-center">No results found</p>
            </div>
          )}
          {isSearching && (
            <div className="absolute top-full left-0 right-0 mt-2 bg-background border rounded-lg shadow-lg p-4 z-50">
              <p className="text-sm text-muted-foreground text-center">Searching...</p>
            </div>
          )}
        </div>
      </div>

      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
        >
          <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
          <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          <span className="sr-only">Toggle theme</span>
        </Button>

        <Popover open={isOpen} onOpenChange={setIsOpen}>
          <PopoverTrigger asChild>
            <Button variant="ghost" size="icon" className="relative">
              <Bell className="h-5 w-5" />
              {mounted && unreadCount > 0 && (
                <Badge className="absolute -right-1 -top-1 h-5 w-5 rounded-full p-0 text-xs flex items-center justify-center">
                  {unreadCount > 9 ? "9+" : unreadCount}
                </Badge>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-96 p-0" align="end">
            <div className="flex items-center justify-between border-b p-4">
              <h3 className="font-semibold">Notifications</h3>
              {unreadCount > 0 && (
                <Button variant="ghost" size="sm" onClick={markAllAsRead}>
                  <Check className="mr-2 h-4 w-4" />
                  Mark all read
                </Button>
              )}
            </div>
            <div className="max-h-96 overflow-y-auto">
              {notifications.length === 0 ? (
                <div className="p-4 text-center text-sm text-muted-foreground">
                  No notifications yet
                </div>
              ) : (
                notifications.map((notification) => (
                  <div
                    key={notification.id}
                    className={cn(
                      "flex items-start gap-3 border-b p-4 hover:bg-muted/50 transition-colors",
                      notification.status === "unread" && "bg-muted/30"
                    )}
                    onClick={() => notification.status === "unread" && markAsRead(notification.id)}
                  >
                    <div className={cn("mt-1 rounded-full p-2 border", getTypeStyles(notification.type))}>
                      <Bell className="h-3 w-3" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-sm truncate">{notification.title}</p>
                      <p className="text-xs text-muted-foreground line-clamp-2 mt-1">
                        {notification.message}
                      </p>
                      <p className="text-xs text-muted-foreground mt-1">
                        {new Date(notification.created_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                ))
              )}
            </div>
            {notifications.length > 0 && (
              <div className="border-t p-2">
                <a href="/notifications">
                  <Button variant="ghost" size="sm" className="w-full">
                    View all notifications
                  </Button>
                </a>
              </div>
            )}
          </PopoverContent>
        </Popover>

        {status === "authenticated" && session?.user ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="relative h-10 w-10 rounded-full">
                <Avatar>
                  <AvatarImage src={session.user.image || undefined} alt={session.user.name || ""} />
                  <AvatarFallback>
                    {session.user.name
                      ? session.user.name.split(" ").map(n => n[0]).join("").toUpperCase().slice(0, 2)
                      : session.user.email?.[0]?.toUpperCase() || "U"}
                  </AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>
                <div className="flex flex-col space-y-1">
                  <p className="text-sm font-medium">{session.user.name}</p>
                  <p className="text-xs text-muted-foreground">{session.user.email}</p>
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem asChild>
                <a href="/settings">Settings</a>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-destructive cursor-pointer"
                onClick={() => signOut({ callbackUrl: "/auth/signin" })}
              >
                Sign Out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <a href="/auth/signin">
            <Button>Sign In</Button>
          </a>
        )}
      </div>
    </header>
  );
}