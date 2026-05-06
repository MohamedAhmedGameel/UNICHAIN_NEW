"""
fabric/ca_client.py — Register and enroll users with the Fabric CA.
Uses the Fabric CA REST API directly via httpx.
"""
from __future__ import annotations

import base64
import json

import httpx
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.x509.oid import NameOID
from cryptography.hazmat.backends import default_backend

from config import settings


class FabricCAClient:
    """
    Lightweight Fabric CA client that talks to the CA REST API.
    Handles register + enroll to produce an X.509 cert + private key pair.
    """

    def __init__(self):
        self.ca_url = settings.fabric_ca_url
        self.ca_name = settings.fabric_ca_name
        self.admin_id = settings.fabric_ca_admin
        self.admin_secret = settings.fabric_ca_admin_pw
        self._admin_token: str | None = None

    async def _get_admin_token(self) -> str:
        """Enroll the CA admin to get a token for registering new users."""
        if self._admin_token:
            return self._admin_token

        # Generate a temporary key pair for admin enrollment
        private_key = ec.generate_private_key(ec.SECP256R1(), default_backend())
        csr = (
            x509.CertificateSigningRequestBuilder()
            .subject_name(x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, self.admin_id)]))
            .sign(private_key, hashes.SHA256(), default_backend())
        )
        csr_pem = csr.public_bytes(serialization.Encoding.PEM).decode()

        async with httpx.AsyncClient(verify=False) as client:
            resp = await client.post(
                f"{self.ca_url}/enroll",
                json={
                    "hosts": ["localhost"],
                    "certificate_request": csr_pem,
                    "profile": "",
                    "crl_override": "",
                    "label": "",
                    "NotBefore": "",
                    "NotAfter": "",
                    "CAName": self.ca_name,
                },
                auth=(self.admin_id, self.admin_secret),
            )
            if resp.status_code != 200:
                raise RuntimeError(f"CA admin enroll failed: {resp.status_code} {resp.text}")

            data = resp.json()
            cert_b64 = data.get("result", {}).get("Cert", "")
            self._admin_token = base64.b64decode(cert_b64).decode()

        return self._admin_token

    async def register(self, username: str, secret: str, user_type: str = "client") -> str:
        """
        Register a new identity with the CA.
        Returns the enrollment secret.
        """
        await self._get_admin_token()

        async with httpx.AsyncClient(verify=False) as client:
            resp = await client.post(
                f"{self.ca_url}/register",
                json={
                    "id": username,
                    "type": user_type,
                    "secret": secret,
                    "affiliation": "org1.department1",
                    "max_enrollments": -1,
                    "attrs": [],
                    "CAName": self.ca_name,
                },
                auth=(self.admin_id, self.admin_secret),
            )
            if resp.status_code != 200 and "already registered" not in resp.text.lower():
                raise RuntimeError(f"CA register failed: {resp.status_code} {resp.text}")

        return secret

    async def enroll(self, username: str, secret: str) -> tuple[str, str]:
        """
        Enroll a registered identity to get cert + private key.
        Returns (cert_pem, private_key_pem).
        """
        # Generate key pair
        private_key = ec.generate_private_key(ec.SECP256R1(), default_backend())
        csr = (
            x509.CertificateSigningRequestBuilder()
            .subject_name(x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, username)]))
            .sign(private_key, hashes.SHA256(), default_backend())
        )
        csr_pem = csr.public_bytes(serialization.Encoding.PEM).decode()

        async with httpx.AsyncClient(verify=False) as client:
            resp = await client.post(
                f"{self.ca_url}/enroll",
                json={
                    "hosts": ["localhost"],
                    "certificate_request": csr_pem,
                    "profile": "",
                    "crl_override": "",
                    "label": "",
                    "NotBefore": "",
                    "NotAfter": "",
                    "CAName": self.ca_name,
                },
                auth=(username, secret),
            )
            if resp.status_code != 200:
                raise RuntimeError(f"CA enroll failed: {resp.status_code} {resp.text}")

            data = resp.json()
            cert_b64 = data.get("result", {}).get("Cert", "")
            cert_pem = base64.b64decode(cert_b64).decode()

        key_pem = private_key.private_bytes(
            serialization.Encoding.PEM,
            serialization.PrivateFormat.PKCS8,
            serialization.NoEncryption(),
        ).decode()

        return cert_pem, key_pem


# Singleton
ca_client = FabricCAClient()
