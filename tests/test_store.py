from __future__ import annotations
 
import asyncio
from datetime import datetime, timedelta, timezone
 
import pytest
 
from app.models import ClickEvent
from app.store import URLEntry, URLStore


def make_entry(code="abc123", ttl_seconds=None, max_clicks=100):
    expires_at = None
    if ttl_seconds is not None:
        expires_at = datetime.now(timezone.utc) + timedelta(seconds=ttl_seconds)
    return URLEntry(
        code=code,
        original_url="https://example.com/very/long/path",
        created_at=datetime.now(timezone.utc),
        expires_at=expires_at,
        max_clicks=max_clicks,
    )


async def test_save_and_get():
    store = URLStore()
    await store.save(make_entry("abc"))
    entry = await store.get("abc")
    assert entry is not None
    assert entry.code == "abc"

async def test_get_missing_returns_none():
    store = URLStore()
    assert await store.get("missing") is None

async def test_exists():
    store = URLStore()
    await store.save(make_entry("xyz"))
    assert await store.exists("xyz") is True
    assert await store.exists("nope") is False

async def test_expired_entry_returns_none():
    store = URLStore()
    entry = make_entry("exp", ttl_seconds=0.01)
    await store.save(entry)
    await asyncio.sleep(0.02)
    assert await store.get("exp") is None

async def test_expired_entry_evicted_on_get():
    store = URLStore()
    await store.save(make_entry("exp2", ttl_seconds=0.01))
    assert len(store) == 1
    await asyncio.sleep(0.02)
    await store.get("exp2")
    assert len(store) == 0

async def test_evict_expired():
    store = URLStore()
    await store.save(make_entry("live"))
    await store.save(make_entry("dead", ttl_seconds=0.01))
    await asyncio.sleep(0.02)
    evicted = await store.evict_expired()
    assert evicted == 1
    assert len(store) == 1
    assert await store.get("live") is not None

async def test_delete():
    store = URLStore()
    await store.save(make_entry("del"))
    assert await store.delete("del") is True
    assert await store.delete("del") is False
    assert await store.get("del") is None

async def test_record_click_increments_count():
    entry = make_entry()
    await entry.record_click(referrer="https://google.com", user_agent="Mozilla/5.0")
    await entry.record_click(referrer="https://twitter.com", user_agent=None)
    stats = await entry.get_stats()
    assert stats["click_count"] == 2

async def test_record_click_tracks_referrers():
    entry = make_entry()
    await entry.record_click("https://google.com/search", None)
    await entry.record_click("https://google.com/search", None)
    await entry.record_click("https://twitter.com", None)
    stats = await entry.get_stats()
    assert stats["referrers"]["google.com"] == 2
    assert stats["referrers"]["twitter.com"] == 1

async def test_click_ring_buffer_drops_oldest():
    entry = make_entry(max_clicks=3)
    for i in range(5):
        await entry.record_click(f"https://ref{i}.com", None)
    stats = await entry.get_stats()
    
    assert stats["click_count"] == 5
    assert len(stats["recent_clicks"]) == 3

async def test_concurrent_click_recording_exact_count():
    """Fire 200 concurrent click coroutines and assert the count is exactly 200, verifying the asyncio.Lock prevents lost updates."""
    entry = make_entry(max_clicks=1000)
    await asyncio.gather(*[entry.record_click(None, None) for _ in range(200)])
    stats = await entry.get_stats()
    assert stats["click_count"] == 200
