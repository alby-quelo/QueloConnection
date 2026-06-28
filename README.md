# Quelo Connect (nossh)

**English** · **[Italiano](README.it.md)**

**Outbound SSH bridge for machines without a public IP.**

Quelo Connect lets you reach Linux and Windows hosts over SSH without opening inbound port 22 on remote networks and without a public IP on each machine. Remote **agents** connect *out* to a central **bridge server**; **clients** and an optional **web admin panel** connect *in* to the bridge.

No remote desktop (TeamViewer, RustDesk, etc.) — only SSH (shell access).

---

## How it works

```
[Remote machine]  --agent (outbound)-->  [Bridge server]  <--client--  [User SSH]
        |                                        |
   local OpenSSH                           forwards SSH
   (127.0.0.1 on Windows)                  to the agent
```

1. Each remote machine runs **nossh-agent** and dials the bridge (default port **4443**).
2. The bridge admin assigns a **friendly name** to each agent.
3. Users connect with the CLI/GUI client or the web panel:  
   `nossh connect machine-name username`

Remote machines do **not** need router port forwarding for SSH.

---

## Repository layout

| Folder | Purpose |
|--------|---------|
| [`QueloConnection/`](QueloConnection/) | Public release: prebuilt binaries, install scripts, optional GUI `.deb` |
| [`SOURCE_CODE/`](SOURCE_CODE/) | Full Go source (server, agent, client, GUI) |
| [`web/`](web/) | PHP admin panel (agent management + in-browser SSH terminal) |
| [`windows-agent/`](windows-agent/) | Windows agent: `ESEGUIBILI/` + source to rebuild `nossh-agent.exe` |
| [`windows-client/`](windows-client/) | Windows client: portable binaries + source |
| [`LICENSE`](LICENSE) | Custom non-commercial license |
| [`LEGGIMI.txt`](LEGGIMI.txt) | Overview and quick start (Italian) |

---

## Bridge server ports

| Port | Direction | Service |
|------|-----------|---------|
| **4443** | Inbound on VPS | Agent connections |
| **7000** | Inbound on VPS | Client SSH proxy |
| **8081** | Localhost only | Admin API (used by web panel) |

---

## Quick start

### 1. Bridge server (VPS)

Install from [`QueloConnection/`](QueloConnection/) — see **[`QueloConnection/LEGGIMI.txt`](QueloConnection/LEGGIMI.txt)** for full steps (systemd, firewall, tokens).

```bash
cd QueloConnection/scripts
sudo ./install-server.sh
```

### 2. Remote agent (Linux)

```bash
sudo ./install-agent.sh
```

Note the **agent code** and assign a name on the bridge:

```bash
nossh name AGENT-CODE my-machine
```

### 3. Remote agent (Windows)

Copy [`windows-agent/ESEGUIBILI/`](windows-agent/ESEGUIBILI/) to the PC and run **`install-agent.bat`** as Administrator.  
See **[`windows-agent/LEGGIMI.txt`](windows-agent/LEGGIMI.txt)**.

### 4. Connect (CLI)

```bash
nossh connect my-machine username
```

### 5. Web admin panel (optional)

Deploy [`web/`](web/) on the same host as `nossh-server` (Apache/nginx + PHP).  
See **[`web/LEGGIMI.txt`](web/LEGGIMI.txt)**.

Features: login with admin token, list/rename/revoke agents, **Connect** button with in-browser terminal (xterm.js + WebSocket).

### 6. Windows client

Use [`windows-client/portable_client/`](windows-client/portable_client/) or build from [`windows-client/SOURCE_CODE/`](windows-client/SOURCE_CODE/).

---

## Requirements

**Bridge (VPS)**  
Linux amd64/arm64, systemd recommended, inbound TCP **4443** and **7000**.

**Linux remote**  
sshd on port 22, outbound access to bridge **4443**.

**Windows remote**  
Windows 10/11 or Server; installer sets up OpenSSH on **127.0.0.1** and the `nossh-agent` Windows service.

**Client**  
OpenSSH client (`ssh`); outbound access to bridge **7000**.

---

## Building from source

Full tree: [`SOURCE_CODE/`](SOURCE_CODE/)  
Windows agent only: [`windows-agent/SOURCE_CODE/`](windows-agent/SOURCE_CODE/) — `make agent` or `scripts/pack-windows-agent.sh`

Requires Go 1.21+ (see `go.mod` in each tree).

---

## Security notes

- Keep `install_token` and `admin_token` secret; do not commit real tokens.
- Use strong passwords on SSH accounts; prefer dedicated non-admin users where possible.
- On Windows, OpenSSH listens on **127.0.0.1** only; access is via the bridge tunnel.
- The web panel should be served over HTTPS in production.

---

## License

**Custom non-commercial license** — see [`LICENSE`](LICENSE).

Free for personal, educational, and non-commercial use. Modification and redistribution allowed with attribution. **Commercial use requires written permission** from the author.

---

## Author

**Alberto Frosio**  
Email: [alby@gnumerica.org](mailto:alby@gnumerica.org)

Commercial licensing: contact the author at the email above.
