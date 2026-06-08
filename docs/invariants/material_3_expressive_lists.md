# Material 3 Expressive Lists Invariants

## Scope

These invariants apply to the shared `wx.List` and `wx.ListItem` widgets,
expressive list style selection, list item shape and color treatment, and the
existing HTMX, selection, context-menu, and collapsible item behavior rendered by
those widgets.

## Invariants

### Standard Zero-Value Default

Rule: A `wx.List` with no explicit `Style` must render as a standard expressive
list.

Why: Existing call sites use struct literals and must receive the requested new
default without migration.

Enforced in: `ListStyleStandard` being the zero value and `List.IsStyleStandard()`
matching the zero-value style.

### Visual Style Does Not Change Behavior

Rule: Switching between standard and segmented list style must only change visual
treatment, not list semantics or interaction behavior.

Why: The Material spec defines standard and segmented as visual choices.

Enforced in: `ui/widget/list.gohtml` and `ui/widget/list_item.gohtml` through
direct Tailwind classes and template conditionals.

### Position-Aware Item Shape

Rule: List item shape must be derived from the item position within the rendered
list: single, first, middle, or last. Middle/default corners use extra-small 4dp,
first and last outer corners use large 16dp, and single items use large 16dp on
all corners.

Why: M3 Expressive lists rely on grouped, position-aware corners rather than a
single identical radius on every item.

Enforced in: Tailwind structural variants rendered directly on the list widget:
`rounded-xs`, `rounded-t-lg`, `rounded-b-lg`, and `rounded-lg`.

### State-Aware Item Shape

Rule: Expressive interaction states must use the Material list state shapes.
Hover uses medium 12dp, while focus, pressed, checked, and selected states use
large 16dp.

Why: The Material list shape state list changes shape before color in several
states. Leaving state shapes at the default 4dp makes selected and interactive
items visually inconsistent with M3 Expressive lists.

Enforced in: Tailwind state and structural variants rendered directly on the list
widget plus the selected item `data-selected` marker.

### Shape Inheritance

Rule: Any child element that paints a state layer or selected background must
inherit the outer list item shape.

Why: Hover, focus, pressed, checked, and selected states must not square off or
bleed outside the expressive item shape.

Enforced in: direct `rounded-[inherit]` utilities on state-layer containers,
details, and summaries.

### Expressive Color Tokens

Rule: Segmented item containers use `surfaceBright`, standard item containers use
transparent backgrounds, and selected non-radio items use `secondaryContainer`.

Why: These map to the Material Components Android expressive list defaults while
using existing SimpleDMS Material color tokens.

Enforced in: direct Tailwind background utilities on item wrappers.

### Segmented Light-Mode Limitation

Rule: Segmented style is available but must be treated as visually incomplete in
light mode until the theme or composition provides visible separation.

Why: The current light theme maps `surface` and `surfaceBright` to the same token,
so segmented item gaps are visible in dark mode but not in light mode.

Enforced in: the `ListStyleSegmented` code comment and this invariant.

### Selection Applies To The Item

Rule: Selected styling must visually apply to the whole list item, not only the
checkbox, radio button, icon, or label.

Why: Material list selection modes define selected state at the item level.

Enforced in: the non-radio selected wrapper utility and the existing radio
checked-state item container utility.

### Existing Widget Semantics Are Preserved

Rule: Expressive list styling must not remove `role="list"`, `role="listitem"`,
HTMX attributes, page-link behavior, context menus, radio selection behavior, or
collapsible item rendering.

Why: The list widget is shared across navigation, file browsing, inbox, metadata,
management, and dialog flows.

Enforced in: focused template changes that add classes without moving behavior to
new widgets or JavaScript.

### Slot Accessibility

Rule: Leading and trailing slots remain supporting slots, and the content slot
must remain the largest item area.

Why: Material list slots are not automatically accessible. Consistent placement
and adequate target sizing preserve scannability and screen-reader usability.

Enforced in: the existing flex layout where the content area is `flex-grow`, the
existing `min-h-14` item target, and continued use of leading/content/trailing
slots in `ListItem`.

### No List Runtime State

Rule: Expressive list shape and style must not depend on client-side JavaScript.

Why: Lists are server-rendered HTMX fragments and must remain correct after full
page loads, partial swaps, load-more updates, and script failures.

Enforced in: Tailwind structural variants and server-rendered utility classes only.

### Swipe-To-Reveal Is Not Implicit

Rule: Swipe-to-reveal behavior must not be assumed available for `wx.ListItem`.

Why: Material Components Android provides swipe-to-reveal as a separate behavior,
but SimpleDMS has no corresponding web runtime or accessibility implementation in
this pass.

Enforced in: no swipe fields, no reveal layout, and no list-specific JavaScript
added by this change.
