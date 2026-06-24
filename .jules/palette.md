## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.

## 2024-11-20 - Ensure aria-labels reflect dynamic state for ambiguous toggles
**Learning:** For ambiguous controls such as mode toggles where the inner text is abbreviated (e.g., 'AI' vs 'Vec' for search modes due to screen size constraints), screen readers will just read the abbreviation, leaving visually impaired users confused about what the button does or its current state.
**Action:** When updating ambiguous controls or icon buttons, always add descriptive `aria-label` attributes to ensure the component remains clear to screen reader users, and importantly, ensure the `aria-label` is dynamic and matches the current selected state (e.g., `aria-label={searchMode === "spreading" ? "Search mode: Spreading Activation" : "Search mode: Vector Search"}`).
