# Component example coverage

Source of truth: exported constructors and props in `view.go`, `components.go`,
and `style.go`, checked against `docs/components.md` on 2026-09-07. “Interactive”
means the preview emits a visible result in the page-level EVENT panel.

| Component | Public component properties | Supported states / behavior | Showcase blocks |
| --- | --- | --- | --- |
| `Box` | `BoxProps{Key, Style, Token, States, Pointer, Direction, Gap, Justify, Align}`; zero/variadic children | non-interactive; single-line flex; Vertical is default and changing Direction retains node identity | both directions; all justify/align values; grow/shrink/basis/margin/align-self; absolute/insets/z-index/overflow; common paint/state props; Padding/PaddingXY/Round |
| `Scroll` | common fields, `Axis`, `InitialOffset`, `Offset`, `Scrollbar`, `OnScroll` | retained or controlled offset; focus, wheel, arrows, Home/End, thumb drag; empty/no-range | vertical/horizontal + initial offset; both axes + auto/always/hidden; interactive controlled offset |
| `Colors` | all 26 Primitive 50..950 scales | fixed typed color tokens | typed namespace and complete swatches |
| `Theme` | Primitive, Semantic, and Component tokens; mutable `Theme.Semantic.Colors` and `Theme.Components`; local `Style` | live Light/Dark replacement; all 26 families for each global palette setting; coordinated shades and reset | semantic intent preview; interactive Primary/Surface/Danger/Success configuration through `App.SetTheme`; reusable Component theme beside a one-off local override |
| `Text` / `Label` | common fields, `Value`, `Wrap`, `MaxLines`; default-style Label shorthand | non-interactive; empty content | Label basic value; Text no-wrap/empty; words + unlimited/limited lines; complete `TextStyle` |
| `Icon` | common fields, `Data`, `Size`, `Color`, `StrokeWidth`; `ViewBox`, all path commands; all 2,066 official Lucide names, exact Go names, and aliases | Icon itself is non-interactive; case-insensitive catalog filtering, bounded pagination, and icon-only Button copy actions | 16×16 light-background catalog grid with neutral gray icons and click-to-copy exact `GoName`; one generated icon at 16/1, 24/2, and 32/3; default/token/literal size/color; Move/Line/Quad/Cubic/Close; custom Button child |
| `Image` | common fields, `Source`, `Fit`, `Alignment`, `MaxPixels`, `OnLoad`, `OnError` | loading success, decode/limit error, intrinsic/empty visual placeholder | all fits; alignment 0/.5/1; interactive callbacks and default/low decode limits |
| `Avatar` | common fields, `Source`, `Shape`, `Size`, `OnLoad`, `OnError` | theme/custom square sizing, centered cover, circular/square clip, shared Image load/error lifecycle | default/48/64 sizes; circle/square; border; load and decode-error callbacks; Avatar + Badge profile row |
| `Badge` | common fields, one arbitrary child | non-interactive; compact intrinsic pill; descendant text/icon tint inheritance with local override | text, numeric, and icon badges; compound horizontal Box child; custom background and padding |
| `ProgressBar` | common fields, `Value` | non-interactive determinate left-to-right fill; clamps to 0..1; NaN/-Inf empty and +Inf complete | 0%, 25%, 50%, 100%; custom dimensions, padding, colors, border, radius, opacity, and clip |
| `Button` / `TextButton` | common fields, typed `Variant`, `Tone`, `Size`, `Disabled`, `OnPress`, custom child or centered string label | default, hover, focus, pressed, disabled; pointer pass-through | TextButton default; sizes; colors; soft; outline; dash; ghost/link; disabled and `PointerNone`; square/circle; custom icon/text; static loading spinner |
| `ButtonGroup` | common fields, `Orientation`, `Dividers`, Button children | group is non-focusable; child hover, focus, pressed, disabled remain independent | default seam-free horizontal; horizontal with Dividers; equal-width vertical; mixed Disabled |
| `InputGroup` | common fields; `InputGroupContent{Input, Prefix, Suffix}` with one required Input and optional View adornments | group is non-focusable and owns its border/focus-within surface; passive adornments add no stop; Button adornments retain independent focus and activation | search; prefix icon; suffix unit; suffix action Button |
| `Input` | common fields, `Value`, `OnChange`, `Selection`, `OnSelectionChange`, `Placeholder`, `Password`, `ShowPasswordToggle`, `Disabled`, `ReadOnly`, `OnSubmit`; `Assign` callback helper | empty, focus, selection, composition/caret internally, keyed identity across reorder, read-only, disabled, masked/revealed password with eye hover/pressed, application-styled error | controlled/empty; rune selection; keyed reorder; password toggle/read-only/disabled/nil callback; Assign pure assignment; submit/error style; controlled form with Select, Checkbox, and ButtonGroup |
| `Textarea` | common fields, `Value`, `OnChange`, `Selection`, `OnSelectionChange`, `Placeholder`, `Disabled`, `ReadOnly`, `Wrap` | empty, focus, selection, multiline, no-wrap/wrap, read-only, disabled, internal X/Y scroll | controlled multiline; both wraps + placeholder; selection/read-only/disabled |
| `Select` | common fields, `Value`, `Options`, `Placeholder`, `Disabled`, `OnChange`; option `Value`, `Label`, `Disabled` | placeholder, matched/unmatched/selected, open/active, disabled option/control, empty options | interactive controlled select; placeholder/unmatched/empty/nil callback; disabled and clipped-anchor overlay |
| `Popover` | common fields, controlled `Open`, `Placement`, `Offset`, `OnOpenChange`; anchor and content | activation; interactive content; Tab transfer; Escape/outside top-layer dismissal and focus restore; flip/clamp | controlled interactive overlay |
| `Tooltip` | common fields, `Placement`, `Offset`, `Delay`, `Disabled`; anchor and content | delayed hover/focus open; leave/blur close; no focus or pointer interception | delayed hint; disabled behavior |
| `Tabs` | common fields, `Value`, `Items`, `Disabled`, `OnChange`; item `Value`, `Label`, `Disabled` | matched/empty/unmatched selection, hover, focus, selected, pressed, disabled item/control; arrows, Home/End, Enter/Space, Tab | controlled selector plus application-owned content; empty/unmatched/disabled/nil-callback matrix |
| `Menu` | common fields, controlled `Value`, vertical/horizontal `Orientation`, `Items`, `Disabled`, `OnAction`; item `Value`, `Label`, `Disabled` | default, hover, keyboard current, selected without focus, focus, pressed, disabled item/control; orientation arrows, Home/End, Enter/Space, Tab | matched/empty/unmatched selection; interactive vertical action feedback; horizontal content-width layout; disabled and empty/nil-callback matrix |
| `Checkbox` | common fields, `Checked`, `Disabled`, `OnChange`, one label child | unchecked/checked, hover, focus, pressed, disabled; no tri-state | interactive controlled value; complete boolean matrix and nil callback |
| `Radio` | common fields, `Selected`, `Disabled`, `OnSelect`, one label child | unselected/selected, hover, focus, pressed, disabled; selected cannot deselect itself | multiple controls sharing one business value; selected/disabled/nil callback |
| `ToggleSwitch` | common fields, `Checked`, `Disabled`, `OnChange` | unchecked/checked, hover, focus, pressed, disabled | interactive controlled value; complete boolean matrix/nil callback; runtime theme change |
| `Slider` | common fields, `Value`, `Min`, `Max`, `Step`, `Disabled`, `OnChange` | hover, focus, pressed/dragging, disabled; track click, captured drag, arrows, Home/End | interactive controlled stepped value; defaults, boundary and inert/nil-callback states |

