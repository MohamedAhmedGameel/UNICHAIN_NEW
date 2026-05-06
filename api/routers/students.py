"""
routers/students.py — Student CRUD endpoints.
"""
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from sqlalchemy.ext.asyncio import AsyncSession

from auth.router import get_current_user
from db.database import get_db
from identity.bridge import get_or_create_identity
from services.fabric_svc import FabricService

router = APIRouter(prefix="/students", tags=["Students"])
svc = FabricService()


class StudentCreate(BaseModel):
    name: str
    major_id: int
    year: int
    supervisor_address: str = ""
    wallet_address: str = ""

class StudentUpdate(BaseModel):
    name: str = ""
    major_id: int = 0
    year: int = 0
    supervisor_address: str = ""
    wallet_address: str = ""


@router.get("")
async def list_students(user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetAllStudents")


@router.get("/{student_id}")
async def get_student(student_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetStudent", str(student_id))


@router.post("", status_code=201)
async def create_student(req: StudentCreate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(
            identity, "AddStudent",
            req.name, str(req.major_id), str(req.year),
            req.supervisor_address, req.wallet_address,
        )
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.put("/{student_id}")
async def update_student(student_id: int, req: StudentUpdate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(
            identity, "UpdateStudent",
            str(student_id), req.name, str(req.major_id), str(req.year),
            req.supervisor_address, req.wallet_address,
        )
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/{student_id}")
async def delete_student(student_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "DeleteStudent", str(student_id))
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))
