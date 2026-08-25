## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.

## 2026-06-08 - Added descriptive aria-labels to abbreviated text controls
**Learning:** Ambiguous controls with abbreviated text (e.g., 'AI' vs 'Vec' for search modes) can be confusing for screen reader users. The text content alone doesn't convey the full meaning or that it's a toggleable option.
**Action:** When updating or creating controls with abbreviated text or complex dynamic state, always add a descriptive `aria-label` that clearly explains both the current state and the available actions to ensure the component remains clear to screen reader users.
