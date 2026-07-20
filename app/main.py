"""
uvicorn app.main:app --host 0.0.0.0 --port 8080
 
POST /shorten       - write path, protected by API key
GET  /r/{code}      - hot redirect path, no auth, analytics recorded in background
GET  /stats/{code}  - read path, protected by API key
GET  /health        - liveness check
"""
 
from __future__ import annotations

from contextlib import asynccontextmanager
from datetime import datetime, timedelta, timezone
 
from fastapi import BackgroundTasks, Depends, FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import RedirectResponse
 
from app.auth import make_api_key_dependency
from app.config import load_settings
from app.models import ShortenRequest, ShortenResponse, StatsResponse
from app.shortener import CodeConflictError, CodeGenerationError, resolve_code
from app.store import URLEntry, URLStore
 
settings = load_settings()
store = URLStore()

@asynccontextmanager
async def lifespan(_: FastAPI):
    if not settings.api_key:
        print("WARNING: API_KEY is not set - /shorten and /stats are open.")
    print(f"URL Shortener starting on :{settings.port} (base_url={settings.base_url})")
    yield

app = FastAPI(title="URL Shortener", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET", "POST", "DELETE", "OPTIONS"],
    allow_headers=["Content-Type", "X-API-Key"],
)

require_api_key = make_api_key_dependency(settings.api_key)

@app.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok", "links": str(len(store))}

@app.post("/shorten", dependencies=[require_api_key])
async def shorten(req: ShortenRequest) -> ShortenResponse:
    try:
        code = await resolve_code(
            store,
            custom_code=req.custom_code,
            code_length=settings.code_length,
            max_retries=settings.max_retries,
        )
    except CodeConflictError as e:
        raise HTTPException(status_code=409, detail=str(e))
    except CodeGenerationError as e:
        raise HTTPException(status_code=503, detail=str(e))

    ttl = req.ttl_seconds if req.ttl_seconds is not None else settings.default_ttl_seconds
    expires_at = None
    if ttl and ttl > 0:
        expires_at = datetime.now(timezone.utc) + timedelta(seconds=ttl)

    entry = URLEntry(
        code=code,
        original_url=str(req.url),
        created_at=datetime.now(timezone.utc),
        expires_at=expires_at,
        max_clicks=settings.max_clicks_per_url,
    )
    await store.save(entry)

    return ShortenResponse(
        code=code,
        short_url=f"{settings.base_url}/r/{code}",
        original_url=str(req.url),
        expires_at=expires_at,
    )


@app.get("/r/{code}")
async def redirect(code: str, request: Request, background_tasks: BackgroundTasks) -> RedirectResponse:
    """
    Just a dict lookup and a 302. Analytics are recorded after
    the response is sent so they never add latency here.
    """
    entry = await store.get(code)
    if entry is None:
        raise HTTPException(status_code=404, detail="short URL not found or expired")
 
    referrer = request.headers.get("referer")
    user_agent = request.headers.get("user-agent")
    background_tasks.add_task(entry.record_click, referrer, user_agent)
 
    return RedirectResponse(url=entry.original_url, status_code=302)


app.get("/stats/{code}", dependencies=[Depends(require_api_key)])
async def stats(code: str) -> StatsResponse:
    entry = await store.get(code)
    if entry is None:
        raise HTTPException(status_code=404, detail="short URL not found or expired")
 
    click_stats = await entry.get_stats()
    return StatsResponse(
        code=entry.code,
        original_url=entry.original_url,
        created_at=entry.created_at,
        expires_at=entry.expires_at,
        click_count=click_stats["click_count"],
        referrers=click_stats["referrers"],
        recent_clicks=click_stats["recent_clicks"],
    )


@app.delete("/links/{code}", dependencies=[Depends(require_api_key)])
async def delete_link(code: str) -> dict[str, str]:
    deleted = await store.delete(code)
    if not deleted:
        raise HTTPException(status_code=404, detail="short URL not found")
    return {"deleted": code}