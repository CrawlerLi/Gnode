# Layer Boundaries

This document defines the intended package boundaries for Gnode. It is a
working architecture rulebook, not a fixed framework. When the code grows, keep
new types and functions consistent with these boundaries unless there is a
clear reason to change the rule.

## Package Roles

### `cmd`

The CLI layer.

Responsibilities:

- Parse user commands and arguments.
- Load the active node config path.
- Call `service` or `node`.
- Print user-facing output.
- Handle process-level concerns such as `os.Signal`.

Should not:

- Implement blockchain validation rules.
- Directly manipulate block or wallet storage details.
- Return or expose protobuf DTOs to users.

### `internal/config`

Configuration loading and saving.

Responsibilities:

- Read and validate node config files.
- Read and write CLI config files.
- Store the active node config path.

Should not:

- Open databases.
- Start nodes.
- Call services.

### `internal/core`

Core blockchain domain.

Responsibilities:

- Define blockchain domain types such as `Block`, `Transaction`, `UTXOSet`,
  `BestState`, and `BlockChain`.
- Validate transactions and blocks.
- Update best state and UTXO state.
- Read and write chain data through the database abstraction.

Allowed dependencies:

- `internal/infra/database`
- Low-level utility packages such as `pkg/utils`

Should not depend on:

- `internal/service`
- `internal/node`
- `internal/p2p`
- `cmd`

Recommended domain types:

```go
type ChainBlock struct {
	Height int
	Block  *Block
}
```

`ChainBlock` means "a block in the current main-chain view". It should not
contain peer address, remote node ID, protobuf fields, or CLI formatting data.

### `internal/wallet`

Wallet model and wallet persistence.

Responsibilities:

- Define wallet data structures.
- Store and load wallets.

Should not:

- Own blockchain synchronization logic.
- Start network servers.

### `internal/service`

Application service layer.

Responsibilities:

- Coordinate core and wallet operations.
- Provide application use cases such as wallet creation, transfer, balance
  query, chain info query, and chain state query.
- Return service DTOs for callers such as `cmd`, `node`, and future control RPC.

Allowed dependencies:

- `internal/core`
- `internal/wallet`
- `internal/infra/database`
- `pkg/...`

Should not depend on:

- `internal/p2p`
- `internal/node`
- `cmd`

Service DTO examples:

```go
type ChainState struct {
	Height   int
	LastHash []byte
}

type ChainInfo struct {
	Height   int
	LastHash []byte
	Blocks   []core.ChainBlock
}
```

Rules:

- Service DTOs may reference core types when they represent application results.
- Service DTOs should not use protobuf types.
- Service methods should not print user-facing output.

### `internal/node`

Long-running node runtime.

Responsibilities:

- Own the running node lifecycle.
- Start and stop P2P server.
- Manage connected peers.
- Add peer context to P2P responses.
- Call `service` for local blockchain and wallet capabilities.
- Eventually coordinate periodic sync tasks.

Allowed dependencies:

- `internal/service`
- `internal/p2p`
- `internal/core` when node-level DTOs need domain types

Should not:

- Directly open databases.
- Implement low-level gRPC handlers.
- Put peer/network fields into core types.

Node DTO examples:

```go
type PeerChainState struct {
	PeerAddr     string
	RemoteNodeID string
	Height       int
	LastHash     []byte
}

type PeerBlocks struct {
	PeerAddr     string
	RemoteNodeID string
	Blocks       []core.ChainBlock
}
```

Rules:

- Node DTOs may include peer address and remote node ID.
- Node DTOs should not be stored in core or service.
- Node should convert P2P responses into node-level DTOs before giving them to
  higher-level sync logic.

### `internal/p2p`

Network transport layer for peer-to-peer communication.

Responsibilities:

- Implement gRPC server and client.
- Convert between protobuf DTOs and transport-facing data.
- Expose callbacks/providers so node/service can supply local data.

Allowed dependencies:

- `internal/p2p/proto`
- Standard library networking packages
- gRPC packages

Preferred rule:

- Avoid depending on `internal/service` directly.
- Avoid opening databases directly.

Short-term exception:

- It is acceptable for `p2p` to import `internal/core` for simple conversion
  with `core.ChainBlock` while the project is small. If this starts making P2P
  aware of core storage rules, replace it with a transport DTO such as:

```go
type BlockPayload struct {
	Height int
	Data   []byte
}
```

### `internal/p2p/proto`

Generated protobuf package.

Responsibilities:

- Contain generated request and response DTOs.
- Represent the wire format between nodes.

Rules:

- Proto DTOs should stay at the P2P boundary.
- `core`, `service`, and `wallet` should not depend on generated proto types.

## Dependency Direction

Preferred dependency flow:

