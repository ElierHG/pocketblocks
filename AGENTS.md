# AGENTS.md

## Cursor Cloud specific instructions

### Architecture

PocketBlocks is an open-source low-code app builder combining Openblocks (React frontend) with PocketBase (Go backend + embedded SQLite). Two components form the stack:

- **Server** (`server/`) — Go/PocketBase backend serving both APIs and embedded frontend assets on `:8090`. All Openblocks-format API routes (`/api/v1/...`, `/api/auth/...`, etc.) are implemented in `server/apis/openblocks.go`.
- **Client** (`client/`) — React/Vite frontend (Yarn Berry 3.2.4 monorepo with 8 workspace packages)

No external database is needed; PocketBase uses embedded SQLite. There is no proxy layer.

### Build flow

```
client (yarn build) → server/ui/dist/
server (go build)   → single binary with embedded frontend
```

### Running the server

```bash
cd server && go run main.go serve --http=0.0.0.0:8090
```

The admin panel is at `http://localhost:8090/_/`. The app UI is at `http://localhost:8090/`.

### Key caveats

- The client uses **Yarn Berry 3.2.4** (not classic Yarn). The `.yarnrc.yml` at `client/.yarnrc.yml` sets `nodeLinker: node-modules` and points to the bundled Yarn release at `.yarn/releases/yarn-3.2.4.cjs`. Run `yarn install` from the `client/` directory.
- Auth uses cookie-based JWT tokens (`pb_auth` cookie). The server validates tokens directly from cookies via `server/apis/openblocks.go` helper methods.
- Only admin users (created via `POST /api/admins`) have `orgDev=true` and can create apps.

### Lint

- **Server**: `cd server && go vet ./...`

### Tests

- **Client**: `cd client && yarn test` — 62/66 suites pass; 4 pre-existing failures
- **Server**: `cd server && go test ./...`
