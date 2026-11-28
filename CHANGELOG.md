# Changelog

Newest first. The server reads this file to update itself, so keep the headings as `## <version>`.
Add ` (server)` to the heading when the release changes the Go server (`main.go`, `update.go`):
those are pulled by hand (stop, `git pull`, `go run .` / `go build`). Releases without it only touch
`public/`, and the server pulls them on its own when it starts (or from the version button in the panel).

## 1.0.1
- OBS overlays pick up new banners and sponsors without restarting OBS

## 1.0.0 (server)
- Version tracker in the control panel's top bar, with automatic page updates from GitHub
- Info screen (banners + Markdown text + chat), boxed bottom score bar, rolling score digits
