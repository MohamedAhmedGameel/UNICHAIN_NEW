"""
main.py — FastAPI application entry point.
Assembles all routers and initializes the database on startup.
"""
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from db.database import init_db
from auth.router import router as auth_router
from routers.majors import router as majors_router
from routers.professors import router as professors_router
from routers.students import router as students_router
from routers.courses import router as courses_router
from routers.enrollments import router as enrollments_router
from routers.roles import router as roles_router, perm_router, user_router


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: create DB tables
    await init_db()
    yield
    # Shutdown: nothing to clean up


app = FastAPI(
    title="UNICHAIN — Hyperledger Fabric University API",
    description=(
        "Blockchain-backed university management system on Hyperledger Fabric.\n\n"
        "All data lives on-chain. Each API user gets a transparent Fabric identity — "
        "signing and role-based access control are handled server-side."
    ),
    version="1.0.0",
    lifespan=lifespan,
)

# ─── CORS (allow all for development) ────────────────────────────────────────

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# ─── Register routers ────────────────────────────────────────────────────────

app.include_router(auth_router)
app.include_router(majors_router)
app.include_router(professors_router)
app.include_router(students_router)
app.include_router(courses_router)
app.include_router(enrollments_router)
app.include_router(roles_router)
app.include_router(perm_router)
app.include_router(user_router)


# ─── Health check ─────────────────────────────────────────────────────────────

@app.get("/health", tags=["System"])
async def health():
    return {"status": "ok", "blockchain": "hyperledger-fabric"}


@app.get("/", tags=["System"])
async def root():
    return {
        "name": "UNICHAIN Hyperledger Fabric API",
        "version": "1.0.0",
        "docs": "/docs",
    }
