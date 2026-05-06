# UniChain Documentation

Hyperledger Fabric 2.5 — University Management System  
Runs on **Windows 11 + WSL2**

---

## Docs Index

| File | What's in it |
|------|-------------|
| [windows-wsl-setup.md](./docs/windows-wsl-setup.md) | Copy project from Windows into WSL2, install prerequisites |
| [start-from-scratch.md](./docs/start-from-scratch.md) | **First time only** — generate crypto, create channel, deploy chaincode |
| [start-second.md](./docs/start-second.md) | **Every time after** — start existing network and reconnect |
| [architecture.md](./docs/architecture.md) | Network topology, chaincode internals, state key schema |
| [contract-reference.md](./docs/contract-reference.md) | Every callable function with exact commands |
| [rbac.md](./docs/rbac.md) | Roles, permissions, identity, how to assign access |
| [redeploy.md](./docs/contract-reference.mdredeploy.md) | Upgrade chaincode after code changes |
| [debugging.md](./docs/debugging.md) | Read logs, inspect state, diagnose failures |
| [errors.md](./docs/errors.md) | Every known error and exact fix |

---

## Current Deployment State

| Property | Value |
|----------|-------|
| Channel | `universitychannel` |
| Chaincode name | `university` |
| Latest version | `1.2` |
| Latest sequence | `3` |
| Org MSP | `Org1MSP` |
| Peer | `localhost:7051` |
| Orderer | `localhost:7050` |
