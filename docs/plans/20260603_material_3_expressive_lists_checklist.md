# Material 3 Expressive Lists Checklist

## Material And Current-State Audit

- [x] Read the live M3 list overview page.
- [x] Read the live M3 list specs page.
- [x] Read the live M3 list guidelines page.
- [x] Read the Material Components Android list documentation.
- [x] Review the local Material list reference.
- [x] Inspect current SimpleDMS `wx.List` and `wx.ListItem` widgets.
- [x] Inspect current Tailwind source and generated asset flow.

## Widget Model

- [x] Add `ListStyle` to `wx.List`.
- [x] Make standard style the zero-value default for existing list literals.
- [x] Add an explicit segmented expressive style option.
- [x] Document that segmented styling currently does not produce visible
  segmentation in light mode.
- [x] Add a template helper for choosing direct Tailwind utilities.
- [x] Avoid requiring call-site changes for existing lists.

## Templates And CSS

- [x] Render `wx.List` with direct Tailwind utility classes.
- [x] Render list style utilities from the widget model.
- [x] Add direct shape inheritance utilities to item containers that paint state
  layers or selected radio backgrounds.
- [x] Use Tailwind structural variants for first, middle, last, and single item
  shapes.
- [x] Map default/middle shape to extra-small 4dp (`rounded-xs`).
- [x] Map first/last/single outer shape to large 16dp (`rounded-lg`).
- [x] Map hover state shape to medium 12dp (`rounded-md`).
- [x] Map focus, pressed, checked, and selected shape to large 16dp
  (`rounded-lg`).
- [x] Use `surfaceBright` for segmented item containers.
- [x] Keep standard expressive item containers transparent.
- [x] Keep selected non-radio item containers on `secondaryContainer`.
- [x] Preserve radio item selected styling through the existing checked-state
  selector.
- [x] Preserve inherited shape for state layers, summaries, and item containers.
- [x] Preserve `role="list"` and `role="listitem"` semantics.
- [x] Preserve HTMX attributes, page links, context menus, and collapsible item
  rendering.
- [x] Remove list-specific component CSS classes.

## Out Of Scope

- [x] Do not implement swipe-to-reveal in this pass.
- [x] Do not redesign individual row content.
- [x] Do not add list-specific JavaScript.
- [x] Do not migrate unrelated table, card, or list-detail layouts.

## Verification Commands

- [x] Run `gofmt -w` on changed Go files.
- [x] Run `npx tailwindcss -i ui/uix/web/tailwind.css -o ui/uix/web/assets/tailwind.css`.
- [x] Run a focused Go test/build command.
- [x] Run `go test ./...`.
- [x] Run `git diff --check`.
