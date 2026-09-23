## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2024-05-24 - Dynamic aria-labels for abbreviated toggles
**Learning:** Ambiguous controls like toggles with abbreviated text ('AI' vs 'Vec' for search modes) lack context for screen reader users. Hardcoding static aria-labels overrides dynamic inner text, hiding state.
**Action:** Always add dynamic `aria-label`s to such toggles that clearly explain both the current state (e.g., 'Search mode: Vector Search') and the available action ('Click to change').
