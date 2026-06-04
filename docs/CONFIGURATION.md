# Configuration

This document is the **single source of truth** for configuration in **Lenz**.

## Configuration sources

- **Go backend services** (`backend/`): load configuration from `e.<stage>.yaml` (default) or Vault JSON files when `VAULT_ENABLED=true`.
- **Python API** (`api/`): loads from Vault JSON files when `VAULT_ENABLED=true`, otherwise from a local `.env` file (see `api/env.dev`, `api/env.default`).
- **Frontend** (`frontend/`): uses build/runtime environment variables for feature toggles + runtime hostname detection for API endpoints (see `frontend/app/config.ts`).
- **Sourcemap uploader** (`sourcemap-uploader/`): CLI/library config via env vars or CLI flags.

## Global environment variables

These variables affect multiple components.

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `VAULT_ENABLED` | boolean | `false` | `true` | Optional |
| `STAGE` | string | `local` | `d1` / `s1` / `p1` | Optional |
| `BASE` | string | `.` | `/app` | Optional |

## Go backend (`backend/`)

### How config is loaded

- If `VAULT_ENABLED=true`, config is read from:
  - `/vault/secrets/static.json` (required)
  - `/vault/secrets/dynamic.json` (optional)
- Otherwise, config is read from:
  - `e.<stage>.yaml` where `<stage>` comes from `STAGE` (example: `STAGE=d1` → `e.d1.yaml`)

### Backend-specific environment variables

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `HOSTNAME` | string | (none) | `api-0` | **Required** (non-Fargate) |
| `ECS_CONTAINER_METADATA_URI` | string | (none) | `http://169.254.170.2/v2/metadata` | Optional (AWS Fargate) |
| `DEBUG` | boolean (`"true"`/`"false"`) | (none) | `true` | Optional |
| `ENABLE_EXTRA_LOGS` | boolean (`"true"`/`"false"`) | (none) | `true` | Optional |
| `LOG_SHIPPING_ENABLED` | boolean (`"true"`/`"false"`) | (none) | `true` | Optional |
| `SENTRY_DSN` | string | (none) | `https://<key>@sentry.io/<project>` | Optional |
| `ELASTIC_HOST` | string | (none) | `https://es.example.com` | Optional |
| `ELASTIC_API_KEY` | string | (none) | `base64(apiKey)` | Optional |
| `DATADOG_API_KEY` | string | (none) | `dd_api_key` | Optional |

## Python API (`api/`)

### Local development

- Copy `api/env.dev` → `api/.env` (or run `api/prepare-dev.sh`), then edit required values.
- Start the API with `api/run-dev.sh`.

### Environment variables

The tables below list variables present in the repo’s env templates (`api/env.default`, `api/env.dev`, `api/env.staging`) and consumed via `python-decouple` (or Vault when enabled).

#### Runtime / URLs

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `SITE_URL` | string | (empty) | `http://localhost:3333` | **Required** |
| `ASSIST_URL` | string | (empty) | `http://127.0.0.1:9001/assist/%s` | Optional |
| `ASSIST_KEY` | string | (empty) | `assist-secret-key` | Optional |
| `sourcemaps_reader` | string | (empty) | `http://127.0.0.1:3000/sourcemaps` | Optional |
| `announcement_url` | string | (empty) | `https://status.example.com` | Optional |
| `docs_url` | string | `/docs` | `/docs` | Optional |
| `root_path` | string | `''` | `''` | Optional |
| `LISTEN_PORT` | number | `8000` | `8000` | Optional |
| `APP_NAME` | string | (empty) | `chalice` | Optional |

#### Authentication / JWT

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `JWT_SECRET` | string | (template value) | `set-a-random-secret` | **Required** |
| `JWT_ALGORITHM` | string | `HS512` | `HS512` | Optional |
| `JWT_EXPIRATION` | number (seconds) | `86400` | `86400` | Optional |
| `JWT_ISSUER` | string | (template value) | `lenz` | Optional |
| `JWT_REFRESH_SECRET` | string | (template value) | `set-a-random-secret` | **Required** |
| `JWT_REFRESH_EXPIRATION` | number (seconds) | `604800` | `604800` | Optional |
| `JWT_SPOT_SECRET` | string | (template value) | `set-a-random-secret` | **Required** |
| `JWT_SPOT_EXPIRATION` | number (seconds) | `3600` | `3600` | Optional |
| `JWT_SPOT_REFRESH_SECRET` | string | (template value) | `set-a-random-secret` | **Required** |
| `JWT_SPOT_REFRESH_EXPIRATION` | number (seconds) | `604800` | `604800` | Optional |
| `ASSIST_JWT_SECRET` | string | (empty) | `set-a-random-secret` | Optional |
| `ASSIST_JWT_EXPIRATION` | number (seconds) | `144000` | `144000` | Optional |

