# Multi-window example

`go run ./examples/multi_window` opens one main window. Its button creates
independent native child windows; each child has its own retained state and can
close without affecting its siblings. Closing the main window closes all
children and exits the single App event loop.
