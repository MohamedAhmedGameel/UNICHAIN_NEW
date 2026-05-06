"""
config.py — Fully env-driven configuration. Nothing hardcoded.
"""
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # ── Fabric ────────────────────────────────────────────────────────────────
    fabric_channel: str = "universitychannel"
    fabric_chaincode: str = "university"
    fabric_msp_id: str = "Org1MSP"
    fabric_peer_endpoint: str = "localhost:7051"
    fabric_peer_tls_cert: str = "./network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt"
    fabric_ca_url: str = "https://localhost:7054"
    fabric_ca_name: str = "ca-org1"
    fabric_ca_admin: str = "admin"
    fabric_ca_admin_pw: str = "adminpw"
    fabric_ca_tls_cert: str = "./network/organizations/fabric-ca/org1/tls-cert.pem"
    wallet_path: str = "./wallet"
    connection_profile: str = "./network/connection-profile.yaml"

    # ── Auth ──────────────────────────────────────────────────────────────────
    jwt_secret: str = "CHANGE-ME-IN-PRODUCTION"
    jwt_algorithm: str = "HS256"
    jwt_expire_minutes: int = 1440  # 24 hours

    # ── Database ──────────────────────────────────────────────────────────────
    database_url: str = "sqlite+aiosqlite:///./unichain.db"

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings()
