# Calculator example

`calc` is an interactive four-function calculator based on the supplied
light/dark reference. It uses only the public root `dxui` package.

```sh
go run ./examples/calc
go run ./examples/calc -dark
go run ./examples/calc -software
```

The `{}` key toggles the sign of the displayed number. The example also
supports decimals, percentages, backspace, chained operations, and theme
switching from the pill button at the top.

Keyboard digits, arithmetic operators, Enter/equals, decimal, and Backspace
use backend-neutral `AppOptions.Shortcuts`. Keyboard and pointer actions call
the same calculator state transition.
