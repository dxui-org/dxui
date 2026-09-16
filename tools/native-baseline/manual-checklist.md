# Windows native correctness checklist

Run the built baseline or login executable twice, first with Auto and then with
`-software`. Save the terminal diagnostic line after each run. Use only dummy
credentials.

1. Confirm the window appears with complete content after the first visible
   frame. Resize to the minimum, enlarge, maximize, minimize, restore, cover it
   with another window, then uncover it. Observe complete repaint, no persistent
   blank region and no sustained resize stall.
2. Tab forward and Shift+Tab backward through the login controls. Activate the
   checkbox, theme switch and buttons with Space/Enter as applicable. Confirm
   disabled Login is skipped/inert and keyboard focus is visibly indicated.
3. In Input and Textarea, type and reject/accept controlled edits, move the
   caret, create a selection and scroll. Rebuild through theme/control changes;
   confirm value, caret, selection and scroll stay at the expected position.
4. Open Select and Popover. Use arrows, Home, End, Enter and Escape; click
   outside; confirm close behavior and focus returns to the anchor.
5. Copy `中文剪贴板往返` from one dxui editor and paste it into another, then
   copy it back to a separate trusted editor and compare the exact text.
6. With the installed Chinese IME, enter an editor and exercise pre-edit text,
   candidate choice, caret movement, commit, Escape cancellation and focus
   loss. Confirm candidates follow the caret. Record the IME name/version.
7. Toggle light/dark while controls are hovered, pressed and keyboard-focused;
   confirm state visuals remain distinguishable and content is retained.
8. Move the window between displays with different physical scaling. Record
   logical/pixel diagnostics before and after and confirm text, hit testing,
   popup placement and caret candidate placement. A single-display simulated
   scale does not pass this item.
9. Trigger a real renderer/device reset using the available GPU/VM procedure,
   if one exists, and confirm resources recreate and the complete scene returns.
   A fake reset unit test does not pass this item.
10. For forced software, confirm diagnostics say exactly `software`. For Auto,
    record the actual name and fallback bit. A working software renderer alone
    does not establish operation with the GPU disabled.

For each step record `pass`, `fail`, `not run`, or `blocked`, plus renderer,
display scale, exact action and observation. Never enter a real password or
include one in a screenshot.
