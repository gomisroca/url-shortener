# URL Shortener (Python)

A standalone URL shortener microservice. POST a long URL, get back a short
code. The redirect endpoint is the hot path - it's just a dict lookup and
a 302, with click analytics recorded _after_ the response is sent so they
never add latency.

## Running it

```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8080
```

### Tests

```bash
pytest -v
```

34 tests covering code generation (including forced-collision retry),
store TTL expiry, concurrent click counting, and the full API.

### Docker

```bash
docker build -t url-shortener .
docker run -p 8080:8080 \
  -e API_KEY=your-secret \
  -e BASE_URL=https://short.example.com \
  url-shortener
```

## Config

| Variable                | Default                 | Description                                             |
| ----------------------- | ----------------------- | ------------------------------------------------------- |
| `PORT`                  | `8080`                  | HTTP port                                               |
| `API_KEY`               | _(empty)_               | If set, required on `/shorten`, `/stats`, `/links`      |
| `BASE_URL`              | `http://localhost:8080` | Prepended to codes in responses                         |
| `CODE_LENGTH`           | `6`                     | Length of generated random codes                        |
| `MAX_COLLISION_RETRIES` | `5`                     | Retry attempts before giving up on random generation    |
| `DEFAULT_TTL_SECONDS`   | `0`                     | Default link TTL. `0` = no expiry                       |
| `MAX_CLICKS_PER_URL`    | `1000`                  | Click events kept per URL (ring buffer, oldest dropped) |

## API

### `POST /shorten`

```json
{
  "url": "https://example.com/very/long/path",
  "custom_code": "my-link",
  "ttl_seconds": 86400
}
```

`custom_code` and `ttl_seconds` are both optional. Returns `409` if the
custom code is already taken.

**Response:**

```json
{
  "code": "my-link",
  "short_url": "https://short.example.com/r/my-link",
  "original_url": "https://example.com/very/long/path",
  "expires_at": "2026-07-17T07:00:00Z"
}
```

### `GET /r/{code}`

Redirects `302` to the original URL. No API key required - this is the
public-facing hot path. Click analytics (referrer, user-agent, timestamp)
are recorded in a background task after the response is sent, so redirect
latency is never affected by analytics bookkeeping.

Returns `404` if the code doesn't exist or has expired.

### `GET /stats/{code}`

```json
{
  "code": "my-link",
  "original_url": "https://example.com/very/long/path",
  "created_at": "2026-07-16T07:00:00Z",
  "expires_at": null,
  "click_count": 42,
  "referrers": {
    "google.com": 28,
    "twitter.com": 14
  },
  "recent_clicks": [
    {
      "timestamp": "2026-07-16T08:00:00Z",
      "referrer": "https://google.com/search?q=...",
      "user_agent": "Mozilla/5.0 ..."
    }
  ]
}
```

### `DELETE /links/{code}`

Deletes the link immediately. Returns `404` if not found.

### `GET /health`

`{"status": "ok", "links": "42"}` - always open, no API key required.

## Design notes

**Why BackgroundTasks for click recording?**
The redirect endpoint is the only one that will see real traffic at scale

- potentially millions of hits per day. Recording analytics synchronously
  (before sending the 302) would add the cost of an asyncio.Lock acquisition
  and a list append to every redirect. Moving it to a BackgroundTask means
  the response goes out immediately and the bookkeeping runs after, at no
  cost to the client.

**Why per-URL locks instead of one global lock?**
A single global lock on the store would serialize all click recording
across all URLs. With per-entry locks (`URLEntry._lock`), a burst of
clicks on one popular URL only blocks other clicks to _that_ URL, not to
any other. Same pattern as the rate limiter's per-key `TokenBucket._lock`.

**Why random codes instead of hashing the URL?**
A hash of the URL would be deterministic - two people shortening the same
URL would get the same code. That sounds nice but means you can't have two
different short links pointing to the same destination (e.g. for A/B
tracking). Random codes avoid this, and 6-char base64url gives ~68 billion
possible codes, making collision probability negligible in any realistic
deployment.

## Possible next steps (and the Go port)

- **Go port** - same service translated to Go. The interesting differences:
  `sync.Mutex` per entry instead of `asyncio.Lock`, and `net/http`'s
  `http.Handler` doesn't have a native BackgroundTasks equivalent - you'd
  spawn a goroutine for click recording instead, which is even cheaper.
- **Persistent storage** - back the store with Redis (SETEX for TTL, INCR
  for atomic click counting) or Postgres.
- **Custom domains** - let each user bring their own short domain.
- **QR code endpoint** - `GET /qr/{code}` returns a QR code image for the
  short URL, a natural fit for the image-upload-service's pipeline.
