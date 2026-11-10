# Brothers REST API

REST API for the public site data and a protected admin interface. The API uses SQLite (plain SQL, no ORM) and stores the site configuration, zones, tariffs, specifications, booking time step, and refresh sessions.

## Run

Required environment variables:

- `ADMIN_USERNAME` — initial admin login.
- `ADMIN_PASSWORD` — initial admin password, at least 12 characters.
- `JWT_SECRET` — secret for signing JWTs, at least 32 characters.

Optional:

- `HTTP_ADDR` — listen address; defaults to `0.0.0.0:8080`.
- `DATABASE_PATH` — SQLite file; defaults to `data/brothers.db`.
- `CORS_ALLOWED_ORIGINS` — comma-separated exact browser origins, for example `https://admin.example.com,http://localhost:5173`.

PowerShell example:

```powershell
$env:ADMIN_USERNAME = "admin"
$env:ADMIN_PASSWORD = "replace-with-a-long-password"
$env:JWT_SECRET = "replace-with-at-least-32-random-characters"
$env:CORS_ALLOWED_ORIGINS = "http://localhost:5173"
go run ./cmd/app
```

The database is initialized and seeded with the supplied site configuration, seven zones and `timeStep: 30` on first run. Existing records are not overwritten on restart. The process creates the database directory if needed.

## Authentication

Content `GET` routes and `GET /healthz` are public. Login and refresh use the credentials or token described below. All content changes require an access token.

- `POST /api/v1/auth/login` — body `{"username":"admin","password":"..."}`; returns `accessToken`, `refreshToken`, `tokenType`, and `expiresIn` (seconds).
- `POST /api/v1/auth/refresh` — body `{"refreshToken":"..."}`; returns a new token pair. The old refresh token is invalidated on rotation.
- `POST /api/v1/auth/logout` — body `{"refreshToken":"..."}`; revokes that refresh token.

Send an `Authorization` header using the `Bearer` scheme and the access token to protected endpoints. Access tokens expire in 15 minutes; refresh tokens expire in 7 days and their hashes are persisted in SQLite.

## Content endpoints

Public `GET` endpoints return the same data used by the admin interface. The corresponding writes require an access token.

| Method | Path | Purpose |
|---|---|---|
| GET, PUT | `/api/v1/config` | Read or replace contact and location data |
| GET, PUT | `/api/v1/settings/time-step` | Read or change the booking time step (`15`, `30`, or `60`) |
| GET, POST | `/api/v1/zones` | List zones or create a zone |
| GET, PUT, DELETE | `/api/v1/zones/{id}` | Read, replace, or delete one zone |
| GET | `/healthz` | Liveness check |

`PUT` replaces the complete resource. A zone contains `id`, `name`, `nameKk`, `size`, `seats` (number or `null`), `desc`, `descKk`, `tariff` (`wd` and `we`, each with `h1`, `h3`, `h5`, `day`, `night`), `image`, and `specs` (array of `{ "name": "...", "value": "..." }`). Zone IDs use lowercase letters, digits, `_` and `-`; an update ID must match the path.

Successful reads and updates return JSON. Zone creation returns `201 Created` and a `Location` header; deletion and logout return `204 No Content`. Errors use `{"error":"..."}` with standard `400`, `401`, `404`, `409`, or `500` status codes.
