## YYYY-MM-DD - Initializing Palette Journal\n**Learning:** Started keeping track of UX learnings.\n**Action:** Will add insights as I make them.
## 2026-06-07 - Added aria-labels to icon-only buttons
**Learning:** The project relies on shadcn/ui but has missed several standard accessibility practices like adding 'aria-label' or screen reader only text to icon-only buttons. The Header and Memory Table components, as well as the Sidebar component, have icon-only buttons without proper accessible names.
**Action:** Always verify that 'Button' components with 'size="icon"' and regular '<button>' elements without visible text have 'aria-label' attributes or use '<span className="sr-only">' for accessibility.

## 2026-06-28 - Missing aria-labels on Shadcn UI Icon Buttons
**Learning:** Even when using accessible design systems like Shadcn UI, components that utilize `size="icon"` often miss accessible names. Multiple icon-only `<Button>` elements throughout the app (such as API keys view toggles and Entity graph controls) lacked `aria-label` attributes.
**Action:** Consistently verify that any `<Button size="icon">` includes a descriptive `aria-label` attribute, especially when the button's purpose is only conveyed visually via a child SVG icon.
