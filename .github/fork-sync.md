# Fork sync log

Maintenance notes for the si-mods/mygo fork of egoist/mygo. This is not a
product document: `docs/` is published on the website, and a page added
there is picked up by the docs nav. Nothing here was sent to egoist/mygo.
The candidates at the end are an assessment only.

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

Checked against `659a325e`. A search of egoist/mygo issues and PRs for
`LinkPath`, `SetDrawMenus`, and `OnProgress` found no existing item.
No pull request was opened.

### Worth sending

1. **`Options.LinkPath`, and directories off by default.** Highest
   value. `pathAt` should stay upstream's. After it returns a path,
   `LinkPath` decides; with no callback, a trailing slash (`docs/`,
   `~/`, `docs/:12`, and `~/` `./` `../` alone) is not a link. gorex
   and fork `#18`–`#21` depend on this. Tests: `TestCommandClickDirs`,
   `TestIsDir`, the directory rows of `TestPathAt`, plus upstream
   `links_place_test.go`. Ship the non-ASCII URL boundary (`TestURLAtNotASCII`)
   in the same change only if upstream agrees a URL ends where CJK
   starts; percent-encoded non-ASCII already stays inside the URL.

2. **OSC 9;4 `Options.OnProgress`.** Small and separate. State and
   percent, percent `-1` when absent, the last report in a burst.
   Test: `TestProgress`. No overlap with `OnDrop`.

3. **`makeExecutable` on update.** An archive produced on Windows has
   no exec bits. Set them from the read bits, following a symlink.
   Test: `TestMakeExecutable`. Sits next to the retry added in `#1436`.

4. **`Theme.Menu`.** Optional style for menus drawn in the window
   (panel function, row height, radius, padding, highlight, separator,
   text size). Parts left at zero keep the current look. Tests:
   `TestMenuStyle*` in `ui/drawmenus_test.go`.

5. **Last pass of a frame still opens a drawn editing menu.** Upstream
   sets `consumed` inside `openDrawnEditMenu`. The frame loop stops at
   pass 2, and `requestFrame` does nothing while the frame runs, so an
   ask on that pass waits for an unrelated event. Call
   `host.requestFrame` when `inFrame && pass >= 2`. Test:
   `TestDrawMenusEditMenuAsksForAFrame`. The position is already
   snapshotted by `openAt`; the fork test
   `TestDrawMenusEditMenuKeepsWhereAsked` covers a later ask not moving
   it. Sending this only pays off if upstream wants the system-menu
   fallback for a text input that is not in the tree yet (a toast).
   That fallback is `openSystemEditMenu` from `drawEditMenu`.

### Keep in the fork

- **`SetDrawMenus(bool)`.** Upstream `#1359` replaced the bool with
  `MenuPresentation` on purpose. The fork method is a one-way wrapper so
  existing callers compile. Propose it only as a deprecated alias, or
  migrate callers to `SetMenuPresentation`. Do not ask upstream to drop
  the enum.
- **`ui.SystemGlass`.** It is a different feature from
  `Glass{Native: true}`: under web views, over the window material,
  with `Style`, `Tint`, `Fallback`, and `Opacity`. Upstream native glass
  is above web views and keyed by element id. Contributing it by
  overloading `PlaceGlass` would break one of the two. The fade-group
  exception in `paint.go` exists only because SystemGlass exists.
- **gorex call sites.** Not upstream's, and not changed from this
  repository.

### Not a candidate

Renaming or deleting the fork wrappers in order to match upstream
exactly. That would break gorex (`SetDrawMenus`, `LinkPath`,
`SystemGlass`) for a smaller diff. The compat layer above is the
reduction that does not drop behavior.