#### Interservice authentication (service-to-service)

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `GO_INTERSERVICE_API_KEY` | string | (empty) | `...` | Optional |
| `PY_INTERSERVICE_API_KEY` | string | (empty) | `...` | Optional |

#### Database / cache

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `pg_host` | string | (empty) | `127.0.0.1` | **Required** |
| `pg_port` | number | `5432` | `5432` | **Required** |
| `pg_dbname` | string | `postgres` | `postgres` | **Required** |
| `pg_user` | string | `postgres` | `postgres` | **Required** |
| `pg_password` | string | (template value) | `password` | **Required** |
| `PG_MAXCONN` | number | `20` | `20` | Optional |
| `PG_MINCONN` | number | `8` | `8` | Optional |
| `PG_POOL` | boolean | `true` | `true` | Optional |
| `PG_TIMEOUT` | number (seconds) | `30` | `30` | Optional |
| `PG_RETRY_INTERVAL` | number (seconds) | `2` | `2` | Optional |
| `PG_RETRY_MAX` | number | `20` | `20` | Optional |
| `REDIS_STRING` | string | (empty) | `redis://127.0.0.1:6379` | **Required** |

##### PostgreSQL least privilege (security requirement)

The `pg_user` used by the **Python API** must be a **least-privilege application role**. It must **not** be a superuser-like role and must **not** have broad built-in privileges that enable cross-tenant data exfiltration or infrastructure-level actions.

- **Hard requirements**: the app DB user must *not* be a member of roles like:
  - `pg_read_all_data`, `pg_write_all_data`
  - `pg_signal_backend`
  - `rds_replication`
  - `rds_superuser`, `rds_password` (AWS RDS-specific)

###### Quick audit (run as an operator/admin)

```sql
-- Show memberships for the application login role (replace :app_user).
SELECT r.rolname AS member_of
FROM pg_roles r
WHERE pg_has_role(:app_user, r.oid, 'member')
ORDER BY 1;

-- Check for logical replication slots (should be empty unless intentionally configured).
SELECT slot_name, plugin, slot_type, active, restart_lsn
FROM pg_replication_slots
ORDER BY slot_name;
```

###### Create a restricted application role (example)

The exact grants depend on which tables/functions the API needs. Start narrow and add grants as required by runtime errors.

```sql
-- Create a dedicated login for the API.
CREATE ROLE lenz_app LOGIN PASSWORD :strong_password;

-- Basic connectivity.
GRANT CONNECT ON DATABASE :db TO lenz_app;
GRANT USAGE ON SCHEMA public TO lenz_app;

-- Typical baseline for an app that reads/writes its own schema objects.
-- Prefer table-by-table grants; this broader pattern is an operational starting point.
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO lenz_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO lenz_app;

-- Ensure future tables/sequences inherit the same privileges (run as schema owner).
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO lenz_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO lenz_app;
```

###### Prevent secrets exposure by design

Do not grant the runtime application role access to long-lived secrets and auth material unless absolutely required:

- `users.api_key`, `tenants.api_key`
- password hashes
- invitation tokens / password reset tokens

Recommended patterns:

- **Separate schema**: move auth/secret tables into an `auth` schema and grant it only to a dedicated auth service/role.
- **Column-level privileges**: grant only the columns needed for the specific auth flows.
- **Views / stored procedures**: expose only the minimal fields required to validate an auth request.

###### Incident/runbook: remove a malicious replication slot

An unconsumed logical replication slot can prevent WAL cleanup and exhaust disk.

```sql
-- Drop a specific slot (replace :slot_name).
SELECT pg_drop_replication_slot(:slot_name);
```

#### Object storage (S3-compatible)

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `S3_HOST` | string | (empty) | `http://localhost:9000` | Optional |
| `S3_KEY` | string | (empty) | `minioadmin` | Optional |
| `S3_SECRET` | string | (empty) | `minioadmin` | Optional |
| `S3_DISABLE_SSL_VERIFY` | boolean | (empty) | `false` | Optional |
| `sessions_bucket` | string | `mobs` | `mobs` | Optional |
| `sessions_region` | string | `us-east-1` | `ap-south-1` | Optional |
| `sourcemaps_bucket` | string | `sourcemaps` | `sourcemaps` | Optional |
| `js_cache_bucket` | string | `sessions-assets` | `sessions-assets` | Optional |
| `IOS_VIDEO_BUCKET` | string | `mobs` | `mobs` | Optional |

