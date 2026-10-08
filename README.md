# Piggy Bank

> **macOS · Remote Host Control · WSL Monitoring · Lab Automation**

![macOS](https://img.shields.io/badge/macOS-000000?style=flat-square&logo=apple&logoColor=white)
![Go](https://img.shields.io/badge/Language-Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![Gio UI](https://img.shields.io/badge/UI-Gio-2A2A2A?style=flat-square)
![Status In Development](https://img.shields.io/badge/Status-In%20Development-ffcc00?style=flat-square)

## Project Overview

Piggy Bank is a macOS menubar utility designed to manage a laboratory workstation or remote server environment from a lightweight tray interface. It monitors the availability of a host machine, detects running WSL distributions, exposes quick power controls, and opens a local management window for status and actions.

The application is built with Go and Gio, and it uses a native macOS accessory-style window so it can stay out of the Dock while remaining usable as a focused desktop tool.

## Project Structure

```text
piggy-bank/
├── assets/                              # Embedded application assets
├── cmd/                                 # Command and host logic
│   ├── execute_commands/
│   │   └── executor.go                  # Shell command execution wrapper
│   └── host/
│       └── server.go                    # Remote host, WSL, and power actions
├── config/                              # Local configuration helpers
│   └── config.go                        # Reads ~/.piggy_bank/config
├── docs/                                # Internal notes and implementation docs
│   └── cgo.md                           # macOS Cocoa runtime notes
├── platform/                            # OS-specific integrations
│   └── darwin/
│       └── launch_agent.go              # Login item / startup registration
├── ui/                                  # UI, state, and tray components
│   ├── components/
│   │   ├── icons.go                     # Material Design icons
│   │   └── widgets.go                   # Buttons, nav items, cards, status halo, toast
│   ├── layouts/
│   │   └── app_shell.go                 # Sidebar + page routing (LocalSend-style)
│   ├── pages/                           # Host, WSL, Settings, History, About pages
│   ├── state/
│   │   ├── activity.go                  # Activity history and toast messages
│   │   ├── power.go                     # Host and WSL power actions
│   │   └── state.go                     # Host and WSL state management
│   ├── theme/
│   │   └── theme.go                     # Color palette and Gio theme
│   └── tray/
│       └── tray_icon.go                 # Systray menu and status toggles
├── test/                                # Utility test programs
│   ├── executor/
│   └── server_check/
├── center_mac.h                         # macOS window centering hook header
├── center_mac.m                         # Native Objective-C/Cocoa hook
├── go.mod                               # Go module definition
├── go.sum                               # Dependency lock file
├── LICENSE                              # MIT license
├── main.go                              # App entry point and macOS window setup
├── Makefile                             # Build and run convenience commands
├── PiggyBank.app/                        # Bundled macOS app package
├── target/                              # Build output directory
├── vendor/                              # Vendored Go dependencies
└── README.md                            # Project documentation
```

---

## 🔲 To do tasks

- Add a richer host dashboard view
- Add configurable remote actions per server
- Improve WSL lifecycle controls
- Add command history and logs panel
- Add more host health metrics and latency charts
- Improve configuration validation and setup wizard
- Add secure secret handling for production environments

## Completed

- [x] Added a macOS tray app with quick access controls
- [x] Added startup-on-login support for macOS
- [x] Added host reachability checks via ping
- [x] Added WSL discovery for active Linux instances
- [x] Added remote power controls for a Windows host
- [x] Added a Gio-based local management interface

## Features

### Host monitoring

The app periodically pings the configured host address and tracks whether it is online, along with latency information. This is used to refresh the tray menu and window status in real time.

### WSL discovery

The app runs a remote command over SSH to inspect currently running WSL distributions and displays them in the interface. This allows the user to understand the state of the local development environment without leaving the desktop.

### Remote wake and shutdown actions

The host supports actions such as:

- turning the host on via Telegram bot command
- shutting the host down via SSH to the remote Windows machine

These actions are wired through the tray menu and internal state transitions.

### macOS integration

The app uses Objective-C hooks to:

- force the application to run as an accessory app
- keep the window visible without a Dock entry
- center the window on screen when opened
- raise and activate the app reliably on macOS

## Commands

### Run locally

```bash
make run
```

### Build binary

```bash
make build
```

### Build and start

```bash
make start
```

### Environment configuration

The app reads settings from the user config directory at `~/.piggy_bank/config`. Add environment variables such as:

```bash
export TELEGRAM_BOT_TOKEN="your_telegram_token"
```

The host logic expects this token to be available for remote wake operations.

## Notes

- The app is currently focused on a macOS desktop workflow.
- Host control actions depend on SSH access and remote environment setup.
- The project is designed for a lab or home-server environment rather than a multi-tenant cloud deployment.
- The app intentionally keeps a minimal, lightweight UI while exposing high-value automation functions.

## License

See the [MIT License](LICENSE) file.
