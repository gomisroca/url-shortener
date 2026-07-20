from __future__ import annotations
 
import importlib
 
import pytest
from fastapi.testclient import TestClient


def _build_app(monkeypatch, **env):
    for k, v in env.items():
        monkeypatch.setenv(k, v)
    import app.config
    import app.main
    importlib.reload(app.config)
    importlib.reload(app.main)
    return app.main.app


class TestHealth:
    def test_health(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app) as client:
            res = client.get("/health")
            assert res.status_code == 200
            assert res.json()["status"] == "ok"


class TestShorten:
    def test_shorten_returns_short_url(self, monkeypatch):
        app = _build_app(monkeypatch, BASE_URL="http://short.test")
        with TestClient(app) as client:
            res = client.post("/shorten", json={"url": "https://example.com/very/long"})
        assert res.status_code == 200
        body = res.json()
        assert body["short_url"].startswith("http://short.test/r/")
        assert body["original_url"] == "https://example.com/very/long"
        assert body["code"]
        assert body["expires_at"] is None
 
    def test_shorten_code_length_respected(self, monkeypatch):
        app = _build_app(monkeypatch, CODE_LENGTH="8")
        with TestClient(app) as client:
            res = client.post("/shorten", json={"url": "https://example.com"})
        assert len(res.json()["code"]) == 8
 
    def test_shorten_custom_code(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app) as client:
            res = client.post("/shorten", json={"url": "https://example.com", "custom_code": "my-link"})
        assert res.status_code == 200
        assert res.json()["code"] == "my-link"
 
    def test_shorten_custom_code_conflict_returns_409(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app) as client:
            client.post("/shorten", json={"url": "https://example.com", "custom_code": "clash"})
            res = client.post("/shorten", json={"url": "https://other.com", "custom_code": "clash"})
        assert res.status_code == 409
 
    def test_shorten_with_ttl(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app) as client:
            res = client.post("/shorten", json={"url": "https://example.com", "ttl_seconds": 3600})
        assert res.json()["expires_at"] is not None
 
    def test_shorten_invalid_url_returns_422(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app) as client:
            res = client.post("/shorten", json={"url": "not-a-url"})
        assert res.status_code == 422
 
    def test_shorten_requires_api_key_when_set(self, monkeypatch):
        app = _build_app(monkeypatch, API_KEY="secret")
        with TestClient(app) as client:
            res = client.post("/shorten", json={"url": "https://example.com"})
            assert res.status_code == 401
            res = client.post("/shorten",
                              json={"url": "https://example.com"},
                              headers={"X-API-Key": "secret"})
            assert res.status_code == 200
 
 
class TestRedirect:
    def test_redirect_follows_to_original(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app, follow_redirects=False) as client:
            shorten_res = client.post("/shorten", json={"url": "https://example.com/dest"})
            code = shorten_res.json()["code"]
            res = client.get(f"/r/{code}")
        assert res.status_code == 302
        assert res.headers["location"] == "https://example.com/dest"
 
    def test_redirect_missing_code_returns_404(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app) as client:
            res = client.get("/r/doesnotexist")
        assert res.status_code == 404
 
    def test_redirect_no_api_key_needed(self, monkeypatch):
        app = _build_app(monkeypatch, API_KEY="secret")
        with TestClient(app, follow_redirects=False) as client:
            shorten_res = client.post("/shorten",
                                      json={"url": "https://example.com"},
                                      headers={"X-API-Key": "secret"})
            code = shorten_res.json()["code"]
            res = client.get(f"/r/{code}")
        assert res.status_code == 302
 
 
class TestStats:
    def test_stats_after_clicks(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app, follow_redirects=False) as client:
            code = client.post("/shorten",
                               json={"url": "https://example.com"}).json()["code"]
            client.get(f"/r/{code}", headers={"Referer": "https://google.com"})
            client.get(f"/r/{code}", headers={"Referer": "https://google.com"})
            client.get(f"/r/{code}", headers={"Referer": "https://twitter.com"})
            res = client.get(f"/stats/{code}")
        assert res.status_code == 200
        body = res.json()
        assert body["click_count"] == 3
        assert body["referrers"]["google.com"] == 2
        assert body["referrers"]["twitter.com"] == 1
        assert len(body["recent_clicks"]) == 3
 
    def test_stats_missing_code_returns_404(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app) as client:
            res = client.get("/stats/nope")
        assert res.status_code == 404
 
    def test_stats_requires_api_key_when_set(self, monkeypatch):
        app = _build_app(monkeypatch, API_KEY="secret")
        with TestClient(app) as client:
            code = client.post("/shorten",
                               json={"url": "https://example.com"},
                               headers={"X-API-Key": "secret"}).json()["code"]
            assert client.get(f"/stats/{code}").status_code == 401
            assert client.get(f"/stats/{code}",
                              headers={"X-API-Key": "secret"}).status_code == 200


class TestDelete:
    def test_delete_link(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app, follow_redirects=False) as client:
            code = client.post("/shorten",
                               json={"url": "https://example.com"}).json()["code"]
            res = client.delete(f"/links/{code}")
            assert res.status_code == 200
            assert res.json()["deleted"] == code
            assert client.get(f"/r/{code}").status_code == 404
 
    def test_delete_missing_returns_404(self, monkeypatch):
        app = _build_app(monkeypatch)
        with TestClient(app) as client:
            assert client.delete("/links/ghost").status_code == 404

