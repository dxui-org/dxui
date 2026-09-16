# VirtualList example

This example displays 100,000 fixed-height rows while its `Build` callback is
invoked only for the mounted viewport plus three overscan rows on each side.
The controls modify, insert, delete, reorder, and jump through the data. Each
interactive row uses a stable business key, so mounted state cannot migrate to
another item.

Run with `go run ./examples/virtual_list` on a supported native desktop.
