## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2026-07-03 - Dynamic ARIA labels for ambiguous state toggles
**Learning:** When updating ambiguous controls (e.g., toggles with abbreviated text like 'AI' vs 'Vec'), adding descriptive `aria-label`s that dynamically reflect the current state ensures the component remains clear to screen reader users. Hardcoding a static `aria-label` overrides the inner text, hiding state.
**Action:** Always add dynamic `aria-label`s that include both the selected value/state and the control's purpose when dealing with abbreviated or dynamic inner text.
