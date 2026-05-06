"""
routers/enrollments.py — Enrollment, marks, transcript, and GPA endpoints.
"""
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from sqlalchemy.ext.asyncio import AsyncSession

from auth.router import get_current_user
from db.database import get_db
from identity.bridge import get_or_create_identity
from services.fabric_svc import FabricService
from services import gpa as gpa_svc

router = APIRouter(tags=["Enrollments"])
svc = FabricService()


class EnrollRequest(BaseModel):
    semester: str
    student_id: int
    course_id: str

class BatchEnrollRequest(BaseModel):
    semester: str
    student_ids: list[int]
    course_id: str

class UnenrollRequest(BaseModel):
    semester: str
    student_id: int
    course_id: str

class UpdateMarkRequest(BaseModel):
    semester: str
    student_id: int
    course_id: str
    mark: int


# ─── Enrollment write ────────────────────────────────────────────────────────

@router.post("/enrollments", status_code=201, tags=["Enrollments"])
async def enroll(req: EnrollRequest, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "EnrollStudent", str(req.student_id), req.course_id, req.semester)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.post("/enrollments/batch", status_code=201, tags=["Enrollments"])
async def batch_enroll(req: BatchEnrollRequest, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    ids = [str(s) for s in req.student_ids]
    try:
        return await svc.submit(identity, "BatchEnroll", req.course_id, req.semester, *ids)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/enrollments", tags=["Enrollments"])
async def unenroll(req: UnenrollRequest, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "UnenrollStudent", str(req.student_id), req.course_id, req.semester)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.patch("/enrollments/mark", tags=["Enrollments"])
async def update_mark(req: UpdateMarkRequest, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(
            identity, "UpdateMark",
            str(req.student_id), req.course_id, req.semester, str(req.mark),
        )
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


# ─── Enrollment read ─────────────────────────────────────────────────────────

@router.get("/students/{student_id}/enrollments", tags=["Enrollments"])
async def student_enrollments(student_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetStudentEnrollments", str(student_id))


@router.get("/students/{student_id}/semesters", tags=["Enrollments"])
async def student_semesters(student_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetStudentSemesters", str(student_id))


@router.get("/students/{student_id}/semesters/{semester}", tags=["Enrollments"])
async def semester_enrollments(student_id: int, semester: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetSemesterEnrollments", str(student_id), semester)


@router.get("/students/{student_id}/transcript", tags=["Transcript"])
async def transcript(student_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    records = await svc.evaluate(identity, "GetStudentEnrollments", str(student_id))
    if isinstance(records, dict) and "result" in records:
        import json
        records = json.loads(records["result"]) if isinstance(records["result"], str) else records["result"]
    if not isinstance(records, list):
        records = []
    return gpa_svc.build_transcript(str(student_id), records)


@router.get("/students/{student_id}/gpa", tags=["Transcript"])
async def student_gpa(student_id: int, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    records = await svc.evaluate(identity, "GetStudentEnrollments", str(student_id))
    if isinstance(records, dict) and "result" in records:
        import json
        records = json.loads(records["result"]) if isinstance(records["result"], str) else records["result"]
    if not isinstance(records, list):
        records = []
    gpa = gpa_svc.compute_gpa(records)
    return {"student_id": student_id, "gpa": gpa}


@router.get("/courses/{course_id}/enrollments", tags=["Enrollments"])
async def course_enrollments(course_id: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetCourseEnrollments", course_id)
