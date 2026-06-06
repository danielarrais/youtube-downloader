# README

## About

This is the official Wails React-TS template.

You can configure the project by editing `wails.json`. More information about the project settings can be found
here: https://wails.io/docs/reference/project-config

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.

## Windows Installer

The `Build Windows` GitHub Actions workflow creates a single Windows x64
installer containing:

- The Wails application.
- `ffmpeg.exe`.
- The WebView2 bootstrapper.

Run the workflow manually from the Actions tab, or push a tag such as
`v1.0.0`. Download the `yt-mp3-go-windows-amd64` artifact when the workflow
finishes.

WebView2 is already present on Windows 11 and most Windows 10 installations.
If it is missing, the bundled bootstrapper installs it silently and requires an
internet connection.

### Build Windows from Linux

Docker is the only host dependency:

```bash
./scripts/build-windows.sh
```

The Docker build downloads Go modules, Wails, Node dependencies, FFmpeg and all
Windows build tools. The resulting installer is written to
`build/windows/dist/`.
