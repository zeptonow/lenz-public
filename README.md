# Lenz

Lenz is a session replay + product analytics platform with a web dashboard, ingestion/processing services, and SDKs/trackers.

## Architecture overview

Lenz is organized as a multi-component repository:

- **Dashboard UI** (`frontend/`): the web app used to configure projects and analyze sessions.
- **API** (`api/`): Python service that serves the dashboard API (auth, projects, sessions, integrations, etc.).
- **Backend services** (`backend/`): Go services that power ingestion, processing, and storage workflows (multiple binaries under `backend/cmd/`).
- **Trackers / SDKs** (`tracker/`, `snippet/`): client-side libraries used to capture session data and send it to Lenz.
- **Utilities** (`sourcemap-uploader/`, `networkProxy/`, `spot/`, `mobs/`): supporting tooling and modules.

## Quickstart (local dev)

This repo supports multiple deployment models; the fastest local path is running **API + UI**.

### Prerequisites

- Node.js + yarn (see `frontend/package.json`)
- Python (see `api/requirements.txt`)
- Postgres and Redis available locally (or via Docker)

### 1) Start dependencies (example using Docker)

```bash
docker run --rm -d --name lenz-postgres -e POSTGRES_PASSWORD=secret -p 5432:5432 postgres:16
docker run --rm -d --name lenz-redis -p 6379:6379 redis:7
```

### 2) Configure and run the API

```bash
cd api
./prepare-dev.sh            # creates ./api/.env from env.dev if missing
./run-dev.sh                # runs the API (uvicorn) with autoreload
```

If you need the full configuration reference, see `docs/CONFIGURATION.md`.

### 3) Install and run the dashboard UI

```bash
cd frontend
yarn install
make start-fg
```

## API reference

- **Backend HTTP ingestion endpoints**: see `backend/cmd/http/README.md`.
- **Python API routes**: see `api/routers/` (FastAPI routers).

## Contributing

See `CONTRIBUTING.md` for:

- local setup and workflows
- code style and testing requirements
- pull request process

## Community

- **Code of Conduct**: `CODE_OF_CONDUCT.md`
- **Security**: please follow the repository’s security policy/process (or open a private security report via GitHub Security Advisories if enabled for the repo).

## Links

- **Repository**: `https://github.com/zeptonow/lenZ`
- **Issues**: `https://github.com/zeptonow/lenZ/issues`
- **Pull requests**: `https://github.com/zeptonow/lenZ/pulls`

## License

See `LICENSE`.

