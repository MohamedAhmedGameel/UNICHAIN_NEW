"""
fabric/wallet.py — Filesystem-based identity wallet for per-user Fabric certs.
Each user gets a directory: {wallet_path}/{label}/ containing cert.pem and key.pem.
"""
from __future__ import annotations

import hashlib
import os
from dataclasses import dataclass
from pathlib import Path

from cryptography import x509

from config import settings


@dataclass
class Identity:
    label: str
    msp_id: str
    cert_pem: str
    key_pem: str


def _user_dir(label: str) -> Path:
    return Path(settings.wallet_path) / label


def store(label: str, msp_id: str, cert_pem: str, key_pem: str) -> None:
    """Save cert + key to disk under the given label."""
    d = _user_dir(label)
    d.mkdir(parents=True, exist_ok=True)
    (d / "cert.pem").write_text(cert_pem)
    (d / "key.pem").write_text(key_pem)
    (d / "msp_id.txt").write_text(msp_id)


def load(label: str) -> Identity | None:
    """Load a stored identity by label, or None if not found."""
    d = _user_dir(label)
    if not (d / "cert.pem").exists():
        return None
    return Identity(
        label=label,
        msp_id=(d / "msp_id.txt").read_text().strip(),
        cert_pem=(d / "cert.pem").read_text(),
        key_pem=(d / "key.pem").read_text(),
    )


def exists(label: str) -> bool:
    return ((_user_dir(label)) / "cert.pem").exists()


def cert_to_address(cert_pem: str) -> str:
    """
    Derive a deterministic 40-char hex address from an X.509 certificate.
    Must match the Go chaincode's getCallerAddress() exactly.
    
    Go side: sha256(clientIdentity.GetID())[:40]
    GetID() returns: "x509::CN=username,OU=client::CN=ca-org1..."
    We replicate that format here.
    """
    cert = x509.load_pem_x509_certificate(cert_pem.encode())
    # Fabric's GetID() is the cert subject + issuer concatenation
    subject = cert.subject.rfc4514_string()
    issuer = cert.issuer.rfc4514_string()
    identity_str = f"x509::{subject}::{issuer}"
    h = hashlib.sha256(identity_str.encode()).hexdigest()
    return h[:40]


def delete(label: str) -> None:
    """Remove a stored identity."""
    import shutil
    d = _user_dir(label)
    if d.exists():
        shutil.rmtree(d)
