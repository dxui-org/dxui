# Third-party notices

dxui incorporates or redistributes the third-party software and assets listed
below. This file is an index; the complete license texts are retained under
[`third_party/licenses`](third_party/licenses) and must accompany packaged
binary artifacts as a `licenses` directory together with this file and the
root dxui `LICENSE`.

| Work | Fixed version or source | Purpose and distribution | License text |
| --- | --- | --- | --- |
| dxui/go-sdl3 | `github.com/dxui-org/go-sdl3` | Owner-controlled Go SDL binding and embedded native-library loader linked into dxui applications. | [MIT](third_party/licenses/go-sdl3/LICENSE) |
| SDL | `3.4.0`, embedded by the selected dxui/go-sdl3 version | Native SDL library compressed into the application and expanded at runtime. | [zlib](third_party/licenses/sdl-3.4.0/LICENSE.txt) |
| purego | `github.com/ebitengine/purego v0.10.0` | Foreign-function interface linked through go-sdl3. | [Apache-2.0](third_party/licenses/purego-v0.10.0/LICENSE) |
| Go runtime-derived purego code | Go Authors code identified by purego v0.10.0 | Selected purego files originate from the Go runtime. | [BSD-3-Clause](third_party/licenses/go-runtime/LICENSE) |
| x/image | `golang.org/x/image v0.38.0` | Font parsing, metrics, rasterization, and image support; supplies the embedded Go Regular font. | [BSD-3-Clause](third_party/licenses/x-image-v0.38.0/LICENSE) |
| x/sys | `golang.org/x/sys v0.7.0` | Transitive operating-system support. | [BSD-3-Clause](third_party/licenses/x-sys-v0.7.0/LICENSE) |
| x/text | `golang.org/x/text v0.35.0` | Transitive Unicode support selected by x/image. | [BSD-3-Clause](third_party/licenses/x-text-v0.35.0/LICENSE) |
| Lucide and Feather icons | `lucide-static 1.41.0`, commit `bca7e75a816dcf1e75e8feb5a3198a68cbb8a052` | Generated official icon resources. Feather-derived names are enumerated in the license file. | [ISC and MIT](third_party/licenses/lucide-static-v1.41.0/LICENSE) |
| Noto Sans CJK SC subset | Subset SHA-256 `ccc6fee93c791d430e843aa5349335c2965cb6f2c89f974fdb3b063312b839c6` | Example-only modified font subset in `examples/text_gallery`; not part of the framework default font. | [SIL OFL 1.1](third_party/licenses/noto-sans-cjk-sc-subset/LICENSE) |

These notices are an engineering inventory, not legal advice and not a license
grant beyond the attached upstream texts.
