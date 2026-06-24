## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2024-06-16 - Missing aria-labels on icon-only buttons
**Learning:** Found a common pattern of missing `aria-label` attributes on icon-only buttons (using `size="icon"` or just containing Lucide icons) across various dashboard components like tables and dialogs (e.g., `api-keys/page.tsx`). This causes critical accessibility issues for screen reader users as the button intent is not communicated.
**Action:** When reviewing or adding icon-only buttons using shadcn's `<Button variant="..." size="icon">` or similar constructs, strictly ensure an `aria-label` is provided describing the action (e.g., "Copy API Key", "More options", "Show API Key").
