from __future__ import annotations

import os
from dataclasses import dataclass

@dataclass(frozen=True)
class Settings:
    port: int
    api_key: str

    # Length of generated short codes
    code_length: int
    # Retry attempts before giving up on random code generation
    max_collision_retries: int

    # Default link TTL.
    default_ttl_seconds: int
    # Click events kept per URL
    max_clicks_per_url: int
    # Prepended to codes in responses
    base_url: str

def load_settings() -> Settings:
    return Settings(
        port=int(os.environ.get("PORT", "8080")),
        api_key=os.environ.get("API_KEY", ""),
        code_length=int(os.environ.get("CODE_LENGTH", "6")),
        max_collision_retries=int(os.environ.get("MAX_COLLISION_RETRIES", "5")),
        default_ttl_seconds=int(os.environ.get("DEFAULT_TTL_SECONDS", "0")),
        max_clicks_per_url=int(os.environ.get("MAX_CLICKS_PER_URL", "1000")),
        base_url=os.environ.get("BASE_URL", "http://localhost:8080"),
    )