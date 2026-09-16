# Third-party license texts

This directory retains the license text from each pinned redistributable
dependency or asset used by dxui release artifacts. Directory names include
the exact module or asset version when one exists. The root
[`THIRD_PARTY_NOTICES.md`](../../THIRD_PARTY_NOTICES.md) file maps each entry
to its purpose and distribution scope.

License files copied from Go modules come from the exact versions selected by
`go.mod` and `go.sum`, not from their moving default branches. The SDL text is
from the upstream SDL `release-3.4.0` tag. Lucide/Feather and the example Noto
subset retain additional copies at `icon/LICENSE` and
`examples/text_gallery/FONT-LICENSE.txt` respectively because those packages
and assets can be distributed independently.
