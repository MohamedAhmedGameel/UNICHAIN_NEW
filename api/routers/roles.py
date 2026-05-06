"""
routers/roles.py — Fully dynamic role and permission management.
No role names are hardcoded — everything lives in chaincode state.
"""
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from sqlalchemy.ext.asyncio import AsyncSession

from auth.router import get_current_user
from db.database import get_db
from identity.bridge import get_or_create_identity, get_fabric_address, get_fabric_address_by_user_id
from services.fabric_svc import FabricService

router = APIRouter(prefix="/roles", tags=["Roles & Permissions"])
svc = FabricService()


class RoleCreate(BaseModel):
    name: str
    description: str = ""

class PermissionCreate(BaseModel):
    name: str
    description: str = ""

class PermissionGrant(BaseModel):
    permission: str

class MemberAssign(BaseModel):
    fabric_address: str | None = None
    app_user_id: str | None = None


# ─── Role CRUD ────────────────────────────────────────────────────────────────

@router.get("")
async def list_roles(user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetAllRoles")


@router.post("", status_code=201)
async def create_role(req: RoleCreate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "CreateRole", req.name, req.description)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/{role_name}")
async def delete_role(role_name: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "DeleteRole", role_name)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


# ─── Role ↔ Permission ──────────────────────────────────────────────────────

@router.get("/{role_name}/permissions")
async def role_permissions(role_name: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetRolePermissions", role_name)


@router.post("/{role_name}/permissions")
async def grant_permission(role_name: str, req: PermissionGrant, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "GrantPermission", role_name, req.permission)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/{role_name}/permissions/{perm}")
async def revoke_permission(role_name: str, perm: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "RevokePermission", role_name, perm)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


# ─── Role ↔ Members ──────────────────────────────────────────────────────────

@router.get("/{role_name}/members")
async def role_members(role_name: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetRoleMembers", role_name)


@router.post("/{role_name}/members")
async def assign_role(role_name: str, req: MemberAssign, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    # Resolve fabric address from either direct address or app_user_id
    addr = req.fabric_address
    if not addr and req.app_user_id:
        addr = await get_fabric_address_by_user_id(req.app_user_id, db)
        if not addr:
            raise HTTPException(status_code=404, detail="User has no Fabric identity yet")
    if not addr:
        raise HTTPException(status_code=400, detail="Provide fabric_address or app_user_id")
    try:
        return await svc.submit(identity, "GrantRole", role_name, addr)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.delete("/{role_name}/members/{address}")
async def remove_member(role_name: str, address: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "RevokeRole", role_name, address)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


# ─── Permissions CRUD ────────────────────────────────────────────────────────

perm_router = APIRouter(prefix="/permissions", tags=["Roles & Permissions"])


@perm_router.get("")
async def list_permissions(user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    return await svc.evaluate(identity, "GetAllPermissions")


@perm_router.post("", status_code=201)
async def create_permission(req: PermissionCreate, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    try:
        return await svc.submit(identity, "CreatePermission", req.name, req.description)
    except RuntimeError as e:
        raise HTTPException(status_code=400, detail=str(e))


# ─── User permissions ────────────────────────────────────────────────────────

user_router = APIRouter(prefix="/users", tags=["Roles & Permissions"])


@user_router.get("/{user_id}/roles")
async def user_roles(user_id: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    addr = await get_fabric_address_by_user_id(user_id, db)
    if not addr:
        raise HTTPException(status_code=404, detail="User has no Fabric identity")
    return await svc.evaluate(identity, "GetUserRoles", addr)


@user_router.get("/{user_id}/permissions")
async def user_permissions(user_id: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    identity = await get_or_create_identity(user, db)
    addr = await get_fabric_address_by_user_id(user_id, db)
    if not addr:
        raise HTTPException(status_code=404, detail="User has no Fabric identity")
    return await svc.evaluate(identity, "GetUserEffectivePermissions", addr)


@user_router.get("/{user_id}/fabric-address")
async def user_fabric_address(user_id: str, user=Depends(get_current_user), db: AsyncSession = Depends(get_db)):
    addr = await get_fabric_address_by_user_id(user_id, db)
    if not addr:
        raise HTTPException(status_code=404, detail="User has no Fabric identity")
    return {"app_user_id": user_id, "fabric_address": addr}
