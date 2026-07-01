import { useTheme } from "next-themes";
import { Moon, Sun, Bell, User } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuth } from "@/contexts/auth-context";

export function Header() {
  const { theme, setTheme } = useTheme();
  const { user } = useAuth();

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-background/95 px-6 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex flex-1 items-center gap-4">
        {/* Search Placeholder */}
        <div className="relative max-w-md flex-1">
          <div className="h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-muted-foreground flex items-center">
            Search placeholder...
          </div>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
          aria-label="Toggle theme"
        >
          <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
          <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          <span className="sr-only">Toggle theme</span>
        </Button>

        <Button variant="ghost" size="icon" className="relative" aria-label="Notifications Placeholder">
          <Bell className="h-5 w-5" />
        </Button>

        <Button variant="ghost" className="relative h-10 w-10 rounded-full" aria-label="User account">
          <Avatar>
            <AvatarFallback>
              {user ? user.name.slice(0, 2).toUpperCase() : <User className="h-5 w-5" />}
            </AvatarFallback>
          </Avatar>
        </Button>
      </div>
    </header>
  );
}
