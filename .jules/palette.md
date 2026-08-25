## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2026-06-07 - Add aria-label to icon buttons on Entities page
**Learning:** The project relies on shadcn/ui but often misses accessibility details like 'aria-label' on icon-only buttons (like in the knowledge graph on Entities page).
**Action:** Consistently verify that 'Button' components with 'size="icon"' and no visible text have descriptive 'aria-label' attributes.
