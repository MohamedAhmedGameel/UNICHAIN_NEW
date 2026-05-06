"""
routers/majors.py — Major CRUD endpoints.
"""
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from sqlalchemy.ext.asyncio import AsyncSession

from auth.router import get_current_user
from db.database import get_db
from identity.bridge import get_or_create_identity
from services.fabric_svc import FabricService

router = APIRouter(prefix="/majors", tags=["Majors"])
svc = FabricService()


class MajorCreate(BaseModel):
    code: str
    name: str
    description: str = ""

class MajorUpdate(BaseModel):
    name: str = ""
    description: str = ""


@router.get("")
async def list_majors(user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetAllMajors")


@router.get("/{major_id}")
async def get_major(major_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetMajor", str(major_id))


@router.get("/code/{code}")
async def get_major_by_code(code: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetMajorByCode", code)


@router.post("", status_code=201)
async def create_major(req: MajorCreate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "AddMajor", req.code, req.name, req.description)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.put("/{major_id}")
async def update_major(major_id: int, req: MajorUpdate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "UpdateMajor", str(major_id), req.name, req.description)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/{major_id}")
async def deactivate_major(major_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "DeactivateMajor", str(major_id))
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))
