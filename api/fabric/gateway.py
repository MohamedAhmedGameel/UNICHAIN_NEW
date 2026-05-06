"""
fabric/gateway.py — Fabric Gateway wrapper.

Provides submit_transaction (write) and evaluate_transaction (read)
using a given user identity. Each call signs the proposal with the
user's private key from the wallet.

This implementation uses the Fabric Gateway peer service (gRPC).
For environments without the gRPC SDK, it falls back to invoking
the peer CLI via subprocess.
"""
from __future__ import annotations

import json
import subprocess
from pathlib import Path

from config import settings
from fabric.wallet import Identity


async def submit_transaction(identity: Identity, function: str, *args: str) -> dict:
    """
    Submit a write transaction to the chaincode.
    Signs with the user's private key so the chaincode sees their identity.
    """
    return await _invoke(identity, function, list(args), evaluate=False)


async def evaluate_transaction(identity: Identity, function: str, *args: str) -> dict:
    """
    Evaluate a read-only query (no commit to ledger).
    """
    return await _invoke(identity, function, list(args), evaluate=True)


async def _invoke(identity: Identity, function: str, args: list[str], evaluate: bool) -> dict:
    """
    Internal: invoke chaincode via peer CLI with the user's MSP credentials.

    In production, replace this with the Fabric Gateway gRPC SDK for
    better performance. The CLI approach works universally and is easier
    to set up.
    """
    # Write identity cert/key to temp location for the peer CLI
    import tempfile
    import os

    tmp_dir = Path(tempfile.mkdtemp(prefix="fabric_"))
    msp_dir = tmp_dir / "msp"
    (msp_dir / "signcerts").mkdir(parents=True)
    (msp_dir / "keystore").mkdir(parents=True)
    (msp_dir / "signcerts" / "cert.pem").write_text(identity.cert_pem)
    (msp_dir / "keystore" / "key.pem").write_text(identity.key_pem)

    # Copy cacerts from the org MSP (needed by peer CLI)
    org_msp = Path(settings.fabric_peer_tls_cert).parent.parent / "msp"
    cacerts_src = org_msp / "cacerts"
    if cacerts_src.exists():
        (msp_dir / "cacerts").mkdir(parents=True)
        for f in cacerts_src.iterdir():
            (msp_dir / "cacerts" / f.name).write_text(f.read_text())

    # Build the chaincode call JSON
    cc_args = {"function": function, "Args": args}
    cc_args_json = json.dumps(cc_args)

    env = os.environ.copy()
    env.update({
        "CORE_PEER_TLS_ENABLED": "true",
        "CORE_PEER_LOCALMSPID": identity.msp_id,
        "CORE_PEER_TLS_ROOTCERT_FILE": str(Path(settings.fabric_peer_tls_cert).resolve()),
        "CORE_PEER_MSPCONFIGPATH": str(msp_dir.resolve()),
        "CORE_PEER_ADDRESS": settings.fabric_peer_endpoint,
    })

    cmd = ["peer", "chaincode"]
    if evaluate:
        cmd.append("query")
    else:
        cmd.append("invoke")
        cmd.extend([
            "-o", "localhost:7050",
            "--ordererTLSHostnameOverride", "orderer.university.com",
            "--tls",
            "--cafile", str(Path(settings.fabric_peer_tls_cert).resolve().parent.parent.parent.parent
                           / "ordererOrganizations" / "university.com" / "orderers"
                           / "orderer.university.com" / "msp" / "tlscacerts"
                           / "tlsca.university.com-cert.pem"),
            "--peerAddresses", settings.fabric_peer_endpoint,
            "--tlsRootCertFiles", str(Path(settings.fabric_peer_tls_cert).resolve()),
        ])

    cmd.extend([
        "-C", settings.fabric_channel,
        "-n", settings.fabric_chaincode,
        "-c", cc_args_json,
    ])

    try:
        result = subprocess.run(
            cmd, capture_output=True, text=True, timeout=30, env=env,
        )

        # Clean up temp MSP
        import shutil
        shutil.rmtree(tmp_dir, ignore_errors=True)

        if result.returncode != 0:
            error_msg = result.stderr.strip()
            raise RuntimeError(f"Chaincode call failed: {error_msg}")

        output = result.stdout.strip()
        if not output:
            return {"status": "ok"}

        try:
            return json.loads(output)
        except json.JSONDecodeError:
            return {"result": output}

    except subprocess.TimeoutExpired:
        import shutil
        shutil.rmtree(tmp_dir, ignore_errors=True)
        raise RuntimeError("Chaincode call timed out")
