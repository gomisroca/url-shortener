"""
Code generation for shortened URLs.
 
Two paths:
1. Random code - secrets.token_urlsafe gives cryptographically random
   base64url characters. Slice to code_length characters. On a collision
   (extremely unlikely with 6 chars but bounded-retryable regardless)
   generate a fresh one rather than appending a suffix, which would leak
   information about how many collisions occurred.
 
2. Custom alias - caller-supplied, validated by the Pydantic model.
   If already taken,  raise CodeConflictError immediately (no retry).
"""
from __future__ import annotations
 
import secrets
 
from app.store import URLStore


class CodeConflictError(Exception):
    """Raised when a custom alias is already taken."""
    def __init__(self, code: str) -> None:
        super().__init__(f"code {code!r} is already in use")
        self.code = code


class CodeGenerationError(Exception):
    """Raised when random generation fails after max_retries (should never happen in practice)."""


async def generate_code(
        store: URLStore,
        code_length: int,
        max_retries: int,
) -> str:
    """Generate a unique random short code, retrying on collision."""

    for _ in range(max_retries):
        # token_urlsafe returns base64url chars (A-Z, a-z, 0-9, -, _)
        # Need more raw bytes than code_length since base64 expands 3→4
        candidate = secrets.token_urlsafe(code_length)[:code_length]

        if not await store.exists(candidate):
            return candidate
    raise CodeGenerationError(f"failed to generate unique code after {max_retries} attempts")


async def resolve_code(
    store: URLStore,
    custom_code: str | None,
    code_length: int,
    max_retries: int,
) -> str:
    """
    Return either the custom validated code or a freshly generated random one.
    Raises CodeConflictError if the custom code is already in use.
    """
    if custom_code is not None:
        if await store.exists(custom_code):
            raise CodeConflictError(custom_code)
        return custom_code
    
    return await generate_code(store, code_length, max_retries)



