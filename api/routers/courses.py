"""
routers/courses.py — Course CRUD endpoints.
"""
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from sqlalchemy.ext.asyncio import AsyncSession

from auth.router import get_current_user
from db.database import get_db
from identity.bridge import get_or_create_identity
from services.fabric_svc import FabricService

router = APIRouter(prefix="/courses", tags=["Courses"])
svc = FabricService()


class CourseCreate(BaseModel):
    id: str
    name: str
    professor_id: int

class CourseUpdate(BaseModel):
    name: str

class CourseReassign(BaseModel):
    new_professor_id: int


@router.get("")
async def list_courses(user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetAllCourses")


@router.get("/{course_id}")
async def get_course(course_id: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetCourse", course_id)


@router.post("", status_code=201)
async def create_course(req: CourseCreate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "CreateCourse", req.id, req.name, str(req.professor_id))
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.put("/{course_id}")
async def update_course(course_id: str, req: CourseUpdate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "UpdateCourse", course_id, req.name)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.patch("/{course_id}/reassign")
async def reassign_course(course_id: str, req: CourseReassign, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "ReassignCourse", course_id, str(req.new_professor_id))
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/{course_id}")
async def delete_course(course_id: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "DeleteCourse", course_id)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))