## Direct common fields and `Style`

The `Box`, `Input`, and `Button` blocks jointly cover every common
exported field:

- every component props type directly declares `Key`, `Style`, `Token`,
  `States`, and `Pointer` (`PointerAuto` by
  default and explicit `PointerNone`).
- dimensions: auto zero `Length`, `Px`, `Percent`, width/height, min/max;
- spacing/layout: margin, padding, `Padding`/`PaddingXY`, position/insets, grow, explicit/default
  shrink, basis, align-self, z-index, and overflow visible/clip;
- paint: background, solid/dashed border, non-uniform/uniform/`Round` radius, shadow offset/blur/
  spread/color, opacity set/unset, visibility visible/hidden;
- typography: family fallback, size, line height, all three weights, both
  slants, token/literal color, and start/center/end alignment;
- state patches: default, hover, focus, checked, pressed, and disabled are
  exercised on the semantic controls to which they apply. Patch fields shown
  across the gallery are background, border, radius, shadows, opacity,
  visibility, and text color.

## Non-visual and deliberate limits

- `Key` is demonstrated through a real keyed reorder; identity itself is
  internal and intentionally cannot be printed.
- IME composition and caret-area reporting are internal contracts. Input and
  Textarea provide focused editable surfaces for native manual exercise; the
  public API has no composition callback.
- `ImageFile` is shown in the code/explanation but not bound to a repository
  path, because it intentionally resolves the application's exact runtime
  path. The live preview covers both `ImageBytes` and `ImageFromGo`.
- `PointerNone` is demonstrated by a keyboard-operable Button that ignores
  pointer hit testing, distinguishing it from `Disabled`.
- Loading, error, empty content, selected, expanded popup, read-only, disabled,
  and application-owned error styling are covered where the actual component
  supports them. There is no invented loading prop, validation API, tri-state,
  SVG parser, accessibility label, generic event API, or fake focus/open prop.
