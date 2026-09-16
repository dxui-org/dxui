# Lucide icon gallery

This example intentionally imports `icon/catalog`, so every canonical Lucide
name and compatibility alias is linked. It shows the complete catalog in a
virtualized, vertically scrolling grid of compact icon-only buttons. Only the
visible rows and a small overscan window are built. The full-width search field
filters by exported Go constructor name (and Lucide source name) and resets the
viewport to the first match. Hover or keyboard-focus a button to see its exact
exported Go constructor name below the icon. The light background, tile states,
and icon color follow the Lucide icon catalog.

Normal applications should import `github.com/dxui-org/dxui/icon` and call only
the constructors they use; they should not import `icon/catalog`.

```sh
go run ./examples/icon_gallery
```
