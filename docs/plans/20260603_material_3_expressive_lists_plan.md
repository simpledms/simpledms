# Material 3 Expressive Lists Plan

## Outcome

Update the shared `wx.List` and `wx.ListItem` widgets to use the Material 3
Expressive list treatment. The standard expressive list style is the default for
new and existing lists, while segmented remains available as an explicit visual
alternative.

## Material Guidance Read

Sources:

- https://m3.material.io/components/lists/overview
- https://m3.material.io/components/lists/specs
- https://m3.material.io/components/lists/guidelines
- Material Components Android list documentation
- Local Material skill reference: `reference/components/List.md`

Key guidance applied to this plan:

- M3 Expressive adds a new expressive list variant and recommends it for new
  designs. Baseline lists remain available but do not include the newest visual
  style, selection treatment, or slot functionality.
- Expressive lists have two visual styles: `standard` and `segmented`. The style
  choice must not change list behavior.
- Segmented lists use filled item containers and gaps to define a contained list
  group. Dividers should be limited to uncontained or complex lists.
- Expressive list items use position-aware shapes: middle/default corners use
  extra-small 4dp, first and last outer corners use large 16dp, and single items
  use large 16dp on all corners. Child content that paints state or selection
  must inherit the same shape.
- Expressive list interaction states change shape: hover uses medium 12dp, while
  focus, pressed, checked, and selected states use large 16dp.
- Standard expressive list items use transparent item containers. Segmented list
  items use `surfaceBright`. Selected list items use `secondaryContainer` and
  selected foreground colors.
- Segmented styling currently does not produce visible segmentation in light mode
  because this theme maps `surface` and `surfaceBright` to the same light token.
- List items need a minimum 48dp target. Current SimpleDMS list items already use
  `min-h-14`, which is larger than the minimum.
- Leading and trailing slots must stay narrower than the content slot. The
  content slot remains the largest slot and owns label/supporting text.
- Selection lists should expose one selection interaction per item. The selected
  state applies to the whole list item.
- Lists should stay easy to scan: keep label/supporting text placement consistent
  and avoid varying item anatomy within one list group.

## Current SimpleDMS Context

- `ui/widget/list.go` is a minimal container with children and HTMX attributes.
- `ui/widget/list.gohtml` renders a `role="list"` flex column with fixed padding
  and a `gap-y-1` baseline spacing.
- `ui/widget/list_item.gohtml` renders `role="listitem"` wrappers and keeps HTMX
  activation, radio handling, selected state, context menus, and collapsible
  children inside the item template.
- Existing call sites construct `&wx.List{...}` directly, so the default behavior
  must work from the zero value without requiring call-site migration.
- Tailwind source lives in `ui/uix/web/tailwind.css`; the generated runtime asset
  is `ui/uix/web/assets/tailwind.css`.

## Goals

- Add a widget-level list style model with standard as the zero-value default.
- Render all existing lists with standard expressive styling unless a caller
  explicitly chooses segmented style.
- Add position-aware expressive list shapes for single, first, middle, and last
  items using Tailwind structural variants so call sites do not need to know item
  positions.
- Use Material color tokens already available in Tailwind: `surfaceBright` for
  segmented item containers and `secondaryContainer` for selected items.
- Preserve existing list semantics, HTMX attributes, page links, context menus,
  radio behavior, collapsible items, and selected-state behavior.
- Keep the implementation small and centralized in the shared list widgets and
  direct Tailwind classes in their templates.

## Non-Goals

- Do not implement Android `SwipeableListItem` or swipe-to-reveal behavior in this
  pass.
- Do not redesign individual list row content, file rows, inbox rows, metadata
  rows, or management lists.
- Do not convert lists to cards, carousels, tables, or list-detail layouts.
- Do not add JavaScript runtime state for list positioning; Tailwind structural
  variants are sufficient for this pass.
- Do not remove the existing baseline-compatible behavior from callers that can
  explicitly choose standard styling.

## Implementation Plan

1. Add `ListStyle` to `wx.List` with `ListStyleStandard` as the zero value and
   `ListStyleSegmented` as the explicit alternative.
2. Add a small style predicate so templates can choose direct Tailwind utilities.
3. Update `List` rendering to include spacing, position-aware shape, segmented
   background, and standard background utilities directly in the template.
4. Update `ListItem` rendering so state-layer containers, details, and summaries
   inherit the outer item shape with direct Tailwind utilities.
5. Keep selected background treatment on the selected item with direct Tailwind
   utilities.
6. Use Tailwind structural variants for first, middle, last, single, hover,
   focus, pressed, checked, and selected item shapes so load-more HTMX swaps and
   dynamic list lengths keep working without server-side position metadata.
7. Rebuild the generated Tailwind asset after source CSS and template changes.
8. Run formatting and focused verification.

## Verification Plan

- Run `gofmt -w` on changed Go files.
- Rebuild Tailwind with:

```bash
npx tailwindcss -i ui/uix/web/tailwind.css \
	-o ui/uix/web/assets/tailwind.css
```
- Run `go test ./ui/widget` if the package has tests; otherwise run a focused Go
  test/build command that covers template compilation.
- Run `git diff --check`.
