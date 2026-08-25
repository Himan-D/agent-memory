## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.
## 2024-06-17 - [Icon-Only Button Accessibility in Agents View]
**Learning:** Icon-only action buttons inside data tables and toolbars (e.g., refresh, more actions) within the dashboard lacked descriptive ARIA labels, creating a challenging experience for screen reader users who rely on context to interpret icon functionality.
**Action:** When utilizing `size="icon"` variations of Shadcn UI `Button` components for standard actions, always supply a clear, concise `aria-label` to provide context parity for assistive technologies.
