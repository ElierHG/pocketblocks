# AGENTS.md

## Cursor Cloud specific instructions

### Architecture

PocketBlocks is an open-source low-code app builder combining Openblocks (React frontend) with PocketBase (Go backend + embedded SQLite). Three components form the stack:

- **Server** (`server/`) — Go/PocketBase backend serving APIs and embedded frontend assets on `:8090`
- **Client** (`client/`) — React/Vite frontend (Yarn Berry 3.2.4 monorepo with 8 workspace packages)
- **Proxy** (`proxy/`) — TypeScript API bridge that translates Openblocks API calls to PocketBase SDK calls

No external database is needed; PocketBase uses embedded SQLite.

### Build flow (production-like)

```
client (yarn build) → proxy/public/
proxy (yarn build)  → server/ui/dist/
server (go build)   → single binary with embedded frontend
```

### Running the server

```bash
cd server && go run main.go serve --http=0.0.0.0:8090
```

The admin panel is at `http://localhost:8090/_/`. The app UI is at `http://localhost:8090/`.

### Key caveats

- The client uses **Yarn Berry 3.2.4** (not classic Yarn). The `.yarnrc.yml` at `client/.yarnrc.yml` sets `nodeLinker: node-modules` and points to the bundled Yarn release at `.yarn/releases/yarn-3.2.4.cjs`. Run `yarn install` from the `client/` directory (it uses the local Yarn Berry automatically).
- The proxy uses **classic Yarn** (global `yarn` v1.x).
- Only admin users (created via `POST /api/admins`) have `orgDev=true` and can create apps. Regular users registered through the Sign Up page are non-dev members by default.
- The `proxy/` Vite config uses `vite ^7.x`, while the `client/` uses `vite ^5.x`; these are separate and intentional.

### Lint

- **Proxy**: `cd proxy && npx eslint --quiet "./src/**/*.{ts,tsx}"`
- **Server**: `cd server && go vet ./...`

### Tests

- **Client**: `cd client && yarn test` — runs Jest across all workspace packages (62/66 suites pass; 4 pre-existing failures due to circular imports, missing test fixture, and network access in jsdom)
- **Server**: `cd server && go test ./...`
