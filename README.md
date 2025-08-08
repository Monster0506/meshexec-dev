# MeshExec

**MeshExec** is a Bluetooth-based mesh shell command runner for secure, distributed execution across nearby devices. It enables lightweight, ephemeral command sharing and execution even without Wi-Fi or internet -- ideal for ad-hoc collaboration, local automation, or field-based computing.

> 🔧 Works offline. ⚡ Runs fast. 🐝 Meshes dynamically.

---

## 🧠 Key Features

- 🔗 **Bluetooth Mesh Networking**  
  Dynamically connect with nearby devices using BLE advertisements and GATT servers.

- 📟 **Shell Command Distribution**  
  Broadcast shell commands to nodes in the mesh and gather outputs.

- 🧰 **CLI-First Design**  
  Powerful terminal experience with subcommands, filters, target selectors, and color-coded output.

- 🔐 **Secure by Design**  
  Pairing + identity handshake, sandboxing, and trust scopes per node or command.

- 🌐 **Cross-Platform Support**  
  Native Go bindings for Bluetooth work on **Linux**, **macOS**, and **Windows** (with caveats).

- 📦 **Small Footprint**  
  <10MB binary, portable, and dependency-light.

---

## 🚀 Quick Start

### 🧱 1. Install

**macOS / Linux**
```bash
git clone https://github.com/monster0506/meshexec.git
cd meshexec
go build -o meshexec ./cmd/meshexec
sudo ./meshexec daemon
````

**Windows (Experimental)**

```powershell
git clone https://github.com/monster0506/meshexec.git
cd meshexec
go build -o meshexec.exe ./cmd/meshexec
.\meshexec.exe daemon
```

> ⚠️ On Windows, you must enable Developer Mode and run in an elevated terminal.

---

### 💻 2. Run Your First Command

```bash
meshexec run --cmd "uptime"
```

You should see a list of nodes, followed by output like:

```
[raspi-zero: OK]  18:33:22 up 1 day, 4:12,  1 user,  load average: 0.00, 0.01, 0.05
[laptop: OK]      18:33:23 up 2 days, 7:45,  2 users, load average: 0.14, 0.10, 0.08
```

---

## ⚙️ Architecture Overview

* `daemon`
  Runs a Bluetooth GATT server for device discovery and command transfer.

* `meshexec run`
  Sends a broadcast or targeted command using BLE advertisement payloads.

* `meshexec status`
  Queries current mesh status, connected peers, and device fingerprints.

* `meshexec trust`
  Manage the node trust store (approve, revoke, scope, etc.)

* `meshexec log`
  View the latest received command history or output logs.

---

## 📡 Bluetooth Mesh Internals

MeshExec does not use standard Bluetooth Mesh profiles. Instead, it implements a minimal custom protocol using:

* BLE advertisements (fast broadcast)
  For presence announcements and command identifiers

* BLE GATT characteristics (low-latency unicast)
  For command payloads, ACKs, and output streams

* Optional mesh relaying (store-and-forward)
  Enables multihop routing in sparse topologies

---

## 🔒 Security Model

* 💠 Commands are signed with sender fingerprint
* ✅ Trust is managed per-node via approval flow
* 🪪 Each node has a persistent identity key
* 🧪 Sandbox modes (dry run, readonly) available

---

## 🛡️ Command Safety (Safe Mode)

Safe Mode prevents dangerous or destructive commands from being executed accidentally or maliciously.

What it enforces
- Max command length: rejects overly long inputs (configured via `safety.max_command_length`).
- Dangerous command blocking (OS‑aware):
  - Unix: patterns like `rm -rf`, `dd if=`, `mkfs`, `shutdown`, `poweroff`, recursive `chmod 000 /`, and loose fork‑bomb forms.
  - Windows: `del /s`, `rd /s /q`, `format`, `bcdedit`, `shutdown`, `cipher /w`, plus PowerShell cmdlets like `Remove-Item -Recurse -Force`.
- Wrapper detection: flags dangerous payloads passed via shells (e.g. `sh -c "rm -rf /"`, `powershell -Command "Remove-Item -Recurse"`).
- Customization: extend via `safety.dangerous_commands` (flexible whitespace is allowed between tokens).

Configuration (TOML)
```toml
[safety]
safe_mode = true               # enable/disable safety enforcement
max_command_length = 1024      # reject commands longer than this
dangerous_commands = [         # optional additions/overrides
  "shutdown",
  "format",
]
```

CLI usage
- Prefer preview: `mechexec run --dry-run -- <cmd>`
- Enforce explicitly: `mechexec run --safe-mode -- <cmd>`

Logging & visibility
- When a command is blocked, a warning is logged (pattern and reason). Increase verbosity with `-v`.

Notes & limitations
- Matching is defensive and token‑anchored but does not fully parse shell syntax. Extremely obfuscated inputs may still bypass; use dry‑run and reviews for critical environments.
- Patterns are OS‑aware; when executing on remote devices in future mesh modes, ensure the remote OS context is used.

---

## 🧪 Examples

| Command                                           | Description                            |
| ------------------------------------------------- | -------------------------------------- |
| `meshexec run --cmd "date"`                       | Run `date` on all visible nodes        |
| `meshexec run --cmd "ls /etc" --target device123` | Run only on a specific device          |
| `meshexec trust list`                             | List all approved nodes                |
| `meshexec log`                                    | View logs of received or sent commands |
| `meshexec status`                                 | Print mesh status and available peers  |
| `meshexec run --file ./script.sh`                 | Send and execute a script              |
| `meshexec daemon --port 9001`                     | Change GATT port if needed             |
| `meshexec run --timeout 3`                        | Fail nodes after 3s with no response   |
| `meshexec run --tag "dev"`                        | Target only nodes with tag `dev`       |

---

## 🧬 Roadmap

* [ ] Command chunking for long scripts
* [ ] Multi-hop routing via node relays
* [ ] Node tagging and auto-grouping
* [ ] GUI mesh visualizer
* [ ] File sync and remote copy support
* [ ] Rust FFI runtime support for embedded nodes

---

## 🤝 Contributing

We welcome your PRs, issues, and ideas! Contributions can include:

* Platform support testing
* Bluetooth reliability improvements
* Security enhancements
* UI/UX feedback (CLI ergonomics)
* Docs & tutorials

---

## 📜 License

MIT License. See [LICENSE](./LICENSE).

