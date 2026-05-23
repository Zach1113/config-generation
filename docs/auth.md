# Authentication

## Overview

The system supports local username/password authentication and optional OIDC login using the OAuth 2.0 authorization-code flow. Both flows issue the app's internal JWT and set it as an HttpOnly session cookie. Existing Bearer JWT requests remain supported for API compatibility and tests.

## Session Cookies

- Session cookie: `configgen_session` by default.
- CSRF cookie: `configgen_csrf`.
- SameSite: `Lax` by default.
- Secure: enabled by default, disabled in local `docker-compose.yml`.
- Domain: not set, so cookies are host-only.
- Path: `/`.

The frontend sends credentialed API requests and includes `X-CSRF-Token` on unsafe methods by copying the value from the readable `configgen_csrf` cookie. Cookie-authenticated `POST`, `PUT`, `PATCH`, and `DELETE` requests without a matching CSRF header receive `403`. Bearer-token requests are exempt for compatibility.

## Local Password Flow

`POST /api/auth/register` and `POST /api/auth/login` remain available unless disabled by environment config. On success, both endpoints set the session and CSRF cookies and return the existing response shape:

```json
{
  "token": "eyJhbG...",
  "user": {
    "id": 1,
    "username": "alice",
    "display_name": "Alice Smith",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

The returned token is retained for compatibility. The SPA no longer stores new tokens in `localStorage`.

## OIDC Flow

1. The frontend calls `GET /api/auth/config`.
2. If OIDC is enabled, the login page links to `GET /api/auth/oidc/login?return_to=/projects`.
3. The backend stores short-lived HttpOnly `state`, `nonce`, and return-path cookies, then redirects to the provider.
4. The provider redirects back to `GET /api/auth/oidc/callback`.
5. The backend validates state, exchanges the code, verifies the ID token, links or creates the local user, sets app session cookies, and redirects to the safe relative return path.

Auto-provisioned OIDC users:

- Use verified email as `username` when available.
- Fall back to a stable provider-derived username if email is missing, unverified, or already taken.
- Store `password_hash = ''`, so OIDC users cannot use password login unless explicitly updated later.
- Receive no roles automatically. They can sign in but need an admin to assign roles before they can access protected resources.

## Auth Endpoints

### `GET /api/auth/config`

Returns frontend feature flags:

```json
{
  "oidc_enabled": true,
  "oidc_provider_name": "Dex",
  "password_login_enabled": true,
  "registration_enabled": true
}
```

### `GET /api/auth/me`

Returns the current user for either cookie or Bearer authentication. Returns `401` when unauthenticated.

### `POST /api/auth/session`

Migration bridge for old SPA sessions. Accepts an existing Bearer JWT and sets the HttpOnly session cookie:

```json
{
  "token": "eyJhbG..."
}
```

### `POST /api/auth/logout`

Clears the session and CSRF cookies. If a session cookie is present, the request must include a valid `X-CSRF-Token` header.

## Environment

| Variable | Default | Description |
|---|---:|---|
| `JWT_SECRET` | required | Signs internal app JWTs |
| `OIDC_ENABLED` | `false` | Enables OIDC login |
| `OIDC_ISSUER_URL` | | OIDC issuer discovery URL |
| `OIDC_CLIENT_ID` | | OIDC client ID |
| `OIDC_CLIENT_SECRET` | | OIDC client secret |
| `OIDC_REDIRECT_URL` | | Callback URL registered with the provider |
| `OIDC_BROWSER_AUTH_URL` | | Optional browser-facing authorization URL override for local Docker setups |
| `OIDC_SCOPES` | `openid email profile` | Space-separated provider scopes |
| `OIDC_PROVIDER_NAME` | `SSO` | Login button/provider display name |
| `SESSION_COOKIE_NAME` | `configgen_session` | App session cookie name |
| `SESSION_COOKIE_SECURE` | `true` | Whether cookies require HTTPS |
| `SESSION_COOKIE_SAMESITE` | `Lax` | `Lax`, `Strict`, or `None` |
| `PASSWORD_LOGIN_ENABLED` | `true` | Enables local password login |
| `REGISTRATION_ENABLED` | `true` | Enables public registration |

## Local Dex

`docker-compose.yml` includes a Dex provider for local testing. Open the app at:

```text
http://localhost:3000
```

The registered callback is:

```text
http://localhost:3000/api/auth/oidc/callback
```

The backend discovers Dex on the Docker network at `http://dex:5556/dex`, while the browser is sent to `http://localhost:5556/dex/auth`.

Static test users are configured in `dev/dex/config.yaml`. The local password for both users is `password`.

## Admin User

A superuser admin account is still created on startup when `ADMIN_USERNAME` and `ADMIN_PASSWORD` are set. The default compose credentials remain `admin` / `admin`.
