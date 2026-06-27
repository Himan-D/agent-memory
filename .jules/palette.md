## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2026-06-27 - Clarified ambiguous abbreviated text controls
**Learning:** UI controls often use severe abbreviations (like "AI" vs "Vec") to fit in constrained spaces. These can be entirely meaningless to screen reader users who lack visual context.
**Action:** Always verify that controls using abbreviated text include a descriptive `aria-label` that clearly explains both current state and available actions (e.g., "Current mode: Vector Search. Click to switch to Spreading Activation").
