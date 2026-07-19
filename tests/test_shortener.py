from __future__ import annotations
 
import pytest
 
from app.shortener import (
    CodeConflictError,
    CodeGenerationError,
    generate_code,
    resolve_code,
)
from app.store import URLStore
from app.store import URLEntry
from datetime import datetime, timezone


async def empty_store() -> URLStore:
    return URLStore()


async def store_with(*codes: str) -> URLStore:
    store = URLStore()
    for code in codes:
        entry = URLEntry(
            code=code,
            original_url="https://example.com",
            created_at=datetime.now(timezone.utc),
            expires_at=None,
            max_clicks=100,
        )
        await store.save(entry)
    return store


async def test_generate_code_returns_correct_length():
    store = await empty_store()
    code = await generate_code(store, code_length=6, max_retries=5)
    assert len(code) == 6


async def test_generate_code_unique():
    store = await empty_store()
    codes = {await generate_code(store, code_length=6, max_retries=5) for _ in range(20)}
    # all 20 should be distinct (astronomically unlikely to collide)
    assert len(codes) == 20

async def test_generate_code_retries_on_collision():
    """Pre-fill the store so the first N candidates will collide, then verify
    it eventually finds a fresh one."""
    store = await empty_store()

    code = await generate_code(store, code_length=8, max_retries=3)
    assert len(code) == 8

async def test_generate_code_raises_after_max_retries(monkeypatch):
    """Fill a tiny space so every candidate collides."""
    import app.shortener as mod
 
    call_count = 0
 
    def always_same(_n):
        nonlocal call_count
        call_count += 1
        return "aaaaaa"
 
    monkeypatch.setattr(mod.secrets, "token_urlsafe", always_same)
    store = await store_with("aaaaaa")
 
    with pytest.raises(CodeGenerationError):
        await generate_code(store, code_length=6, max_retries=3)
 
    assert call_count == 3

async def test_resolve_code_uses_custom_when_provided():
    store = await empty_store()
    code = await resolve_code(store, "my-alias", code_length=6, max_retries=5)
    assert code == "my-alias"

async def test_resolve_code_raises_on_conflict():
    store = await store_with("taken")
    with pytest.raises(CodeConflictError) as exc_info:
        await resolve_code(store, "taken", code_length=6, max_retries=5)
    assert exc_info.value.code == "taken"
 
async def test_resolve_code_generates_random_when_no_custom():
    store = await empty_store()
    code = await resolve_code(store, None, code_length=6, max_retries=5)
    assert len(code) == 6
