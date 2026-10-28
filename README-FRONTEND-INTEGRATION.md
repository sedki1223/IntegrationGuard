# Integration Guard frontend bundle

This bundle adds a React + TypeScript dashboard to the existing Go API + PostgreSQL project. It deliberately does not replace your existing `docker-compose.yml`.

## Install files

1. Extract `frontend/` and `Dockerfile.frontend` into the root of `C:\Users\sedki\IntegrationGuard`.
2. Merge the service from `compose-frontend-service.yml` under `services:` in your existing `docker-compose.yml` (same indentation level as `api` and `postgres`). Do not replace the existing compose file.
3. Run from the project root:

```powershell
docker compose config
docker compose up --build -d
```

4. Open http://localhost:5173.

Nginx serves the compiled UI and proxies `/api/*` to the existing Compose service named `api`, so browser requests remain same-origin. The dashboard supports overview, integration listing/creation, JSON payload validation and file upload, report inspection, and saved run history.

## Notes

- API routes and response shapes are based on the existing API described in the current project workflow.
- Integration creation expects the raw `config.Config` object: `name`, `rules`, and optional `match_key`.
- A validation with issues returns HTTP 422. The UI surfaces the API error and refreshes saved run history; it does not yet render the returned 422 report inline. Use Run history to inspect that saved report.
- The initial dashboard is a local developer UI, not an authentication-secured multi-user production deployment.
