# Complete component showcase

This example uses the root component API plus the official `icon/catalog`
resource package. It is the interactive catalog for every public dxui view
constructor. It has searchable categorized Menu navigation, one selected
component detail page, independent navigation/content scrolling, runnable
state and event examples, and source snippets next to each preview.

The root uses `RunResponsive`: windows at least 760 logical units wide show the
sidebar, while narrower windows use top Select/search navigation. The detail
pane stays scrollable and wide previews gain horizontal scrolling.

## Run

From the repository root:

```sh
go run ./examples/components
go run ./examples/components -software
go run ./examples/components -component Textarea
go run ./examples/components -component Popover
go run ./examples/components -component Colors
go run ./examples/components -component Theme
go run ./examples/components -component Input -width 480 -height 420
```

`-software` forces SDL's named software renderer. `-component` chooses the
initial detail page case-insensitively; the default and unknown-value fallback
are both `Box`.
`-duration 2s` is available for a non-interactive native smoke run.
`-width` and `-height` select the initial logical window size, which is useful
for checking the documented 480x420 minimum layout.

The separate Colors page contains the complete 26-family Primitive color
palette. The Theme page shows a Semantic surface/text/accent/hover/danger/
success/border preview that follows the live Light/Dark switch. Its interactive
configurator replaces the global Primary, Surface, Danger, and Success
palettes through `App.SetTheme`. Each setting offers all 26 Primitive color
families; selections survive Light/Dark switches and can be reset together.

The window starts at 1180x760 and has a 480x420 minimum. The square-cornered
navigation column stays 260 logical units wide, while both the filtered
component menus and detail pane remain independently scrollable. Selection,
search text, control values, and event output survive normal declarative root
rebuilds and native expose/resize repaint. A new process can restore a chosen
page with `-component`; the example deliberately does not write user files.

The Button page is organized into twelve groups: default, sizes,
colors, soft, outline, dash, ghost/link, disabled, square/circle, icon, and
loading spinner, and advanced Style/States customization. Common appearances use
ButtonProps Variant/Tone/Size; the theme switch updates their semantic colors.
Copyable code uses the same recipe and typography as each preview. Its spinner is a static vector composition so an example does
not introduce simulated progress or periodic rendering.

The Icon page contains all 2,066 generated Lucide names, including aliases,
through the opt-in `icon/catalog` package. Its case-insensitive substring
search resets to the first result page, and Previous/Next navigation exposes
every match in stable catalog order without constructing the full visual list
in one frame. Each catalog row renders one 24/2 size/stroke preview; a separate
Search-icon example demonstrates 16/1, 24/2, and 32/3 variants.
The catalog itself is an icon-only 16×16 grid with a light semantic background
and theme-adaptive neutral gray icons. Clicking an icon copies its exact
exported Go name, such as `Search`, and reports the result in EVENT. The search
field spans the same full width as the 16-column grid.

The ButtonGroup page demonstrates the default seam-free horizontal group,
opt-in dividers, equal-width vertical orientation, and mixed enabled/disabled
children. Each child remains an independent Button focus stop.

The InputGroup page demonstrates a search field, a passive prefix icon, a
suffix unit, and an independently focusable/activatable suffix Button. Every
example reuses a normal controlled Input under one group border and focus ring.

Practical composition stays inline and copyable: Input includes a controlled
form built from Input, Select, Checkbox, ButtonGroup, and Button; Avatar
includes an Avatar/Text/Badge profile row; Tabs switches ordinary
application-owned content; Menu demonstrates vertical/horizontal inline action
lists with disabled rows and visible callback feedback; and the overlay pages demonstrate interactive
Popover content and a non-intercepting Tooltip. Theme compares Semantic
tokens, a reusable custom Component theme, and a one-off local Style override
under the same live Light/Dark switch.

Component props use direct `Key`, `Style`, `Token`, `States`, and `Pointer`
fields. The Text and Button pages show `Label` and `TextButton` beside the
advanced/custom-content forms; fixed padding/radius examples use `Padding`,
`PaddingXY`, `Edges`, `Corners`, and `Round`. Partial or token-backed values use
`EdgeValues` and `CornerValues`. Pure controlled assignments demonstrate `Assign`, while
callbacks with feedback, validation, reset, or theme work remain explicit.

## Structure

- `main.go` owns flags, App lifecycle, gallery state, and local image/icon
  assets.
- `gallery.go` defines the registry/display contracts and shared shell.
- `examples_layout.go`, `examples_content.go`, and `examples_controls.go`
  register examples by the existing layout/content/control responsibilities.
- `COVERAGE.md` maps every component property and supported state to examples.
- `gallery_test.go` parses the root `view.go` API so a newly exported retained
  component constructor cannot silently disappear from the registry; the
  documented `Label` and `TextButton` compositions remain on their base pages. It also checks
  metadata, live previews, explanatory/code content, categories, theme
  validity and initial selection.

## Add or update a component example

1. Add one `componentExample` entry in the file for its responsibility. Keep
   navigation metadata and content together: `Name`, `Category`, `Purpose`,
   `Coverage`, and a `Build` function.
2. Add focused `exampleBlock` values. Every block requires a short title,
   explanation, real preview, and compilable root-API code snippet.
3. Extend `COVERAGE.md` for every new public property, enum, callback, and
   supported state. Do not demonstrate API that is not present in the source.
4. Run `make fmt` plus `make ci`. The AST-backed registry test discovers
   exported constructors automatically; update the explicit expected ordering
   only when the new page's navigation position is intentional.

Only the selected component's blocks are built, which keeps the catalog useful
as the number of examples grows. Navigation is derived from the same registry,
so there is no separate menu list to synchronize.
