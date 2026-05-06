"""
identity/models.py — Fabric identity mapping ORM model.
"""
import uuid
from datetime import datetime, timezone

from sqlalchemy import Column, DateTime, ForeignKey, String

from db.database import Base


class FabricIdentity(Base):
    __tablename__ = "fabric_identities"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    app_user_id = Column(String, ForeignKey("app_users.id"), unique=True, nullable=False, index=True)
    fabric_username = Column(String, unique=True, nullable=False)
    msp_id = Column(String, nullable=False)
    wallet_label = Column(String, nullable=False)
    fabric_address = Column(String, nullable=False, index=True)
    created_at = Column(DateTime, default=lambda: datetime.now(timezone.utc))
