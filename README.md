# Marabu Protocol Node

This repository provides a full node implementation for the Marabu peer-to-peer network. The architecture utilizes strict separation of concerns to isolate network routing, consensus evaluation, and database state management.


## Package Breakdown

### 1. `internal/core`
This package serves as the central state engine for the blockchain. It encapsulates the core consensus logic, manages the Unspent Transaction Output (UTXO) set, and controls all database interactions to ensure state consistency.
* **`manager.go`:** The public Façade. It defines the strict boundary between the network layer and the database, ensuring no unvalidated data is persisted to disk.
* **`validation.go`:** The consensus evaluation engine. It verifies Proof of Work (PoW), validates digital signatures, checks block timestamps, and enforces UTXO conservation without applying side-effects.
* **`db.go`:** A private wrapper around LevelDB. It handles raw byte serialization, thread-safe pending block management via mutexes, and strict CRUD operations.

### 2. `internal/peer`
This package implements the network layer. It manages all TCP socket connections, handles asynchronous I/O operations, and executes the Marabu peer-to-peer gossip protocols.
* **`peer.go`:** Manages individual connection lifecycles and read buffers, routing incoming raw text to the unmarshaling functions.
* **`handlers.go`:** Defines the controllers for network messages, specifying how the node responds to protocol commands (e.g., serving requested blocks or rejecting requests for unknown objects).
* **`objects.go`:** Orchestrates the object processing pipeline: deduplication, validation requests to the core package, state commits, and network gossiping (`ihaveobject`).
* **`sendMessages.go`:** Contains network broadcast functions and targeted TCP write operations.

### 3. `internal/protocol`
This package governs wire formatting and data serialization. It translates internal Go data structures to and from the strict JSON schema required by the Marabu network protocol.
* **`messages.go`:** Defines the JSON structures for network-specific payloads (e.g., Hello, GetObject, Mempool).
* **`unmarshal.go`:** Implements a dynamic type registry to parse raw incoming TCP streams into typed interface objects, protecting against malformed payloads.
* **`constructors.go`:** Provides helper functions to build outgoing messages using canonical JSON formatting.

### 4. `internal/types`
This package acts as the central repository for the application's domain entities. It contains all structural definitions, constants, and custom unmarshaling logic required to enforce data integrity.
* **`types.go` / `objects.go`:** Defines primary domain models such as `Block`, `Transaction`, and `UTXOSet`, along with network constants like `TARGET` and `DUMMY_HASH`.
* **`unmarshal.go`:** Contains strict custom unmarshalers that enforce low-level architectural rules (e.g., validating that HashIDs are exactly 64 hexadecimal characters and ensuring correct IPv4/IPv6 peer address formats).
* **`messages.go`:** Enumerates all valid wire commands and standard `ErrorCode` constants.

### 5. `internal/crypto`
This package provides a unified interface for all cryptographic operations required by the protocol, acting as a wrapper around standard cryptographic primitives.
* **`blake2s.go`:** Computes object hashes and verifies Proof of Work against the defined network difficulty target.
* **`ed25519.go`:** Manages digital signature generation and verification for transaction inputs.

### 6. `internal/serialization`
This package ensures deterministic data representation, which is a critical requirement for generating consistent cryptographic hashes across the decentralized network.
* **`canonicalize.go`:** Implements a recursive algorithm to sort JSON keys alphabetically and strip whitespace. This guarantees that identical objects produce the exact same byte sequence, ensuring stable hashing regardless of the originating system.

### 7. `internal/discovery` & `internal/bootstrap`
These packages manage the network topology, peer routing table, and the node's startup behavior regarding initial peer connections.
* **`discovery.go`:** Manages the routing table by persisting known peers to `peers.csv` and selecting random peers for outgoing connections.
* **`bootstrap/`:** Utilizes Go build tags (`standard` vs `no_bootstrap`) to dictate whether the node attempts to connect to hardcoded Marabu bootstrap servers upon initialization.

### 8. `internal/logs`, `internal/ui` & `internal/utils`
These utility packages provide the underlying infrastructure for node operation, including monitoring, user interaction, and data conversion.
* **`logs/`:** A custom logging implementation that supports terminal colorization, multi-writer output (file and standard output), and conditional formatting based on build tags (`cli` vs `headless`).
* **`ui/`:** Offers either an interactive terminal prompt or a blocking routine for background server execution.
* **`utils/`:** Provides utility functions, such as mathematical conversions between whole `Picabu` units and fractional `Bu` values.
