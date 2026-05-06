# UniChain Documentation

Hyperledger Fabric 2.5 — University Management System  
Runs on **Windows 11 + WSL2**

---

## Docs Index

| File | What's in it |
|------|-------------|
| [windows-wsl-setup.md](./windows-wsl-setup.md) | Copy project from Windows into WSL2, install prerequisites |
| [start-from-scratch.md](./start-from-scratch.md) | **First time only** — generate crypto, create channel, deploy chaincode |
| [start-second.md](./start-second.md) | **Every time after** — start existing network and reconnect |
| [architecture.md](./architecture.md) | Network topology, chaincode internals, state key schema |
| [contract-reference.md](./contract-reference.md) | Every callable function with exact commands |
| [rbac.md](./rbac.md) | Roles, permissions, identity, how to assign access |
| [redeploy.md](./redeploy.md) | Upgrade chaincode after code changes |
| [debugging.md](./debugging.md) | Read logs, inspect state, diagnose failures |
| [errors.md](./errors.md) | Every known error and exact fix |

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
