"""
In-memory store for shortened URLs and their click analytics.
 
Two concurrency points worth noting:
 
1. Click counting uses an asyncio.Lock per URL entry rather than one global
   lock, so a burst of clicks on one URL doesn't block reads/writes to any
   other URL. This is the same per-key locking pattern used in the rate
   limiter's TokenBucket.
 
2. Click events are recorded *after* the redirect response is sent
   (via FastAPI BackgroundTasks), so the hot-path redirect latency is
   never affected by analytics bookkeeping. The store is designed for
   that: record_click() is async and safe to call from a background task.
"""

from __future__ import annotations
import asyncio
from collections import defaultdict
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Optional
from urllib.parse import urlparse

from app.models import ClickEvent

@dataclass
class URLEntry:
    code: str
    original_url: str
    created_at: datetime
    expires_at: Optional[datetime]
    max_clicks: int

    # Protected by _lock
    click_count: int = 0
    clicks: list[ClickEvent] = field(default_factory=list)
    referrer_counts: dict[str, int] = field(default_factory=lambda: defaultdict(int))
    _lock: asyncio.Lock = field(default_factory=asyncio.Lock, repr=False)

    def is_expired(self) -> bool:
        if self.expires_at is None:
            return False
        return datetime.now(timezone.utc) > self.expires_at
    
    async def record_click(self, referrer: str | None, user_agent: str | None) -> None:
        async with self._lock:
            self.click_count += 1

            event = ClickEvent(
                timestamp=datetime.now(timezone.utc),
                referrer=referrer,
                user_agent=user_agent,
            )
            self.clicks.append(event)
            # Ring-buffer: drop oldest clicks beyond the cap
            if len(self.clicks) > self.max_clicks:
                self.clicks = self.clicks[-self.max_clicks:]

            if referrer:
                domain = _extract_domain(referrer)
                self.referrer_counts[domain] += 1

    async def get_stats(self) -> dict:
        async with self._lock:
            return {
                "click_count": self.click_count,
                "referrers": dict(self.referrer_counts),
                "recent_clicks": list(reversed(self.clicks)),
            }
        

def _extract_domain(referrer: str) -> str:
    try:
        parsed = urlparse(referrer)
        return parsed.netloc or referrer
    except Exception:
        return referrer


class URLStore:
    def __init__(self) -> None:
        self._entries: dict[str, URLEntry] = {}
        self._lock = asyncio.Lock()

    async def save(self, entry: URLEntry) -> None:
        async with self._lock:
            self._entries[entry.code] = entry

    async def get(self, code: str) -> URLEntry | None:
        """Returns the entry if it exists and hasn't expired."""
        async with self._lock:
            entry = self._entries.get(code)
        if entry is None:
            return None
        if entry.is_expired():
            async with self._lock:
                self._entries.pop(code, None)
            return None
        return entry
    
    async def exists(self, code: str) -> bool:
        async with self._lock:
            return code in self._entries
        
    async def delete(self, code: str) -> bool:
        async with self._lock:
            self._entries.pop(code, None) is not None

    async def evict_expired(self) -> int:
        now = datetime.now(timezone.utc)
        async with self._lock:
            stale = [
                code for code, entry in self._entries.items()
                if entry.expires_at is not None and now > entry.expires_at
            ]
            for code in stale:
                del self._entries[code]
        return len(stale)
    
    def __len__(self) -> int:
        return len(self._entries)
