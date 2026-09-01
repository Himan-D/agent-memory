## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.

## 2026-07-15 - Dynamic Accessible Labels for Abbreviated Toggles
**Learning:** Toggles that use abbreviated text (like 'AI' vs 'Vec') for space constraints lose their meaning for screen reader users if relying solely on the inner text. Adding a static aria-label overrides this dynamic text, so the label itself must dynamically represent the state.
**Action:** Always add descriptive, dynamic `aria-label`s to abbreviated controls that clearly explain both the current state and the available actions.
