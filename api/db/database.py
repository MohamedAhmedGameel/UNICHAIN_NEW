"""
db/database.py — Async SQLAlchemy engine + session factory.
"""
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.orm import DeclarativeBase

from config import settings

engine = async_sessionmaker_instance = None


class Base(DeclarativeBase):
    pass


def get_engine():
    global engine
    if engine is None:
        engine = create_async_engine(settings.database_url, echo=False)
    return engine


def get_session_factory():
    global async_sessionmaker_instance
    if async_sessionmaker_instance is None:
        async_sessionmaker_instance = async_sessionmaker(
            get_engine(), class_=AsyncSession, expire_on_commit=False
        )
    return async_sessionmaker_instance


async def get_db():
    """FastAPI dependency — yields an async session."""
    factory = get_session_factory()
    async with factory() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise


async def init_db():
    """Create all tables on startup."""
    async with get_engine().begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
