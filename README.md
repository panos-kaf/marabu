# Marabu Node

A highly concurrent, full-node implementation of the Marabu peer-to-peer Proof-of-Work blockchain.

This project is built in Go and focuses on a strict separation of concerns, isolating network routing, consensus evaluation, and database state management. It features a built-in multi-core miner, a fully featured UTXO wallet, a dynamic topology manager, and a rich Text User Interface (TUI) designed for terminal-native environments.

---

## Architecture

The repository is modularized into distinct domains to ensure thread safety, maintainability, and architectural boundaries. Unvalidated network data is strictly isolated from the database state engine.

### 1. Core State & Consensus (`internal/core`)
The central state machine and gatekeeper of the node.
* **`manager.go`:** The public Façade. It bridges the network layer and the database, ensuring no unvalidated objects are persisted. It exposes thread-safe accessors for chaintip height, active mining cores, and session telemetry.
* **`validation.go`:** The consensus engine. Verifies Proof of Work (PoW) targets, validates Ed25519 digital signatures, enforces block timestamp chronologies, and ensures UTXO conservation and strict Coinbase reward limits.
* **`db.go`:** A thread-safe wrapper around LevelDB. It handles raw byte serialization, mempool tracking, and the complex logic required for chain-reorganizations when a heavier proof-of-work fork is discovered.
* **`miner.go`:** Orchestrates concurrent mining workers across allocated CPU cores. Constructs block templates, calculates live hashes-per-second, and cleanly interrupts workers the moment a competing network block is discovered.

### 2. Network & P2P Layer (`internal/peer`)
Manages asynchronous TCP socket connections, protocol gossiping, and spam mitigation.
* **`connection_manager.go`:** A robust, mutex-locked registry tracking all active TCP sessions. Differentiates between inbound, outbound, and persistent (VIP) bootstrap nodes.
* **`peer.go` / `handlers.go`:** Manages individual connection lifecycles and read buffers. Translates incoming JSON streams into protocol actions, responding to `getobjects` requests or rejecting malformed payloads.
* **`topology.go`:** The network watchdog. Autonomously replenishes dropped outbound connections, churns stale peers to ensure network decentralization, and tracks missing block dependencies.
* **`moderation.go`:** Thread-safe banning and muting logic. Drops connections and ignores traffic from abusive IPs or specific agent names to protect node resources.

### 3. Data Structures & Cryptography (`internal/types`, `internal/protocol`, `internal/crypto`)
The domain entities and wire formatting rules.
* **`internal/types`:** Defines primary domain models like `Block`, `Transaction`, `UTXOSet`, `Picabu`, and `HashID`. Implements custom unmarshalers to enforce architectural bounds at the parsing level (e.g., ensuring HashIDs are exactly 64-character hexadecimal strings).
* **`internal/protocol`:** Uses a dynamic type registry to parse raw TCP streams into typed structs (`Hello`, `GetObject`, `IHaveObject`).
* **`internal/crypto`:** A unified interface wrapping standard cryptographic primitives. Handles `blake2s` hashing for PoW verification and object IDs, and `ed25519` for wallet key generation and transaction signatures.
* **`internal/serialization`:** Implements recursive JSON canonicalization. Keys are sorted alphabetically and whitespace is stripped, guaranteeing deterministic byte sequences and stable cryptographic hashes across the network.

### 4. Node Tooling & Utilities (`internal/wallet`, `internal/discovery`, `internal/bootstrap`)
* **`internal/wallet`:** A built-in UTXO manager. Resolves human-readable aliases to public keys, tracks confirmed balances via the active UTXO set, and constructs/signs outbound transactions. Includes an auto-aliasing "sweep" function for discovering new network participants.
* **`internal/discovery`:** Manages the routing table. Persists known active peers to `peers.csv` and provides randomized peer selection for outgoing network connections.
* **`internal/bootstrap`:** Controls the node's startup behavior using Go build tags, determining whether the node connects to hardcoded seed nodes or starts in complete isolation.

### 5. Interface & Presentation (`internal/ui`, `internal/logs`)
The interactive layer supporting multiple deployment environments.
* **`internal/ui/tui.go`:** A rich, split-pane terminal dashboard powered by `tview`. Features a live-updating node status header, a native scrollable viewport for logs, command history (Up/Down arrow navigation), and a `Ctrl+F` screen-freeze feature for easy text copying.
* **`internal/ui/console.go`:** A classic, scrolling standard-input CLI using pure ANSI escape codes and `io.Writer` abstractions.
* **`internal/ui/commands.go`:** A unified command router shared between the TUI and CLI. Utilizes `tview.ANSIWriter` to seamlessly translate standard terminal color codes into the rich dashboard interface.
* **`internal/logs`:** A multi-target logging system supporting terminal colorization and file rotation, dynamically adjusting its output formatting based on the active build tag.

---

## Build System & Compilation

The project utilizes GNU Make. You can compile any combination of Network Topology and UI Mode.

### Build Matrix

**Network Topologies:**
* `standard`: Connects to seed nodes and actively discovers peers.
* `no-bootstrap`: Starts completely isolated.
* `bootstrap-only`: Connects strictly to seed nodes without peer exchange.

**UI Modes:**
* `tui`: Rich terminal dashboard.
* `cli`: Standard command-line prompt.
* `headless`: Runs quietly in the background as a daemon.

### Standard Commands

Compile the primary node:
```bash
make standard-tui
```

Compile a specific matrix combination:
```bash
make no-bootstrap-cli
```

Compile all 9 combinations simultaneously:
```bash
make build
```

---

## Usage & Execution

### Launcher Script
A provided `launcher.sh` script automates execution within a `kitty` terminal environment. It prompts the user for their desired UI and Topology modes, clears old log states, and cleanly forks a secondary terminal window dedicated purely to raw `tail -f` log outputs if running in CLI or Headless mode.

### TUI Commands Reference
Once inside the node environment, type `help` for a full list of commands. Key operations include:

**Hardware & Identity:**
* `info`: View network diagnostic counts, chaintip status, and active hashrate.
* `cores <num>`: Hot-swap the number of active CPU threads dedicated to mining.
* `note set <msg>`: Update the public miner string embedded in your blocks.

**Network Operations:**
* `peers`: View a formatted table of all active inbound, outbound, and VIP connections.
* `connect <ip:port>` / `disconnect <ip:port>`: Manually control the topology.
* `ban <target>` / `mute <target>`: Apply strict connection filtering.

**Database & Wallet:**
* `objects`: Audit the local LevelDB instance for blocks, transactions, and corrupted files.
* `balance`: Check confirmed UTXO balances.
* `send <alias> <amount>`: Broadcast a signed transaction to the network.

---

## Configuration

The node expects a `.env` file in the execution directory to configure basic runtime parameters.

```env
PORT=18018
AGENT=MarabuNode
STUDENT_IDS=0001742
```
