#!/usr/bin/env python3

from datetime import datetime, timezone
import json
import asyncio
import base64
import hashlib
import hmac
import httpx
import os


class CrusoeAuth(httpx.Auth):
    """
    HMAC-based authentication for Crusoe Cloud API.

    Implements the signature scheme documented at:
    https://docs.crusoecloud.com/reference/api/
    """

    def __init__(self, api_access_key: str, secret_key: str):
        self.api_access_key = api_access_key
        self.secret_key = secret_key

    # This is based on the sample Python code at
    # https://docs.crusoecloud.com/reference/api/
    def auth_flow(self, request: httpx.Request):  # noqa: F821
        # Generate RFC3339 timestamp
        timestamp = datetime.now(timezone.utc).isoformat()

        # Build signature payload
        # Format: http_path\ncanonical_query_params\nhttp_verb\ntimestamp
        path = request.url.raw_path.decode()
        query_params = self._canonicalize_query_params(request.url.params)
        method = request.method
        payload = f"{path}\n{query_params}\n{method}\n{timestamp}\n"

        # Create HMAC SHA256 signature
        # Per Crusoe API docs, secret key must be base64url-decoded to raw bytes
        secret_key_bytes = base64.urlsafe_b64decode(
            self.secret_key + "=" * (-len(self.secret_key) % 4)
        )
        signature = hmac.new(
            key=secret_key_bytes,
            msg=bytes(payload, "ascii"),
            digestmod=hashlib.sha256,
        ).digest()
        signature_b64 = base64.urlsafe_b64encode(signature).decode("ascii").rstrip("=")

        # Set headers
        request.headers["X-Crusoe-Timestamp"] = timestamp
        request.headers["Authorization"] = (
            f"Bearer 1.0:{self.api_access_key}:{signature_b64}"
        )

        yield request

    def _canonicalize_query_params(self, params) -> str:
        """
        Canonicalize query parameters by sorting them lexicographically.
        If no params, return empty string.
        """
        if not params:
            return ""

        # Sort params by key and format as key=value pairs
        sorted_params = sorted(params.items())
        return "&".join(f"{k}={v}" for k, v in sorted_params)


class CrusoeAPI:
    def __init__(self):
        project_id = os.environ["CRUSOE_PROJECT_ID"]
        api_access_key = os.environ["CRUSOE_ACCESS_KEY_ID"]
        secret_key = os.environ["CRUSOE_SECRET_ACCESS_KEY"]

        self.project_id = project_id
        self.http_client = httpx.AsyncClient(
            http2=True,
            timeout=30,
            transport=httpx.AsyncHTTPTransport(retries=5),
            base_url=f"https://api.crusoecloud.com/v1alpha5/projects/{project_id}",
            auth=CrusoeAuth(api_access_key, secret_key),
            headers={"Content-Type": "application/json"},
        )

    async def stop(self) -> None:
        await self.http_client.aclose()

    async def request(self, method: str, path: str, data: dict | None = None) -> dict:
        response = await self.http_client.request(method, path, json=data)
        response.raise_for_status()
        return response.json()
    
    async def poll_image_operation(self, operation_id: str) -> dict:
        while True:
            res = await self.request("GET", f"compute/custom-images/operations/{operation_id}")
            print(json.dumps(res, indent=2))
            if res["state"] in ("SUCCEEDED", "FAILED"):
                return res
            await asyncio.sleep(2)

    async def poll_operation(self, operation_id: str) -> dict:
        while True:
            res = await self.request("GET", f"compute/vms/instances/operations/{operation_id}")
            print(json.dumps(res, indent=2))
            if res["state"] in ("SUCCEEDED", "FAILED"):
                return res
            await asyncio.sleep(2)


async def main():
    import argparse
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)

    poll_op = subparsers.add_parser("poll-op", help="Poll a VM operation")
    poll_op.add_argument("operation_id", type=str, help="Operation ID to poll")

    poll_image_op = subparsers.add_parser("poll-image-op", help="Poll a custom image operation")
    poll_image_op.add_argument("operation_id", type=str, help="Operation ID to poll")

    args = parser.parse_args()
    api = CrusoeAPI()

    if args.command == "poll-op":
        await api.poll_operation(args.operation_id)
    elif args.command == "poll-image-op":
        await api.poll_image_operation(args.operation_id)


if __name__ == "__main__":
    asyncio.run(main())
