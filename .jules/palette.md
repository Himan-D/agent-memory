## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2026-06-07 - Add descriptive aria-labels to ambiguous controls
**Learning:** When using abbreviated text for states (like "AI" or "Vec") in toggle controls, the abbreviation can be confusing for screen reader users, who won't understand what "Vec" means in context.
**Action:** Always add descriptive `aria-label`s that clearly explain both the current state and the available actions to ensure the component remains clear to screen reader users.
