# Quelo Connect (nossh)

**[English](README.md)** · **Italiano**

**Ponte SSH in uscita per macchine senza IP pubblico.**

Quelo Connect permette di raggiungere host Linux e Windows via SSH **senza aprire la porta 22 in ingresso** sulle reti remote e **senza IP pubblico** su ogni macchina. Gli **agent** remoti si connettono *in uscita* al **server ponte**; **client** e **pannello web admin** (opzionale) si connettono *al ponte*.

Niente desktop remoto (TeamViewer, RustDesk, ecc.) — solo SSH (shell).

---

## Come funziona

```
[Macchina remota]  --agent (uscita)-->  [Server ponte]  <--client--  [Utente SSH]
        |                                      |
   OpenSSH locale                         inoltra SSH
   (127.0.0.1 su Windows)                 verso l'agent
```

1. Ogni macchina remota esegue **nossh-agent** e si collega al ponte (porta predefinita **4443**).
2. L'amministratore del ponte assegna un **nome** a ogni agent.
3. Gli utenti si connettono con client CLI/GUI o pannello web:  
   `nossh connect nome-macchina utente`

Le macchine remote **non** richiedono port forwarding SSH sul router.

---

## Struttura del repository

| Cartella | Contenuto |
|----------|-----------|
| [`QueloConnection/`](QueloConnection/) | Release pubblica: binari, script install, GUI `.deb` opzionale |
| [`SOURCE_CODE/`](SOURCE_CODE/) | Sorgente Go completa (server, agent, client, GUI) |
| [`web/`](web/) | Pannello admin PHP (gestione agent + terminale SSH nel browser) |
| [`windows-agent/`](windows-agent/) | Agent Windows: `ESEGUIBILI/` + sorgente per ricompilare `nossh-agent.exe` |
| [`windows-client/`](windows-client/) | Client Windows: binari portabili + sorgente |
| [`LICENSE`](LICENSE) | Licenza custom non commerciale |
| [`LEGGIMI.txt`](LEGGIMI.txt) | Panoramica e avvio rapido (italiano) |

---

## Porte sul server ponte

| Porta | Direzione | Servizio |
|-------|-----------|----------|
| **4443** | Ingresso sul VPS | Connessioni agent |
| **7000** | Ingresso sul VPS | Proxy SSH client |
| **8081** | Solo localhost | API admin (usata dal pannello web) |

---

## Avvio rapido

### 1. Server ponte (VPS)

Installa da [`QueloConnection/`](QueloConnection/) — vedi **[`QueloConnection/LEGGIMI.txt`](QueloConnection/LEGGIMI.txt)** (systemd, firewall, token).

```bash
cd QueloConnection/scripts
sudo ./install-server.sh
```

### 2. Agent remoto (Linux)

```bash
sudo ./install-agent.sh
```

Annota il **codice agent** e assegna un nome sul ponte:

```bash
nossh name CODICE-AGENT nome-macchina
```

### 3. Agent remoto (Windows)

Copia [`windows-agent/ESEGUIBILI/`](windows-agent/ESEGUIBILI/) sul PC ed esegui **`install-agent.bat`** come Amministratore.  
Vedi **[`windows-agent/LEGGIMI.txt`](windows-agent/LEGGIMI.txt)**.

### 4. Connessione (CLI)

```bash
nossh connect nome-macchina utente
```

### 5. Pannello web admin (opzionale)

Deploy di [`web/`](web/) sullo stesso host di `nossh-server` (Apache/nginx + PHP).  
Vedi **[`web/LEGGIMI.txt`](web/LEGGIMI.txt)**.

Funzioni: login con token admin, elenco/rinomina/revoca agent, pulsante **CONNETTI** con terminale nel browser (xterm.js + WebSocket).

### 6. Client Windows

Usa [`windows-client/portable_client/`](windows-client/portable_client/) oppure compila da [`windows-client/SOURCE_CODE/`](windows-client/SOURCE_CODE/).

---

## Requisiti

**Ponte (VPS)**  
Linux amd64/arm64, systemd consigliato, TCP in ingresso **4443** e **7000**.

**Remoto Linux**  
sshd sulla porta 22, uscita verso il ponte sulla **4443**.

**Remoto Windows**  
Windows 10/11 o Server; l'installer configura OpenSSH su **127.0.0.1** e il servizio Windows `nossh-agent`.

**Client**  
Client OpenSSH (`ssh`); uscita verso il ponte sulla **7000**.

---

## Compilare dai sorgenti

Albero completo: [`SOURCE_CODE/`](SOURCE_CODE/)  
Solo agent Windows: [`windows-agent/SOURCE_CODE/`](windows-agent/SOURCE_CODE/) — `make agent` o `scripts/pack-windows-agent.sh`

Richiede Go 1.21+ (vedi `go.mod` in ogni albero).

---

## Sicurezza

- Non condividere `install_token` e `admin_token`; non committare token reali.
- Password robuste sugli account SSH; preferire utenti dedicati non admin dove possibile.
- Su Windows, OpenSSH ascolta solo su **127.0.0.1**; l'accesso passa dal tunnel del ponte.
- In produzione servire il pannello web su **HTTPS**.

---

## Licenza

**Licenza custom non commerciale** — vedi [`LICENSE`](LICENSE).

Uso libero per scopi personali, educativi e non commerciali. Modifica e ridistribuzione consentite con attribuzione. **Uso commerciale solo con autorizzazione scritta** dell'autore.

---

## Autore

**Alberto Frosio**  
Email: [alby@gnumerica.org](mailto:alby@gnumerica.org)

Licenze commerciali: contattare l'autore all'indirizzo sopra.
