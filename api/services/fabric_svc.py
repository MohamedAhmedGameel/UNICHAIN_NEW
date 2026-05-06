"""
services/fabric_svc.py — Base service that wraps gateway calls.
All domain services inherit this.
"""
from __future__ import annotations

from fabric import gateway
from fabric.wallet import Identity


class FabricService:
    async def submit(self, identity: Identity, fn: str, *args: str) -> dict:
        return await gateway.submit_transaction(identity, fn, *args)

    async def evaluate(self, identity: Identity, fn: str, *args: str) -> dict:
        return await gateway.evaluate_transaction(identity, fn, *args)