#### ClickHouse (optional)

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `CH_ENABLED` | boolean | `false` | `true` | Optional |
| `CH_COMPRESSION` | boolean | `true` | `true` | Optional |
| `ch_host` | string | (empty) | `localhost` | Optional |
| `ch_port` | number | `9000` | `9000` | Optional |
| `ch_port_http` | number | `8123` | `8123` | Optional |
| `ch_timeout` | number (seconds) | `30` | `30` | Optional |
| `ch_receive_timeout` | number (seconds) | `10` | `10` | Optional |
| `CH_MAX_ROWS_TO_READ` | number | `200000000` | `200000000` | Optional |
| `CH_QUERY_RETRIES` | number | `1` | `1` | Optional |
| `CH_WAIT_FOR_CNX_POOL_S` | number (seconds) | `10` | `10` | Optional |

#### Email

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `EMAIL_FROM` | string | (template value) | `Lenz <do-not-reply@example.com>` | Optional |
| `EMAIL_HOST` | string | (empty) | `smtp.example.com` | Optional |
| `EMAIL_PORT` | number | `587` | `587` | Optional |
| `EMAIL_USER` | string | (empty) | `smtp-user` | Optional |
| `EMAIL_PASSWORD` | string | (empty) | `smtp-password` | Optional |
| `EMAIL_USE_TLS` | boolean | `true` | `true` | Optional |
| `EMAIL_USE_SSL` | boolean | `false` | `false` | Optional |
| `EMAIL_SSL_CERT` | string | (empty) | `/path/to/cert.pem` | Optional |
| `EMAIL_SSL_KEY` | string | (empty) | `/path/to/key.pem` | Optional |

#### Captcha (optional)

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `captcha_key` | string | (empty) | `site-key` | Optional |
| `captcha_server` | string | (empty) | `secret-key` | Optional |

#### Kafka (optional)

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `KAFKA_SERVERS` | string | (empty) | `localhost:9092` | Optional |
| `KAFKA_USE_SSL` | boolean | `false` | `false` | Optional |

#### Feature flags / misc

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `LOCAL_DEV` | boolean | `false` | `true` | Optional |
| `LOGLEVEL` | string | `INFO` | `INFO` | Optional |
| `S_LOGLEVEL` | string | `warning` | `warning` | Optional |
| `TZ` | string | `UTC` | `UTC` | Optional |
| `PYTHONUNBUFFERED` | number | `1` | `1` | Optional |
| `PRESIGNED_URL_EXPIRATION` | number (seconds) | `3600` | `3600` | Optional |
| `PRIVATE_ENDPOINTS` | boolean | `false` | `false` | Optional |
| `SCH_DELETE_DAYS` | number (days) | `30` | `30` | Optional |
| `EXP_CH_DRIVER` | boolean | `true` | `false` | Optional |
| `EXP_AUTOCOMPLETE` | boolean | `true` | `false` | Optional |
| `EXP_ALERTS` | boolean | `true` | `false` | Optional |
| `EXP_ERRORS_SEARCH` | boolean | `true` | `false` | Optional |
| `EXP_METRICS` | boolean | `true` | `false` | Optional |
| `EXP_SESSIONS_SEARCH` | boolean | `true` | `false` | Optional |
| `EXP_EVENTS` | boolean | `true` | `false` | Optional |