```text
cmd
  -> config
  -> service
  -> node

node
  -> service
  -> p2p

service
  -> core
  -> wallet
  -> infra/database

p2p
  -> p2p/proto
```

Core rule:

```text
Lower-level business packages must not depend on higher-level runtime packages.
```

In practice:

- `core` must not import `service`, `node`, `p2p`, or `cmd`.
- `service` must not import `node`, `p2p`, or `cmd`.
- `p2p` must not open DB files or own blockchain state.
- `cmd` may call into `service` or `node`, but should not implement core logic.

## Type Placement Rules

### Core/domain types

Place in `internal/core`.

Examples:

- `Block`
- `Transaction`
- `BestState`
- `ChainBlock`

Use when:

- The type represents blockchain domain state.
- The type is meaningful without CLI, network, or peer context.

### Service DTOs

Place in `internal/service`.

Examples:

- `ChainState`
- `ChainInfo`
- wallet command results

Use when:

- The type is an application use-case result.
- The type may combine core and wallet data.
- The type is returned to CLI, node, or future control RPC.

### Node DTOs

Place in `internal/node`.

Examples:

- `PeerChainState`
- `PeerBlocks`

Use when:

- The type adds peer context to data returned by P2P.
- The type is meaningful only from the local node's view of remote peers.

### Proto DTOs

Place in `internal/p2p/proto`.

Examples:

- `ChainStateResponse`
- `BlockData`
- `GetBlocksFromHeightResponse`

Use when:

- The type crosses a gRPC boundary.

Do not pass proto DTOs deep into `core` or `service`.

## Blockchain Sync Data Flow

### Query remote chain state

```text
node.GetPeerChainState(peerAddr)
  -> p2p.Client.GetChainState(ctx)
  -> pb.ChainStateResponse
  -> node.PeerChainState
```

### Serve local chain state to a remote peer

```text
remote peer
  -> p2p.Server.GetChainState
  -> ChainStateProvider callback
  -> node callback closure
  -> service.BlockchainService.RequireChainState
  -> service.ChainState
  -> pb.ChainStateResponse
```

### Request blocks from remote peer

```text
node.GetPeerBlocksFromHeight(peerAddr, startHeight, limit)
  -> p2p.Client.GetBlocksFromHeight(ctx, startHeight, limit)
  -> pb.GetBlocksFromHeightResponse
  -> deserialize pb.BlockData.block
  -> node.PeerBlocks
```

### Serve local blocks to remote peer

```text
remote peer
  -> p2p.Server.GetBlocksFromHeight
  -> BlocksFromHeightProvider callback
  -> node callback closure
  -> service.BlockchainService.RequireBlocksFromHeight
  -> core.BlockChain.BlocksFromHeight
  -> []core.ChainBlock
  -> serialize core.Block
  -> pb.GetBlocksFromHeightResponse
```

## Database Ownership

While a node is running, the node process should be the owner of the chain and
wallet databases.

Rules:

- Long-running node opens and owns DB handles through `service.OpenServices`.
- CLI commands that need to work while node is running should call a local
  control RPC in the future.
- Do not make multiple processes write to the same DB file.

Current transition state:

- Some CLI commands still open DB files directly.
- This is acceptable for offline usage, but those commands may fail while a
  node process is running.

## Practical Rules for New Code

When adding a new feature, ask these questions:

1. Is this blockchain state or validation logic?

Place it in `core`.

2. Is this an application use case?

Place it in `service`.

3. Is this about a remote peer or node runtime?

Place it in `node`.

4. Is this gRPC request/response handling?

Place it in `p2p`.

5. Is this command parsing or printing?

Place it in `cmd`.

6. Is this a generated network DTO?

Place it in `p2p/proto`.

## Anti-Patterns

Avoid these:

```go
// Bad: service returns proto DTOs.
func RequireBlocksFromHeight(...) []*pb.BlockData
```

```go
// Bad: core knows about peers.
type ChainBlock struct {
	PeerAddr string
	NodeID   string
	Block    *Block
}
```

```go
// Bad: p2p directly opens database.
func (s *Server) GetBlocksFromHeight(...) {
	db, _ := database.OpenDB(...)
}
```

```go
// Bad: cmd implements blockchain traversal rules.
func printChain(...) {
	// Direct DB cursor logic and block validation here.
}
```

## Current Recommended Next Steps

1. Add `core.ChainBlock`.
2. Change `service.ChainInfo.Blocks` from `[]*core.Block` to
   `[]core.ChainBlock`.
3. Add `core.BlockChain.BlocksFromHeight(startHeight, limit int)`.
4. Add `service.BlockchainService.RequireBlocksFromHeight`.
5. Add P2P `GetBlocksFromHeight` callback and conversion.
6. Add `node.PeerBlocks`.
