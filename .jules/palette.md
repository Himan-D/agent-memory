## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.

## 2024-06-15 - Missing ARIA labels on Icon-Only Buttons
**Learning:** Found a recurring pattern across the dashboard where `<Button variant="outline" size="icon">` used with icon components like `<RefreshCw />` lacked `aria-label` attributes. This breaks keyboard/screen reader accessibility as the buttons have no readable text.
**Action:** When implementing new icon-only buttons, always include a descriptive `aria-label` (e.g., `aria-label="Refresh data"`).
