## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2026-06-25 - Added dynamic aria-label to search mode toggle
**Learning:** When using controls with abbreviated or cryptic text (like "AI" vs "Vec" for search modes), it's crucial to provide a full, descriptive `aria-label`. If the label is dynamic based on state, make sure the `aria-label` also updates dynamically so screen reader users always have clear context of the current state and what the button does.
**Action:** Always add descriptive `aria-label`s to ambiguous controls, especially when they use abbreviated text. Ensure the `aria-label` dynamically updates when the inner text changes based on state.
