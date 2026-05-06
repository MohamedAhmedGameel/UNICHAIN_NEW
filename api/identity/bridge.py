"""
identity/bridge.py — Maps AppUser → Fabric identity (creates on demand).

On a user's first Fabric action:
  1. Register username with Fabric CA
  2. Enroll → get X.509 cert + private key
  3. Store in filesystem wallet
  4. Save mapping to DB

On subsequent calls:
  Load from wallet directly.
"""
from __future__ import annotations

import uuid

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from auth.models import AppUser
from config import settings
from fabric import wallet
from fabric.ca_client import ca_client
from identity.models import FabricIdentity


async def get_or_create_identity(user: AppUser, db: AsyncSession) -> wallet.Identity:
    """
    Returns a loaded wallet Identity for the given app user.
    Creates a new Fabric identity (CA register + enroll) if one doesn't exist yet.
    """
    # 1. Check DB for existing mapping
    result = await db.execute(
        select(FabricIdentity).where(FabricIdentity.app_user_id == user.id)
    )
    mapping = result.scalar_one_or_none()

    if mapping:
        identity = wallet.load(mapping.wallet_label)
        if identity:
            return identity
        # Wallet file missing — re-enroll
        # (This shouldn't happen normally, but handle gracefully)

    # 2. Create new Fabric identity
    fabric_username = f"user_{user.id.replace('-', '')[:16]}"
    fabric_secret = uuid.uuid4().hex

    # Register with CA
    await ca_client.register(fabric_username, fabric_secret)

    # Enroll to get cert + key
    cert_pem, key_pem = await ca_client.enroll(fabric_username, fabric_secret)

    # Derive the Fabric address (must match chaincode's getCallerAddress)
    fabric_address = wallet.cert_to_address(cert_pem)

    # Store in wallet
    wallet_label = f"user_{user.id}"
    wallet.store(wallet_label, settings.fabric_msp_id, cert_pem, key_pem)

    # Save mapping to DB
    if not mapping:
        mapping = FabricIdentity(
            app_user_id=user.id,
            fabric_username=fabric_username,
            msp_id=settings.fabric_msp_id,
            wallet_label=wallet_label,
            fabric_address=fabric_address,
        )
        db.add(mapping)
    else:
        mapping.wallet_label = wallet_label
        mapping.fabric_address = fabric_address
    await db.flush()

    return wallet.load(wallet_label)


async def get_fabric_address(user: AppUser, db: AsyncSession) -> str:
    """Get the Fabric address for a user (creates identity if needed)."""
    identity = await get_or_create_identity(user, db)
    return wallet.cert_to_address(identity.cert_pem)


async def get_fabric_address_by_user_id(user_id: str, db: AsyncSession) -> str | None:
    """Look up fabric address by app user ID without creating identity."""
    result = await db.execute(
        select(FabricIdentity).where(FabricIdentity.app_user_id == user_id)
    )
    mapping = result.scalar_one_or_none()
    return mapping.fabric_address if mapping else None