#### Storage patterns / routes (advanced)

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `FS_DIR` | string | (empty) | `/mnt/efs` | Optional |
| `CANVAS_PATTERN` | string | `%(sessionId)s/%(recordingId)s.tar.zst` | `%(sessionId)s/%(recordingId)s.tar.zst` | Optional |
| `SESSION_IOS_VIDEO_PATTERN` | string | `%(sessionId)s/replay.tar.zst` | `%(sessionId)s/replay.tar.zst` | Optional |
| `SESSION_MOB_PATTERN_S` | string | `%(sessionId)s/dom.mobs` | `%(sessionId)s/dom.mobs` | Optional |
| `SESSION_MOB_PATTERN_E` | string | `%(sessionId)s/dom.mobe` | `%(sessionId)s/dom.mobe` | Optional |
| `DEVTOOLS_MOB_PATTERN` | string | `%(sessionId)s/devtools.mob` | `%(sessionId)s/devtools.mob` | Optional |
| `EFS_SESSION_MOB_PATTERN` | string | `%(sessionId)s` | `%(sessionId)s` | Optional |
| `EFS_DEVTOOLS_MOB_PATTERN` | string | `%(sessionId)sdevtools` | `%(sessionId)sdevtools` | Optional |
| `assist` | string | `/sockets-live` | `/sockets-live` | Optional |
| `assistList` | string | `/sockets-list` | `/sockets-list` | Optional |
| `invitation_link` | string | `/api/users/invitation?token=%s` | `/api/users/invitation?token=%s` | Optional |
| `change_password_link` | string | `/reset-password?invitation=%s&&pass=%s` | `/reset-password?invitation=%s&&pass=%s` | Optional |

## Frontend (`frontend/`)

### Environment variables

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `NODE_ENV` | string | (varies) | `development` | **Required** |
| `API_EDP` | string | `http://localhost:3000` (local makefile) | `http://localhost:8000/api` | Optional |
| `PRODUCTION` | boolean | (derived) | `true` | Optional |
| `SOURCEMAP` | boolean/string | (none) | `true` | Optional |
| `KAI_TESTING` | boolean/string | (none) | `true` | Optional |
| `SENTRY_ENABLED` | boolean/string | `false` | `true` | Optional |
| `SENTRY_URL` | string | (none) | `https://<key>@sentry.io/<project>` | Optional |
| `CAPTCHA_ENABLED` | boolean/string | `false` | `true` | Optional |
| `CAPTCHA_SITE_KEY` | string | (none) | `site-key` | Optional |
| `MINIO_ENDPOINT` | string | (none) | `localhost` | Optional |
| `MINIO_PORT` | number/string | (none) | `9000` | Optional |
| `MINIO_USE_SSL` | boolean/string | (none) | `false` | Optional |
| `MINIO_ACCESS_KEY` | string | (none) | `minioadmin` | Optional |
| `MINIO_SECRET_KEY` | string | (none) | `minioadmin` | Optional |
| `VERSION` | string | `1.25.0` | `1.25.0` | Optional |
| `TRACKER_ENABLED` | boolean/string | `false` | `true` | Optional |
| `TRACKER_VERSION` | string | `17.1.6` | `17.1.6` | Optional |
| `TRACKER_MAJOR_VERSION` | string | `17` | `17` | Optional |
| `TRACKER_PROJECT_KEY` | string | (none) | `project-key` | Optional |
| `TRACKER_HOST` | string | (none) | `https://tracker.example.com` | Optional |
| `COMMIT_HASH` | string | (none) | `abc1234` | Optional |
| `TEST_FOSS_LOGIN` | string | (none) | `test@example.com` | Optional (tests) |
| `TEST_FOSS_PASSWORD` | string | (none) | `password` | Optional (tests) |
| `STRIPE_KEY` | string | (none) | `pk_live_...` | Optional |
| `CRISP_KEY` | string | (none) | `crisp-key` | Optional |

### Runtime endpoint selection (no env vars required)

The frontend selects endpoints based on the browser hostname (local/qa/prod). For local overrides, set:

- `localStorage["__env_override"] = "local" | "qa" | "prod"`

## Sourcemap uploader (`sourcemap-uploader/`)

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `API_KEY` | string | (none) | `api-key` | Optional (if using CLI flags instead) |
| `PROJECT_KEY` | string | (none) | `project-key` | Optional (if using CLI flags instead) |
| `USERVER` | string | (none) | `http://localhost:8000/api` | Optional (if using CLI flags instead) |
| `TARGET_URL` | string | (none) | `https://myapp.com/static` | Optional (if using CLI flags instead) |

## Vault-based env generation (`generateEnv`)

This repo includes a helper script `generateEnv` that can fetch a stage config and write `e.<stage>.yaml` locally.

| Name | Type | Default | Example | Required |
|---|---:|---|---|---:|
| `SECRET_ENV` | string | `qa` | `production` / `qa` | Optional |
| `VAULT_ROLE_ID` | string | (none) | `...` | Optional |
| `VAULT_SECRET_ID` | string | (none) | `...` | Optional |
| `GITHUB_TOKEN` | string | (none) | `ghp_...` | Optional |

