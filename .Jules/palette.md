## 2026-06-06 - Added ARIA Labels to Icon-Only Buttons
**Learning:** Found several icon-only buttons across the application (like notifications, sidebar toggle, memory actions) lacking `aria-label` attributes, making them opaque to screen readers.
**Action:** Always verify that buttons containing only icons (e.g., from `lucide-react`) have a descriptive `aria-label`. Ensure dynamic states (like expanded/collapsed) update the label accordingly (e.g., `aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}`).
