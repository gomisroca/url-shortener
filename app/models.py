from __future__ import annotations
 
from datetime import datetime
from typing import Any
 
from pydantic import BaseModel, HttpUrl, field_validator


class ShortenRequest(BaseModel):
    url: HttpUrl
    # Optional caller-supplied alias, e.g. "my-blog-post". If omitted, a
    # random code is generated. If the alias is already taken, 409 is returned.
    custom_code: str | None = None
    # Per-link TTL in seconds. Overrides the server default. 0 = no expiry.
    ttl_seconds: float | None = None
 
    @field_validator("custom_code")
    @classmethod
    def validate_custom_code(cls, v: str | None) -> str | None:
        if v is None:
            return v
        if not v.isalnum() and not all(c in "-_" for c in v if not c.isalnum()):
            raise ValueError("custom_code may only contain letters, numbers, hyphens, and underscores")
        if len(v) < 2 or len(v) > 64:
            raise ValueError("custom_code must be between 2 and 64 characters")
        return v
    

class ShortenResponse(BaseModel):
    code: str
    short_url: str
    original_url: str
    expires_at: datetime | None


class ClickEvent(BaseModel):
    timestamp: datetime
    referrer: str | None
    user_agent: str | None


class StatsResponse(BaseModel):
    code: str
    original_url: str
    created_at: datetime
    expires_at: datetime | None
    click_count: int
    referrers: dict[str, int]
    recent_clicks: list[ClickEvent]