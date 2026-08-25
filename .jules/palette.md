## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2026-07-13 - Added dynamic aria-labels to ambiguous UI controls
**Learning:** When using abbreviated text (like 'AI' vs 'Vec' for search modes) to save screen space, the control's purpose becomes ambiguous to screen reader users if static or missing aria-labels are used. A hardcoded aria-label completely overrides the inner textual content.
**Action:** Always add descriptive `aria-label`s and native HTML `title` tooltips that dynamically clearly explain both the current state and the available actions based on the component's state to ensure clarity for all users.
