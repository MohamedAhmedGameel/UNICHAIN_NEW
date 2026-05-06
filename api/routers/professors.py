"""
routers/professors.py — Professor CRUD endpoints.
"""
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from sqlalchemy.ext.asyncio import AsyncSession

from auth.router import get_current_user
from db.database import get_db
from identity.bridge import get_or_create_identity
from services.fabric_svc import FabricService

router = APIRouter(prefix="/professors", tags=["Professors"])
svc = FabricService()


class ProfessorCreate(BaseModel):
    name: str
    department: str
    professor_address: str  # Fabric address of the professor

class ProfessorUpdate(BaseModel):
    name: str = ""
    department: str = ""
    new_address: str = ""


@router.get("")
async def list_professors(user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetAllProfessors")


@router.get("/{professor_id}")
async def get_professor(professor_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetProfessor", str(professor_id))


@router.post("", status_code=201)
async def create_professor(req: ProfessorCreate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "AddProfessor", req.name, req.department, req.professor_address)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.put("/{professor_id}")
async def update_professor(professor_id: int, req: ProfessorUpdate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "UpdateProfessor", str(professor_id), req.name, req.department, req.new_address)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/{professor_id}")
async def delete_professor(professor_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "DeleteProfessor", str(professor_id))
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))
