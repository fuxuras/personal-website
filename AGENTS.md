# Repository Guidelines

## Project Structure & Module Organization
- `src/pages/` contains Astro routes (e.g., `src/pages/index.astro`, `src/pages/blog/[slug].astro`).
- `src/components/` holds reusable UI components (Astro and React).
- `src/content/` stores content collections for blog, projects, and commonplace entries.
- `src/styles/` is global styling; Tailwind is configured via `@tailwindcss/vite`.
- `public/` holds static assets served as-is (favicons, images).
- `backend/` is a small Go service for view counts, backed by SQLite (`backend/data/views.db`).

## Build, Test, and Development Commands
- `npm install` installs frontend dependencies.
- `npm run dev` starts the Astro dev server at `http://localhost:4321`.
- `npm run build` builds the production site to `dist/`.
- `npm run preview` serves the production build locally.
- `docker compose up --build` runs the view-counter service (ports `8080:8080`) for dev.
- `go run ./backend/main.go` runs the Go service without Docker.

## Coding Style & Naming Conventions
- Indentation: 2 spaces for Astro/JS/TS/CSS, tabs per `gofmt` for Go.
- Component files are PascalCase (e.g., `ProjectCard.astro`, `ViewCounter.tsx`).
- Content slugs use kebab-case (e.g., `src/content/blog/java-tips.md`).
- Keep Tailwind utility class ordering consistent within a file.

## Testing Guidelines
- No automated test suite is currently configured.
- If you add tests, document the framework and add a script in `package.json` (e.g., `npm run test`).

## Commit & Pull Request Guidelines
- Commits follow a lightweight Conventional Commit style: `feat: ...` (see recent history).
- PRs should include a clear description, relevant screenshots for UI changes, and mention any content updates.
- If you change the backend API, note any required frontend updates.

## Security & Configuration Tips
- The Go service uses SQLite at `backend/data/views.db`. Configure `DB_PATH`, `PORT`, `FEED_USERNAME`, and `FEED_PASSWORD` in `.env` if needed.
- CORS is currently permissive; lock it down if deploying beyond local use.
