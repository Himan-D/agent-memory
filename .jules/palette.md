## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.

## 2024-06-22 - [ARIA Labels on Icon/Ambiguous Buttons]
**Learning:** Found several buttons acting as controls (e.g., search mode toggles, clear search) that used icons or ambiguous text ("AI" vs "Vec") without proper ARIA labels. This pattern occurs commonly in the header component for compact design reasons but makes the interface confusing for screen reader users.
**Action:** When working on complex header or control components, always add descriptive `aria-label`s to buttons where the visual text is either absent (icon-only like clear 'X') or heavily abbreviated/ambiguous (like 'AI'/'Vec' toggles).
