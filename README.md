# URL Shortener (Go)

A Go port of [`URL Shortener (Python)`](https://github.com/gomisroca/url-shortener/tree/python). Same HTTP API
contract, swap one for the other and nothing calling it needs to change.

## Running it

```bash
go run .
# or
go build -o server . && ./server
```

### Tests

```bash
go test -race ./...
```

### Docker

```bash
docker build -t url-shortener .
docker run -p 8080:8080 \
  -e API_KEY=your-secret \
  -e BASE_URL=https://short.example.com \
  url-shortener
```

## Config

Same env vars as the Python version - `PORT`, `API_KEY`, `BASE_URL`,
`CODE_LENGTH`, `MAX_COLLISION_RETRIES`, `DEFAULT_TTL_SECONDS`,
`MAX_CLICKS_PER_URL`, `CLEANUP_INTERVAL_SECONDS`.

## API

Same as the Python version - `POST /shorten`, `GET /r/{code}`,
`GET /stats/{code}`, `DELETE /links/{code}`, `GET /health`.
See the Python README for the full API reference.
