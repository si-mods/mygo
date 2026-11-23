# Fork sync log

Maintenance notes for the si-mods/mygo fork of egoist/mygo. This is not a
product document: `docs/` is published on the website, and a page added
there is picked up by the docs nav. Nothing here was sent to egoist/mygo.
The candidates at the end are an assessment only.

## 2026-11-23

Sync of `upstream/main` into a fresh branch from current fork `main`.
No open pull requests were on the fork. si-mods/mygo is still a fork of
egoist/mygo, and both default branches are still `main`.

| | |
| --- | --- |
| Fork tip started from | `5895deae` — round undecorated windows' corners on X11 ([#42](https://github.com/si-mods/mygo/pull/42)) |
| Upstream tip merged | `a0eaaaaf` — pass the inherited environment to process.execve ([egoist/mygo#2222](https://github.com/egoist/mygo/pull/2222)) |
| Merge base | `31bf65e6` — the tip the last merged sync ([#43](https://github.com/si-mods/mygo/pull/43)) already contained |
| Compare `origin/main...upstream/main` before the merge | 55 fork-only, 173 upstream-only |
| Upstream range | `31bf65e6..a0eaaaaf` |
| Merge commit | `91794e51` |
| Pull request | [#44](https://github.com/si-mods/mygo/pull/44) (`sync-upstream-20261123`) |

Upstream in that range includes releases 0.9.2 and 0.10.0, apps on iOS
and Android ([#940](https://github.com/egoist/mygo/pull/940)), a Windows
path linked as a whole from its drive or a UNC share
([#2120](https://github.com/egoist/mygo/pull/2120)), rounding the corners
of transparent Windows 10 windows
([#2176](https://github.com/egoist/mygo/pull/2176)), a sideways wheel
reported to a program that takes the mouse
([#2118](https://github.com/egoist/mygo/pull/2118)), a disabled drawn
menu item's tooltip ([#1372](https://github.com/egoist/mygo/pull/1372)),
`PageOptions.ZoomControl` ([#2097](https://github.com/egoist/mygo/pull/2097)),
and the CLI passing the inherited environment to `process.execve`
([#2222](https://github.com/egoist/mygo/pull/2222)). Fork-only work kept
on top includes SystemGlass, `SetDrawMenus`, `Theme.Menu`, `LinkPath`,
OSC 9;4 `OnProgress`, `makeExecutable`, the half-minute updater retry,
libghostty-vt 1.0.1, `ReadsClipboard`, performable Command+C /
Control+Shift+C, the autolink / path-link fixes, `v.top`, and the X11
corner rounding from [#42](https://github.com/si-mods/mygo/pull/42).

### Conflicts

Six textual conflicts. Several other files changed on both sides and
auto-merged; those were read. Two build tags that auto-merged without a
marker would have compiled `corners.go` into the Android build of
`internal/linux`, whose other files now exclude Android.

| File | Root cause | What we kept |
| --- | --- | --- |
| `ui/paint.go` | Fork [#42](https://github.com/si-mods/mygo/pull/42) clips the content to `windowCorners` (X11, `CornerSurface`). Upstream [#2176](https://github.com/egoist/mygo/pull/2176) clips the frame with `roundFrame` (transparent Windows 10, `ContentCorners`). Git put the two functions in one body. | Both functions, and both call sites. `windowCorners` still cuts the content. `roundFrame` still cuts the frame. `isSystemGlass` still keeps a SystemGlass element out of a fade group. |
| `ui/popovermenu.go` | Fork `Theme.Menu` sets row height, radius and padding, then called `Disabled`. Upstream [#1372](https://github.com/egoist/mygo/pull/1372) stopped calling `Disabled`: a disabled item is `unavailable`, so it stays under the pointer and its tooltip can say why. | The theme's row style. `row.Label` without `Disabled`. `themed` already passed `parts.level` (the auto-merge did not revive `m.level`). |
| `ui/api_gen.go` | Both sides added a wrapper in the same alphabetical slot (`SetDrawMenus`, `SafeArea`). | `go generate ./ui`. `SafeArea` and `SetDrawMenus` / `DrawMenus` are both there. Not hand-merged. |
| `plugins/terminal/links.go` | Fork directories (`~/`, `./`, `../`) and drive paths, and upstream [#2120](https://github.com/egoist/mygo/pull/2120) (a Windows path, drive or UNC, is one link; a backslash elsewhere is not). The two `drive` patterns were the same expression. | One `drivePath`. `windowsPath` decides which names take `\` as a separator, so `D:\dev\main.go` and `\\server\share\a.pdf` are whole links and `hello\nworld` is not. `dirOnly` still accepts `~/`, `./` and `../`, which hold no letter. `pathAtRows` is unchanged. `isDir` treats a `windowsPath` that ends in `\` as a directory, so a UNC share is not a link unless `LinkPath` says so. |
| `plugins/terminal/links_test.go` | Both sides appended path cases and a span test. | Fork directory, CJK, drive and `pathAtRows` cases, plus upstream's Windows and UNC rows and `TestPathAtWindowsSpan`. |
| `docs/plugins/terminal.md` | Fork lists `docs/` among printed paths. Upstream lists `D:\dev\main.go`. | Both. |

`internal/linux/corners.go`, `corners_test.go` and
`internal/e2e/corners_linux_test.go` kept the pre-Android tag
`linux && (amd64 || arm64)`. Upstream's Linux package is now
`linux && !android`. Those three files got `!android` too, so an Android
build does not compile the X11 corner code alone. `go list` for
`GOOS=android` excludes every file of `internal/linux`.

### Semantic review of auto-merged files

| File | What both sides needed | Resolution |
| --- | --- | --- |
| `ui/menu.go`, `ui/menupresentation.go` | Upstream's edit commands follow `platform.MacKeys` (iOS as well as macOS). The fork still opens a drawn editing menu on the last pass and falls back to the system menu for a toast. | Upstream command list. Fork `editMenu`, `requestFrame` when `inFrame && pass >= 2`, and `openSystemEditMenu`. `menupresentation.go` was unchanged upstream. |
| `ui/runtime.go` | Upstream's frame work, plus the fork placing SystemGlass after present. | `placeSystemGlass()` after `placeNativeGlass` in `runFrame` and after `present` in `repaintFrame`. |
| `ui/theme.go` | Upstream's body font sizes on iOS and Android. Fork `Theme.Menu`. | Both. |
| `ui/input.go`, `ui/handler.go`, `internal/darwin/surface.go` | Upstream refactored the key by position. The fork still delivers the key of a menu command (Command+C) on `SurfaceCommand`. | `pressedKey` is upstream's. A command still carries `key`, `physical`, `mods` and `repeat`. |
| `ui/transition.go` | A material paints in place of the background, which must not fade in under it. | The fork's `fillMaterial` target stays, inside upstream's color animation. |
| `internal/platform/surface.go` | Upstream `ViewSurface` and touch. Fork `SystemGlassSurface` and `CornerSurface`. | Sibling types. SystemGlass was not folded into `GlassSurface` or `ViewSurface`. X11 rounding was not folded into `ContentCorners`. |
| `internal/linux/window.go`, `app.go`, `surface.go`, `testhooks.go` | Upstream's hidden title bar on KDE, and the accel group released on close. Fork [#42](https://github.com/si-mods/mygo/pull/42) shapes X11 corners and watches allocation. | Both. `watchAllocation` serves the input shape and the corner shape. `blendCorners` still runs before the window is realized. |
| `window.go`, `ui/window.go`, `internal/platform/platform.go`, `docs/windows.md` | Upstream documented new window options. Fork documented X11 `CornerRadius`. | Both. `windowHost.windowCorners` still reads `CornerSurface`. |
| `plugins/terminal/view.go`, `terminal.go` | Upstream [#2118](https://github.com/egoist/mygo/pull/2118) sends a sideways wheel to a program that takes the mouse, and otherwise to the handlers outside the view. The fork still joins a wrapped path through `pathAtRows` and trims autolink prose. | Both. `OpenLink`'s comment names Windows paths. `pathAtRows` is still the filter after `pathAt`. |
| `plugins/terminal/paint.go`, `updater.go`, `mygo-plugin.json` | Unchanged upstream since `31bf65e6`. | Fork text. `v.top` is still set after the cursor is known. `httpRetries` is still `1s`, `5s`, `20s`. The pin is still ghostty `108cacb`. |
| `internal/darwin/app.go`, `notify.go` | Upstream's notification flow. The fork asks for authorization when the Dock badge is first set. | `requestNotificationAuthorization`, with `notifyAsking` so a badge and a notification do not ask twice. |
| `internal/darwin/surface.go`, `testhooks.go` | Upstream's command key. Fork SystemGlass panes. | `systemGlass` and `TestSystemGlass` stay. `destroy` still calls `destroySystemGlass`. |

### Behavior that must keep working

The 2026-11-16 list still applies. In addition:

- X11 still rounds an undecorated window's corners ([#42](https://github.com/si-mods/mygo/pull/42)): smooth where a compositor blends an alpha frame, hard-edged otherwise, square while maximized, tiled or full screen, and not on Wayland. Transparent windows are not cut.
- A transparent Windows 10 window still cuts its frames with `roundFrame` ([#2176](https://github.com/egoist/mygo/pull/2176)). That path is `ContentCorners`, not `CornerSurface`.
- A disabled drawn menu item is unavailable, not `Disabled`, so its tooltip still shows. `Theme.Menu` still styles the row.
- A Windows path is one link from its drive or UNC root. `~/`, `./` and `../` are still paths. A directory, including `\\server\share\`, is still not a link unless `LinkPath` says so.
- `pathAtRows` is still a filter after `pathAt`.

### Tests

`CGO_ENABLED=0 go build ./...`, `GOOS=darwin`, `GOOS=windows`, and
`GOOS=js GOARCH=wasm` (175 packages, excluding `cmd/mygo` and
`internal/e2e`) all built. `go vet` passed for linux, darwin, windows,
`-tags mygo_noinspector`, and js/wasm (wasm vet excluded `cmd/mygo`).
`GOOS=android GOARCH=arm64 go list` excludes every file of
`internal/linux`.

`CGO_ENABLED=0 go test -count=1 -skip TestDiffFrameAllocs ./...` passed
except `internal/nativegen` `TestScaffoldBuildsRust Zig`: rustc warns
`struct NativeB is never constructed` on the generated stub, and Zig is
not installed (logged, not the failure). That test is upstream
native-module code and was not edited in the resolution.
`plugins/terminal`, `ui` and `internal/linux` passed, including the link,
viewport, menu, updater and corner tests. `TestDiffFrameAllocs` was
skipped (it already fails on upstream). GUI e2e on macOS and Windows was
not run.

Regression coverage for this resolution: `TestMenuStyle*`,
`TestDrawMenus*`, `TestSystemGlass*`, `TestPathAt`, `TestPathAtDrive`,
`TestPathAtWindowsSpan`, `TestPathAtRows`, `TestIsDir`,
`TestCommandClickDirs`, `TestViewShowsTheRowsAroundTheCursorOfATallerScreen`,
`TestMakeExecutable`, `TestHTTPGetRetries`, `TestReadsClipboard`,
`TestPerformableCopyKeys`, and `TestCorner` in `internal/linux` and `ui`.

### Hotspots

The same files remain hot: `ui/popovermenu.go` (`parts.level`, and now
do not put `Disabled` back on a drawn item), `ui/paint.go` (two corner
clips), `plugins/terminal/view.go` / `links.go` (`pathAtRows` after
`pathAt`), and `updater.go`.

Smaller next time, beyond the 2026-11-16 list:

- `windowCorners` and `roundFrame` are different clips. Do not collapse
  them into one function, and do not drop either call site.
- In `pathOf`, keep `dirOnly` when taking upstream's colon check, and
  keep `windowsPath` for the separator. Do not put a second `drive`
  pattern next to `drivePath`.
- Linux files the fork adds need `linux && !android`, the same tag as
  the rest of `internal/linux`, or an Android build compiles them alone.
- A drawn item's theme style stays. Its disabled state is `unavailable`
  from `item`, which is what lets the tooltip show.

## 2026-11-16

Sync of `upstream/main` into a fresh branch from current fork `main`.
The 2026-11-09 attempt ([#25](https://github.com/si-mods/mygo/pull/25),
`sync-upstream-20261109`, upstream through `c4197b7f`) was never merged
and is superseded by this one. It was not merged into `main`.

| | |
| --- | --- |
| Fork tip started from | `381cb550` — bind libghostty-vt 1.0.1 ([#41](https://github.com/si-mods/mygo/pull/41)) |
| Upstream tip merged | `31bf65e6` — PopoverMenu takes Panel and Item options ([egoist/mygo#1889](https://github.com/egoist/mygo/pull/1889)) |
| Merge base | `659a325e` — the tip the last merged sync ([#23](https://github.com/si-mods/mygo/pull/23)) already contained |
| Compare `origin/main...upstream/main` before the merge | 50 fork-only, 281 upstream-only |
| Upstream range | `659a325e..31bf65e6` |
| Merge commit | `e66ba756` |
| Pull request | [#43](https://github.com/si-mods/mygo/pull/43) (`sync-upstream-20261116`) |

Upstream in that range includes PopoverMenu Panel and Item options
([#1889](https://github.com/egoist/mygo/pull/1889), [#1742](https://github.com/egoist/mygo/pull/1742)),
`Element.Touches` ([#1884](https://github.com/egoist/mygo/pull/1884)),
`Window.CornerRadius` ([#1849](https://github.com/egoist/mygo/pull/1849);
Linux returns 0), native modules ([#1430](https://github.com/egoist/mygo/pull/1430)),
a wrapped link opened as a whole ([#1677](https://github.com/egoist/mygo/pull/1677)),
reused scrolled rows ([#1687](https://github.com/egoist/mygo/pull/1687)),
symbols drawn as text and centered on Windows ([#1551](https://github.com/egoist/mygo/pull/1551)),
glyphs shrunk to mid-cell ([#1550](https://github.com/egoist/mygo/pull/1550)),
`OnInstalled` ([#1577](https://github.com/egoist/mygo/pull/1577)),
stalled-download retry ([#875](https://github.com/egoist/mygo/pull/875)),
and release 0.9.1. Fork-only work kept on top includes SystemGlass,
`SetDrawMenus`, `Theme.Menu`, `LinkPath`, OSC 9;4 `OnProgress`,
`makeExecutable`, the half-minute updater retry, libghostty-vt 1.0.1,
`ReadsClipboard`, performable Command+C / Control+Shift+C, and the
autolink / path-link fixes ([#28](https://github.com/si-mods/mygo/pull/28),
[#34](https://github.com/si-mods/mygo/pull/34)–[#40](https://github.com/si-mods/mygo/pull/40)).

### Conflicts

Six textual conflicts. Several other files changed on both sides and
auto-merged; those were read, and two of them did not compile until the
semantic fixes below.

| File | Root cause | What we kept |
| --- | --- | --- |
| `internal/text/dwrite_windows.go` | Fork [#26](https://github.com/si-mods/mygo/pull/26) / [#27](https://github.com/si-mods/mygo/pull/27) and upstream [#1551](https://github.com/egoist/mygo/pull/1551) both draw a text-presentation symbol in the text color. Fork [#27](https://github.com/si-mods/mygo/pull/27) also selects Segoe UI Symbol. Upstream does that and centers the narrower glyph (`cellShifts`). | Upstream file. The fork copy is covered and extended. |
| `plugins/terminal/paint.go` | Fork `v.top` (a screen taller than the view keeps the cursor in sight) and the mid-cell shrink landed in functions upstream rewrote for row reuse ([#1687](https://github.com/egoist/mygo/pull/1687)) and the same shrink ([#1550](https://github.com/egoist/mygo/pull/1550)). | Upstream `updateRows`, `nextRun`, and `shape([]rune)`. `v.top` is set after the cursor is known, and `draw` still starts at that row. The shrink math is upstream's. |
| `plugins/terminal/view.go` | Upstream [#1677](https://github.com/egoist/mygo/pull/1677) walks a soft-wrapped hyperlink or printed URL across rows. Fork autolink ([#34](https://github.com/si-mods/mygo/pull/34)–[#38](https://github.com/si-mods/mygo/pull/38)) trims prose inside an OSC 8 URL on one row, and `pathAtRows` ([#40](https://github.com/si-mods/mygo/pull/40)) joins a path the program wrapped and then applies `LinkPath`. | Both. A clean hyperlink and a printed URL use upstream `spanOf`. An autolink that contains prose stays this row's cells, with `last` set to that row so the new `linkSpan` underlines it. Paths go through `pathAtRows`, which calls `pathAt` and then `isLink`. Calling `pathAt` on the joined text alone would open a truncated first row and would link a directory whenever `OpenLink` is set. |
| `plugins/terminal/terminal_test.go` | Both sides appended a test after `TestCommandClickLinks`. `linkSpan` grew a `last` field, so unkeyed literals needed a fourth value. | Upstream `TestCommandClickWrappedLink` stayed. `TestCommandClickDirs` moved to `terminal_fork_test.go`. Row 0 spans use `last` 0; a span on row 1 uses `last` 1. |
| `ui/paint.go` | The fork resets `systemGlassPlaces` at the start of a paint. Upstream begins the path spill there. | Both lines. `isSystemGlass` still keeps a SystemGlass element with opacity below 1 and children out of a fade group. |
| `ui/theme.go` | The fork added `Theme.Menu`. Upstream added `followed` for `WithPreferences`. | Both fields, and both `MenuStyle` and `followedTheme`. |

### Semantic review of auto-merged files

| File | What both sides needed | Resolution |
| --- | --- | --- |
| `ui/popovermenu.go` | Upstream split `drawnMenuBuilt` and renamed `*menuParts` from `m` to `parts` ([#1889](https://github.com/egoist/mygo/pull/1889) also added per-menu `Panel` / `Item`). The fork hook still said `m.level`. Auto-merge left that on the `*Menu` parameter, which has no `level` field. Same miss as the unmerged 2026-11-09 attempt. | `themed` passes `parts.level`. `SetDrawMenus` / `DrawMenus` and `Theme.Menu` stay. Upstream `PopoverMenuOptions.Panel` / `Item` stay; they style one menu, and `Theme.Menu` styles every drawn menu. |
| `ui/drawmenus_drawer_test.go` | Upstream renamed `menusInWindow.contextMenu` to `beginMenu`. The test still named the old field. | The test checks `beginMenu`. |
| `ui/menu.go`, `ui/menupresentation.go` | Upstream stores `beginMenu` / `menuBuilt` and opens an editing menu through the system menu only. The fork still snapshots `openAt`, asks for a frame on the last pass, and falls back to the system menu for a toast. | Upstream slots and `editCommands`. Fork `editMenu` still calls `openEditMenu` and `host.requestFrame`. `openDrawnEditMenu` still calls `requestFrame` when `inFrame && pass >= 2`. `drawEditMenu` still calls `openSystemEditMenu` for a toast. |
| `ui/runtime.go` | Upstream's frame work, plus the fork placing SystemGlass after present. | `systemGlassPlaces`, `placeSystemGlass()` after `placeNativeGlass` in `runFrame` and after `present` in `repaintFrame`. |
| `internal/platform/surface.go` | Upstream `ViewSurface` (native modules). Fork `SystemGlassSurface`. | Sibling types. SystemGlass was not folded into `GlassSurface` or `ViewSurface`. |
| `plugins/terminal/links.go`, `links_test.go` | Upstream's only delta since `659a325e` is the non-ASCII URL pattern ([#1481](https://github.com/egoist/mygo/pull/1481)), which the fork already had. The fork also has `pathAtRows`, Windows drive paths, `isLink` / `LinkPath`, and full-width punctuation. | Fork text. No second URL pattern. |
| `plugins/terminal/keybind.go` | Upstream did not change this file since the merge base. Fork [#33](https://github.com/si-mods/mygo/pull/33) makes Command+C and Control+Shift+C performable. Upstream already had performable Control+C on Windows. | Fork text. |
| `plugins/updater/updater.go` | Fork [#32](https://github.com/si-mods/mygo/pull/32) and upstream [#1577](https://github.com/egoist/mygo/pull/1577) added the same `OnInstalled`. Upstream also rechecks while a release is being offered. | Upstream file. The `OnInstalled` body matches. Not kept as a second implementation. |
| `updater.go` | Fork `makeExecutable` and a longer `httpRetries` (`1s`, `5s`, `20s`). Upstream [#875](https://github.com/egoist/mygo/pull/875) retries a stalled download; its gateway budget is still `1s`, `3s`. | Fork `makeExecutable` (exec bits from read bits, through a symlink) and the half-minute budget. Upstream's stall handling stays. |
| `plugins/terminal/mygo-plugin.json` | Fork [#41](https://github.com/si-mods/mygo/pull/41) pins libghostty-vt `108cacb` (1.0.1) on si-mods releases. Upstream is still `befcdfd`. | Fork pin. |
| `ui/api_gen.go` | Both sides added methods (`SetDrawMenus` / `DrawMenus`; `Touches`, `RowClicked`, `WindowDoubleClicked`, and the rest of upstream). | `go generate ./ui` matched the auto-merged file. Not hand-merged. |

`plugins/terminal/cell_width_test.go` and the updater plugin doc were the same change on both sides.

### Behavior that must keep working

The 2026-11-08 list still applies. In addition:

- `Theme.Menu.Panel` receives the `*menuParts` nesting level (`parts.level`), not a field of the `*Menu` being built.
- A path the program wraps onto the next row is one link. A directory is still not a link unless `LinkPath` says so.
- An OSC 8 hyperlink that includes prose ends at the URL. A clean hyperlink and a printed URL still span the rows they wrap over.
- `v.top` still shows the rows around the cursor when the screen is taller than the view.
- `httpRetries` is still about half a minute. Upstream's stall retry does not replace that.

### Tests

`CGO_ENABLED=0 go build ./...`, `GOOS=darwin`, `GOOS=windows`, and
`GOOS=js GOARCH=wasm` (150 packages, excluding `cmd/mygo` and
`internal/e2e`) all built. `go vet` passed for linux, darwin, windows,
`-tags mygo_noinspector`, and js/wasm.

`CGO_ENABLED=0 go test -count=1 -skip TestDiffFrameAllocs ./...` passed
except `internal/nativegen` `TestScaffoldBuildsRustZig`: rustc 1.83
warns `struct NativeB is never constructed` on the generated stub, and
Zig is not installed (logged, not the failure). That test is upstream
native-module code and was not edited in the resolution.
`plugins/terminal` and `ui` passed, including the link, viewport, menu,
and updater tests. `TestDiffFrameAllocs` was skipped (it already fails
on upstream). GUI e2e on macOS and Windows was not run.

Regression coverage for this resolution: `TestMenuStyle*`,
`TestDrawMenusEditMenuAsksForAFrame`, `TestCommandClickDirs`,
`TestCommandClickWrappedLink`, `TestPathAtRows`, `TestViewShowsTheRowsAroundTheCursorOfATallerScreen`,
`TestMakeExecutable`, `TestHTTPGetRetries`, `TestReadsClipboard`,
`TestPerformableCopyKeys`.

### Hotspots

The same files remain hot, and this sync collided on them again:
`ui/popovermenu.go` (the `parts.level` miss), `ui/paint.go`,
`plugins/terminal/view.go` / `links.go`, `plugins/terminal/paint.go`
(`v.top` against row reuse), and `updater.go`.

Smaller next time, beyond the 2026-11-08 list:

- After a rename in `drawnMenuBuilt`, the theme hook's level is
  `parts.level`. `PopoverMenuOptions.Panel` is a different callback and
  does not receive a level.
- `pathAtRows` stays a filter after `pathAt`. Do not replace it with
  `pathAt` on the joined text: a program-wrapped path's first row can
  look like a shorter path.
- Fork tests that would land at the bottom of a file upstream also
  appends to go in `terminal_fork_test.go` (`TestCommandClickDirs` moved
  there for that reason).
- Where upstream already has the behavior, drop the fork copy. This
  sync did that for `OnInstalled`, the mid-cell shrink, Segoe UI Symbol,
  and the non-ASCII URL pattern.

Open [si-mods/mygo#42](https://github.com/si-mods/mygo/pull/42)
(`feat/linux-x11-corner-radius`) was not touched. After this sync it
will need a rebase. Both sides edit `ui/paint.go`,
`internal/platform/surface.go`, `internal/linux/window.go`,
`ui/window.go`, `window.go`, `docs/architecture.md`, `docs/windows.md`,
and the Linux app, surface, and testhook files. Upstream Linux
`CornerRadius` returns 0. #42 adds X11 rounding on top of that.

## 2026-11-09

Sync of `upstream/main` into fork `main`.

| | |
| --- | --- |
| Fork tip merged | `677bef7b` — Merge pull request [#23](https://github.com/si-mods/mygo/pull/23) |
| Upstream tip merged | `c4197b7f` — cut panes in a continuous frame's corners in Metal's shader ([egoist/mygo#1516](https://github.com/egoist/mygo/pull/1516)) |
| Merge base | `659a325e` — the tip the previous sync ([#23](https://github.com/si-mods/mygo/pull/23)) already contained |
| Compare `origin/main...upstream/main` before the merge | 34 fork-only, 28 upstream-only (left is the fork, right is upstream) |
| Upstream range | `659a325e..c4197b7f` |

Upstream in that range includes native modules (Go interfaces in Swift,
Rust or Zig, [egoist/mygo#1430](https://github.com/egoist/mygo/pull/1430)),
`Power.KeepRunning` ([#1486](https://github.com/egoist/mygo/pull/1486)),
`App.ChosenLocale` ([#1470](https://github.com/egoist/mygo/pull/1470)),
`ClipboardOptions.Concealed` ([#1476](https://github.com/egoist/mygo/pull/1476)),
`Bitmap.UpdateYUV` ([#1487](https://github.com/egoist/mygo/pull/1487)),
`WindowDoubleClicked` ([#1490](https://github.com/egoist/mygo/pull/1490)),
`RowClicked` ([#1513](https://github.com/egoist/mygo/pull/1513)),
hiding a button's tooltip while its menu shows ([#1491](https://github.com/egoist/mygo/pull/1491)),
the non-ASCII URL boundary ([#1481](https://github.com/egoist/mygo/pull/1481)),
WebView2 custom-scheme streaming ([#1504](https://github.com/egoist/mygo/pull/1504)),
and GPU clip work ([#1489](https://github.com/egoist/mygo/pull/1489),
[#1516](https://github.com/egoist/mygo/pull/1516)). Fork-only work kept on
top includes SystemGlass, `SetDrawMenus`, `Theme.Menu`, `LinkPath`, OSC 9;4
`OnProgress`, and exec bits on Windows-built archives.

### Conflicts

GitHub merged `egoist/main` into the sync branch with no textual
conflicts (`merge_type: merge`). Nine files still differed from both
parents after the merge (a clean three-way result). Two other files
changed on both sides since `659a325e` and kept the fork text, because
the fork already contained the upstream edit.

| File | Root cause | What we kept |
| --- | --- | --- |
| `ui/menu.go` | Upstream routes `ContextMenu` / `Menu` through `drawMenuWith` so the build function does not escape every frame, and hides the trigger's tooltip when a menu is asked ([#1491](https://github.com/egoist/mygo/pull/1491), [#1514](https://github.com/egoist/mygo/pull/1514)). The fork still opens a drawn editing menu on the last pass and falls back to the system menu for a toast. | Upstream `drawMenuWith`. Fork `editMenu` still calls `menusInWindow.openEditMenu` and `host.requestFrame`; `openSystemEditMenu` stays the toast fallback. |
| `ui/popovermenu.go` | Upstream split `drawMenu` into `beginDrawnMenu` / `drawnMenuBuilt` and renamed the local `*menuParts` from `m` to `parts`. The fork's `Theme.Menu` panel hook still said `m.level`. Auto-merge left `m.level` pointing at the new `*Menu` parameter, which has no `level` field. | Upstream split, `popupShows`, and allocation rewrite. `SetDrawMenus` / `DrawMenus` and `Theme.Menu` stay. After review, `themed` passes `parts.level`. |
| `ui/menupresentation.go` | Upstream now stores `beginMenu` / `menuBuilt` on `menusInWindow` instead of `contextMenu` / `menuButton`. The fork still snapshots `openAt` and asks for a frame on the last pass. | Upstream function slots. Fork `openAt` copy, `requestFrame` when `inFrame && pass >= 2`, and the toast fallback in `drawEditMenu`. |
| `ui/runtime.go` | Upstream added native-module views, same-frame hover after a scroll ([#1508](https://github.com/egoist/mygo/pull/1508) / [#1515](https://github.com/egoist/mygo/pull/1515)), `quietClicks`, and `handleDoubleClick`. The fork still places SystemGlass after present. | Both. `systemGlassPlaces` and `placeSystemGlass()` remain. |
| `ui/paint.go` | Upstream draws a native module's texture (`drawTexture`). The fork still skips fade groups for SystemGlass and resets `systemGlassPlaces`. | Both. `isSystemGlass` still keeps those elements flat. |
| `ui/api_gen.go` | Both sides added wrappers (`SetDrawMenus` / `DrawMenus` on the fork; `RowClicked`, `WindowDoubleClicked`, `Multiline`, `NativeView` upstream). | All of them. Branch has every upstream generated function plus the two fork wrappers. |
| `internal/platform/surface.go` | Upstream added `ViewSurface` for native-module views. The fork still has `SystemGlassSurface`. | Both, as separate interfaces. SystemGlass types were not folded into `GlassSurface`. |
| `docs/ui/windows.md` | Upstream documented `WindowDoubleClicked`. The fork documented SystemGlass. | Both sections. |
| `docs/architecture.md` | Upstream added native modules, `KeepRunning`, concealed clipboard writes, and related tables. The fork documented SystemGlass among the other internals. | Both. The SystemGlass paragraph stays in its previous place. |
| `plugins/terminal/links.go` | No textual conflict. Upstream [#1481](https://github.com/egoist/mygo/pull/1481) is the same URL pattern the fork already landed as [#21](https://github.com/si-mods/mygo/pull/21). | Fork file: identical `urlPattern`, plus `LinkPath` / `isDir` / `~/` `./` `../`. |
| `plugins/terminal/links_test.go` | Same pair: upstream added `TestURLAtNotASCII`, which the fork already had, and the fork also tests directories. | Fork file: `TestURLAtNotASCII` plus the directory rows of `TestPathAt` and `TestIsDir`. |

### Behavior that must keep working

The 2026-11-08 list still applies. In addition, `Theme.Menu.Panel` must
receive the `*menuParts` nesting level (`parts.level`), not a field of
the `*Menu` being built.

### Tests

The 2026-11-08 set still applies, especially `TestMenuStyle*`,
`TestDrawMenus*`, `TestURLAtNotASCII`, `TestPathAt`, and `TestIsDir`.
`ui/popovermenu.go` would not compile until `themed` used `parts.level`.

### Hotspots

The same files remain hot. This sync did not collide in text on the
menu files, but `ui/popovermenu.go` still needed a semantic fix after
upstream renamed a local the fork's `Theme.Menu` hook used. The terminal
URL work did not collide because [#21](https://github.com/si-mods/mygo/pull/21)
and [egoist/mygo#1481](https://github.com/egoist/mygo/pull/1481) are the
same change.

Smaller next time:

- After a rename in `drawnMenuBuilt`, read every remaining `m.` in the
  fork hook: `m` is now the `*Menu` parameter.
- Keep treating `LinkPath` as a filter after upstream `pathAt`. The URL
  pattern no longer needs a fork copy; upstream `#1481` owns it.
- Keep SystemGlass on its own types. Upstream's new `ViewSurface` is a
  sibling of `SystemGlassSurface`, not a replacement.

## 2026-11-08

Sync of `upstream/main` into fork `main`.

| | |
| --- | --- |
| Fork tip merged | `07f122e9` — end a URL at the first non-ASCII character ([#21](https://github.com/si-mods/mygo/pull/21)) |
| Upstream tip merged | `659a325e` — showcase MyVNC ([egoist/mygo#1480](https://github.com/egoist/mygo/pull/1480)) |
| Merge base | `ee3d4bd9` — the tip the previous sync ([#13](https://github.com/si-mods/mygo/pull/13)) already contained |
| Compare `origin/main...upstream/main` before the merge | 31 fork-only, 136 upstream-only (left is the fork, right is upstream) |
| Upstream range | `ee3d4bd9..659a325e` |

Upstream in that range includes menu presentation
([#1359](https://github.com/egoist/mygo/pull/1359), richer drawn items
[#1391](https://github.com/egoist/mygo/pull/1391) / [#1392](https://github.com/egoist/mygo/pull/1392)),
terminal path places ([#1302](https://github.com/egoist/mygo/pull/1302)),
`Options.OnDrop` ([#1466](https://github.com/egoist/mygo/pull/1466)),
updater HTTP retry ([#1436](https://github.com/egoist/mygo/pull/1436)),
`ui/video`, and releases 0.8.0–0.8.3. Fork-only work kept on top includes
SystemGlass, `SetDrawMenus`, `Theme.Menu`, `LinkPath`, OSC 9;4
`OnProgress`, the non-ASCII URL boundary ([#21](https://github.com/si-mods/mygo/pull/21)),
and exec bits on Windows-built archives ([#14](https://github.com/si-mods/mygo/pull/14)).

### Conflicts

Eight files. Each one is the same kind of collision: both sides added
behavior in the same place after `ee3d4bd9`. The 2026-11-05 sync (#13)
had already collided on `ui/menu.go` (the `menuAt.last` field) and on the
glass types. These eight will collide again if the fork keeps a second
copy of an upstream mechanism.

| File | Root cause | What we kept |
| --- | --- | --- |
| `ui/menu.go` | Fork drew every menu through its own `drawer` and deferred the editing menu to the next frame. Upstream replaced that with `SetMenuPresentation` and `openDrawnEditMenu`. | Upstream engine. `SetDrawMenus(true)` is `PopoverMenus`. A drawn editing menu still opens where it was asked (`openAt` copies x, y, key, below, last). On the last pass of a frame, `host.requestFrame` runs, because `requestFrame` does nothing while a frame runs. A text input that is not in the tree (a toast) falls back to the system menu. |
| `ui/popovermenu.go` | Same drawer, plus `Theme.Menu`, landed on top of upstream's richer drawn items (icons, danger, trailing, content, max width). | One `openAt(rt, bounds)`. Upstream item rendering. `Theme.Menu` still styles the panel, rows, highlight, separator, and text size. `SetDrawMenus` / `DrawMenus` live here as the bool wrapper. |
| `ui/api_gen.go` | Generated wrappers for the two menu methods occupied one slot. | `go generate ./ui` after both methods exist on `*context`. Do not hand-merge this file. |
| `docs/ui/context-menu.md` | Fork documented `SetDrawMenus` and `Theme.Menu`. Upstream documented richer items and `SetMenuPresentation`. | Both sections. `SetDrawMenus` is described as the bool form of `SetMenuPresentation`. |
| `plugins/terminal/links.go` | Fork `#18`–`#21` (`LinkPath`, directories, non-ASCII URLs) and upstream `#1302` (Chinese prose, quotes, `(line,col)`, `#L`) each replaced path detection. | Upstream `pathAt`, which rewrites every place to `path:line:col`. Fork URL pattern (stop at the first non-ASCII). `~/`, `./`, and `../` stay paths. `isLink` applies `LinkPath`, or rejects a directory when `LinkPath` is nil. |
| `plugins/terminal/terminal_test.go` | Both sides appended tests at the end. | `TestDrawnMenu`, `TestProgress`, and `TestOnDrop` all kept. The two fork tests now live in `terminal_fork_test.go` so the next append does not land in the same hunk. |
| `docs/plugins/terminal.md` | Both sides extended the `OpenLink` path list. | Upstream places and Chinese prose, plus fork directories (`docs/`, `docs/:12`) and the `LinkPath` paragraph that already followed the conflict. |
| `updater_test.go` | Same append collision: fork `TestMakeExecutable` ([#14](https://github.com/si-mods/mygo/pull/14)), upstream `TestHTTPGetRetries` ([#1436](https://github.com/egoist/mygo/pull/1436)). | Both. `TestMakeExecutable` now lives in `updater_exec_test.go`. |

`ui/runtime.go` auto-merged with both menu engines still called. The fork
`drawer` path was removed so a frame cannot draw an editing menu twice.
SystemGlass fields in that file were left as #13 set them.

### Behavior that must keep working

- `Context.SetDrawMenus` / `DrawMenus`. gorex calls `SetDrawMenus`. The
  name stays until that caller can use `SetMenuPresentation`.
- `Theme.Menu` on menus drawn in the window, including ones opened by
  `SetMenuPresentation(PopoverMenus)`.
- Editing menus open at the point asked, including when the ask happens
  on the last pass of a frame, and only while the window draws menus.
- `Options.LinkPath`. Without it, a directory printed as text (`docs/`,
  `~/`, `docs/:12`) is not a link. `OpenLink` alone must not link every
  `pathAt` match: that drops `LinkPath` and would link prose such as `w/`.
- A URL printed as text ends at the first non-ASCII character ([#21](https://github.com/si-mods/mygo/pull/21)).
- `Options.OnProgress` (OSC 9;4) and `Options.OnDrop` both exist.
  `OnDrop` returning true pastes nothing.
- `ui.SystemGlass` stays under web views, over the window material. It is
  not `Glass{Native: true}`, which upstream places above web views.
- An element with `SystemGlass`, an opacity below 1, and children still
  punches the window. Upstream fade groups would paint that hole into a
  layer, so those elements stay flat (`isSystemGlass` in `ui/paint.go`).
- `makeExecutable`: an archive built on Windows gets exec bits from its
  read bits, including through a symlink. The updater still retries a
  gateway timeout.

### Tests

- `ui`: `TestDrawMenus*`, `TestMenuStyle*`, `TestSystemGlass*`,
  `TestMenuPresentation*`. `TestDrawMenusDrawerOnlyWhenUsed` now checks
  `drawsMenus` per window. `Tester.Menu` reports a drawn menu too, so
  tests that mean the system menu check `tt.h.menu`.
- `plugins/terminal`: `TestURLAtNotASCII`, `TestPathAt` (including `~/`,
  `./`, `../`), `TestIsDir`, `TestCommandClickDirs`, `links_place_test.go`,
  `TestDrawnMenu`, `TestProgress`, `TestOnDrop`.
- `TestMakeExecutable`, `TestHTTPGetRetries`.

### Hotspots

These files collided this time and will again if both sides keep editing
them in place:

- `ui/menu.go`, `ui/popovermenu.go`, `ui/api_gen.go`, `ui/runtime.go`
- `docs/ui/context-menu.md`
- `plugins/terminal/links.go`, `plugins/terminal/terminal.go`,
  `plugins/terminal/view.go`, `docs/plugins/terminal.md`
- `updater.go`, `updater_test.go`
- SystemGlass: `ui/systemglass.go`, `ui/paint.go`, `ui/runtime.go`,
  `internal/platform/surface.go`, `internal/darwin/surface.go`,
  `internal/darwin/glass_view.go`

The 2026-11-05 sync collided on the glass type names and on `menuAt`.
Renaming the fork glass types to `SystemGlass*` removed that collision
from this sync: upstream's `GlassPlacement` auto-merged. The menu
collision came back because the fork still had its own drawer. This sync
deletes that drawer and calls upstream's `menusInWindow` instead.

Smaller next time:

- Add fork-only tests in their own file (`terminal_fork_test.go`,
  `updater_exec_test.go`), not at the bottom of a file upstream also
  appends to.
- Regenerate `ui/api_gen.go`. Do not resolve it by hand.
- Keep fork behavior as a filter or a wrapper around the upstream
  function (`isLink` after `pathAt`, `SetDrawMenus` over
  `SetMenuPresentation`). Do not keep a second parser or a second menu
  engine.
- Leave SystemGlass on its own types and its own host methods
  (`supportsSystemGlass`, `placeSystemGlass`). Do not reuse
  `PlaceGlass`.

## Upstream candidates

Checked against `a0eaaaaf` on 2026-11-23. `git grep` of that tree and a
search of egoist/mygo issues and pull requests found no existing item
for `LinkPath`, `OnProgress`, `makeExecutable`, `Theme.Menu` /
`MenuStyle`, `ReadsClipboard`, `SystemGlass`, or `SetDrawMenus`.
Markdown autolinks are a different feature. Nothing was opened or
commented on egoist/mygo.

### Worth sending

1. **`Options.LinkPath`, and directories off by default.** Highest
   value, and still absent. `pathAt` stays upstream's. After it returns
   a path, `LinkPath` decides; with no callback, a trailing slash
   (`docs/`, `~/`, `docs/:12`, and `~/` `./` `../` alone) is not a link.
   Sending this removes the filter half of the `links.go` / `view.go`
   conflict. Tests: `TestCommandClickDirs`, `TestIsDir`, the directory
   rows of `TestPathAt`. Compatibility: `OpenLink` alone must not start
   linking every `pathAt` match.

2. **OSC 9;4 `Options.OnProgress`.** Still absent. State and percent,
   percent `-1` when absent, the last report in a burst. Test:
   `TestProgress` in `terminal_fork_test.go`. No overlap with `OnDrop`.

3. **`makeExecutable` on update.** Still absent. An archive produced on
   Windows has no exec bits. Set them from the read bits, following a
   symlink (`update.Perm`). Test: `TestMakeExecutable`. Sits next to
   upstream's retry work and does not replace it.

4. **Updater gateway budget of about half a minute.** Upstream
   `httpRetries` is still `1s`, `3s` (three tries). [#1436](https://github.com/egoist/mygo/pull/1436)
   is that short budget. [#875](https://github.com/egoist/mygo/pull/875)
   retries a stalled body, which this sync already contains. Neither
   covers a gateway that times out for some seconds. Fork budget: `1s`,
   `5s`, `20s`. Test: `TestHTTPGetRetries`. Low compatibility risk if
   the waits stay bounded.

5. **`Theme.Menu`.** Still absent. A theme-wide style for menus drawn
   in the window (panel function, row height, radius, padding,
   highlight, separator, text size). Zero parts keep the current look.
   The panel callback must receive the `*menuParts` nesting level.
   This is not [#1889](https://github.com/egoist/mygo/pull/1889): that
   `Panel` / `Item` pair styles one `PopoverMenu`. Sending `Theme.Menu`
   removes the `parts.level` miss. Tests: `TestMenuStyle*` in
   `ui/drawmenus_test.go`.

6. **Last pass of a frame still opens a drawn editing menu.** Still
   fork-only inside `openDrawnEditMenu`. Call `host.requestFrame` when
   `inFrame && pass >= 2`. Test: `TestDrawMenusEditMenuAsksForAFrame`.
   The position is already snapshotted by `openAt`. The toast fallback
   (`openSystemEditMenu` when the text input is not in the tree) is a
   separate choice; send it only if upstream wants that fallback.

7. **Terminal autolink and wrapped-path fixes.** Not in upstream's
   terminal package. [#1677](https://github.com/egoist/mygo/pull/1677)
   already opens a soft-wrapped URL as a whole, and
   [#2120](https://github.com/egoist/mygo/pull/2120) already links a
   Windows path as a whole from its drive or a UNC share. Do not send
   either again. The fork extras that remain are: end an OSC 8 autolink
   before the prose a program attached
   ([#34](https://github.com/si-mods/mygo/pull/34)–[#38](https://github.com/si-mods/mygo/pull/38)),
   join a path the program wrapped
   ([#40](https://github.com/si-mods/mygo/pull/40)), and trim full-width
   punctuation ([#28](https://github.com/si-mods/mygo/pull/28)).
   `pathAtRows` should stay a call to `pathAt` plus the `LinkPath`
   filter, so this can ship with item 1 or just after it. A drive path
   inside a larger token (`PATH=C:\bin;D:\bin`, `main.go:12:code`) is
   still the fork's `pathInToken` alternative; [#2120](https://github.com/egoist/mygo/pull/2120)
   does not split those. Tests: `TestPathAtRows`, `TestPathAtDrive`,
   `TestPathAtSpanCJKPunct`, `TestCommandClickWrappedPath`, the autolink
   prose tests. Sending this removes the `linkAt` conflict. A
   soft-wrapped path's underline is still the hovered row; the opened
   string is the whole path.

8. **`ReadsClipboard`.** Small export, still absent. Test:
   `TestReadsClipboard` in `headless_test.go`.

9. **Performable Command+C and Control+Shift+C.** Upstream already has
   `Performable` and performable Control+C on Windows. The fork also
   makes Command+C and Control+Shift+C performable, and still delivers
   Command+C to a program that copies its own selection. Tests:
   `TestPerformableCopyKeys`, `TestPerformableCommandKeyAlt`,
   `TestPerformableCopyKeepsSelection`. Small delta on the existing
   keybind, not a new command.

10. **Keep the cursor in sight when the screen is taller than the
    view** (`v.top`). Older than [#26](https://github.com/si-mods/mygo/pull/26),
    and it collided again with row reuse ([#1687](https://github.com/egoist/mygo/pull/1687)).
    Test: `TestViewShowsTheRowsAroundTheCursorOfATallerScreen`.
    Sending it removes the `paint.go` viewport conflict. It has to
    run after the cursor is known and before `updateRows` draws.

11. **libghostty-vt 1.0.1.** Upstream's pin is still `befcdfd` (the
    original terminal commit). The fork pin is ghostty `108cacb`
    (1.0.1), and the binaries are published on si-mods/mygo releases.
    Worth upstream having a newer pin, but an upstream pull request
    needs egoist to publish those builds. Not a drop-in patch. No
    dedicated 1.0.1 bump was found in egoist/mygo pull requests.

12. **X11 rounding of undecorated window corners**
    ([si-mods/mygo#42](https://github.com/si-mods/mygo/pull/42)).
    Upstream rounds macOS and Windows 11 itself
    ([#1849](https://github.com/egoist/mygo/pull/1849)) and clips a
    transparent Windows 10 frame
    ([#2176](https://github.com/egoist/mygo/pull/2176), `ContentCorners`
    / `roundFrame`). Neither cuts an X11 window. The fork does, with a
    compositor (alpha frame, `CornerSurface`) or without (an X shape),
    square while maximized, tiled or full screen, and not on Wayland or
    for a transparent window. Sending it removes the `windowCorners` /
    `roundFrame` collision in `ui/paint.go` and the Linux window files,
    if it stays a sibling of `ContentCorners` rather than a second copy
    of `roundFrame`. Tests: `internal/linux` `TestCorner`,
    `ui/corners_test.go`. Linux GUI e2e (`internal/e2e/corners_linux_test.go`)
    was not run here.

### Keep in the fork

- **`SetDrawMenus(bool)`.** Still a wrapper: `true` is
  `SetMenuPresentation(PopoverMenus)`. gorex calls `SetDrawMenus`.
  Do not ask upstream to drop the enum.
- **`ui.SystemGlass`.** Still a different feature from
  `Glass{Native: true}` and from `ViewSurface`: under web views, over
  the window material, on `supportsSystemGlass` / `placeSystemGlass` /
  `SystemGlassSurface`. `Window.CornerRadius` ([#1849](https://github.com/egoist/mygo/pull/1849))
  does not replace it. On Linux X11, `CornerRadius` is now the fork's
  rounding ([#42](https://github.com/si-mods/mygo/pull/42)), not 0; that
  is candidate 12, not a replacement for SystemGlass.
- **gorex call sites.** Not upstream's.

### Not a candidate

- **`OnInstalled`.** Already [egoist/mygo#1577](https://github.com/egoist/mygo/pull/1577).
  The bodies match. Upstream's recheck-while-offering is the extra, and
  this sync took it.
- **Glyphs shrunk to mid-cell.** Already [#1550](https://github.com/egoist/mygo/pull/1550).
- **Symbols in the text color, and Segoe UI Symbol.** Already
  [#1551](https://github.com/egoist/mygo/pull/1551), which also centers
  the glyph. The fork copies ([#26](https://github.com/si-mods/mygo/pull/26),
  [#27](https://github.com/si-mods/mygo/pull/27)) were dropped.
- **A URL ending at the first non-ASCII character.** Already
  [#1481](https://github.com/egoist/mygo/pull/1481). No fork-only copy.
- **A Windows path linked as a whole.** Already
  [#2120](https://github.com/egoist/mygo/pull/2120). This sync took it.
  The fork still adds `pathAtRows`, the `LinkPath` filter, and the
  drive alternative inside a larger token.
- **Clipping a transparent Windows 10 frame.** Already
  [#2176](https://github.com/egoist/mygo/pull/2176). This sync kept
  `roundFrame`. It does not round X11.
- Renaming or deleting the remaining wrappers to match upstream. That
  would break gorex (`SetDrawMenus`, `LinkPath`, `SystemGlass`).
