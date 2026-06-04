# Contributing to Lenz

Thanks for taking the time to contribute to **Lenz**. Contributions of all sizes are welcome: bug fixes, features, tests, docs, and tooling.

## Code of Conduct

By participating in this project, you are expected to uphold our [Code of Conduct](CODE_OF_CONDUCT.md).

## Where to start

- Browse the issue tracker in `https://github.com/zeptonow/lenZ/issues`.
- If you want to work on an existing issue, leave a comment so others don’t duplicate the effort.
- If you’re proposing a larger change, open an issue first to align on approach and scope.

## Development setup (run locally)

The quickest local loop is **API + UI**:

```bash
# 1) API
cd api
./prepare-dev.sh
./run-dev.sh

# 2) UI
cd ../frontend
yarn install
make start-fg
```

For the full configuration reference (all env vars and runtime config sources), see `docs/CONFIGURATION.md`.

## Code style

- **Go** (`backend/`)
  - Format: `gofmt` (required)
  - Lint: keep changes idiomatic; avoid introducing new exported identifiers without docstrings
- **TypeScript/JavaScript** (`frontend/`, `tracker/`, tools)
  - Follow existing ESLint rules. Run `yarn lint` where available (for the dashboard UI: `cd frontend && yarn lint`).
- **Python** (`api/`)
  - Keep changes consistent with existing patterns (FastAPI + decouple-based config).
  - Prefer small, well-scoped modules and explicit typing where it improves clarity.

## Testing requirements

At minimum, PRs should include:

- **A test plan in the PR description** (what you ran / verified).
- **Automated tests** for non-trivial bug fixes or new logic.

Common commands:

```bash
# Go (from repo root)
cd backend && go test ./...

# Frontend (from repo root)
cd frontend && yarn test:ci

# Python API (from repo root)
cd api && python -m unittest
```

## Pull request process

- **Branching**: create a feature branch from the default branch.
- **PR description**: explain *why* the change is needed and *what* it changes.
- **Link issues**: reference the issue with “Fixes #123” when appropriate.
- **Keep PRs reviewable**: prefer small PRs with focused scope.
- **No secrets**: do not commit credentials, tokens, or `.env` files.

## Security

Do not disclose security issues publicly. If you believe you have found a vulnerability, follow the repository’s security reporting process (or open a private report via GitHub Security Advisories if enabled).
