Fork of egoist/mygo through a0eaaaaf (releases 0.9.2 and 0.10.0), merged onto fork main 5895deae.

The module path stays `github.com/egoist/mygo`. `mygo.Version` is 0.10.1, the same version this tag carries. Consumers:

    replace github.com/egoist/mygo => github.com/si-mods/mygo v0.10.1

`go get github.com/si-mods/mygo@v0.10.1` fails Go's module-path check. The replace form resolves.

This tag does not publish mygo-runtime, the @mygo-plugins packages, or mygo-cli. Those npm names belong to egoist/mygo. `.github/workflows/release.yml` publishes them, and asks the Go module proxy for `github.com/egoist/mygo`, only when `github.repository` is `egoist/mygo`.

Fork-only behavior kept: X11 windowCorners, Windows roundFrame, Theme.Menu with parts.level, SetDrawMenus, SystemGlass, LinkPath, ReadsClipboard, ghostty pin 108cacb, updater makeExecutable and the 1s/5s/20s retries.

Upstream behavior now included: iOS and Android apps, a Windows path linked from its drive or UNC share, transparent Windows 10 corner clipping, PageOptions.ZoomControl off by default, a sideways terminal wheel reported to a program that takes the mouse, and the CLI passing the inherited environment to process.execve.

Non-GUI tests passed except `internal/nativegen` `TestScaffoldBuildsRust Zig`. rustc 1.83 warns that `struct NativeB is never constructed` on the clash stub, and the same warning fails on upstream a0eaaaaf. Zig 0.16.0 compiles the generated packages. The test was not skipped. GUI e2e on macOS, Windows and Linux was not run.
