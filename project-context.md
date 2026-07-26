---
project_name: 'xbvr'
user_name: 'maicon'
date: '2026-07-25'
sections_completed: ['technology_stack', 'language_rules', 'framework_rules', 'testing_rules', 'quality_rules', 'workflow_rules', 'anti_patterns']
status: 'complete'
rule_count: 34
optimized_for_llm: true
---

# Project Context for AI Agents

_This file contains critical rules and patterns that AI agents must follow when implementing code in this project. Focus on unobvious details that agents might otherwise miss._

---

## Technology Stack & Versions

- **Backend Language:** Go 1.25.0
- **Database ORM:** GORM
- **Router:** gorilla/mux
- **Frontend Framework:** Vue 2.7.16 (Options API expected, matching version)
- **Frontend Routing & State:** Vue Router 3.6.5, Vuex 3.6.2
- **UI Components:** Buefy 0.9.29, Bulma Extensions 6.2.7
- **Build/Tooling:** Webpack via Vue CLI, Babel, ESLint (Standard config)

## Critical Implementation Rules

### Language-Specific Rules

**Go (Backend):**
- Strictly use Go 1.25 idioms and maintain type safety.
- Data structures must align with GORM declarative tags for database consistency.
- Rely on standard `gorilla/mux` patterns for all HTTP routing rather than the built-in `net/http` router, to maintain consistency.
- Keep a hard boundary between the Go backend services and the frontend artifacts (served via `fs.go` or similar embedding).

**JavaScript/Vue (Frontend):**
- All frontend assets, Vue components, and state management must remain isolated within the `ui/` directory.
- JavaScript must be compatible with Babel transpilation and pass standard ESLint configurations (`@vue/standard`).
- State mutation must happen explicitly via Vuex actions/mutations to prevent reactivity bugs in Vue 2.7.

### Framework-Specific Rules

**Vue 2.7 & Buefy (Frontend):**
- Strictly use the Vue 2 Options API to match existing patterns. Do not arbitrarily mix Composition API syntax (`setup()`) in existing files unless there is a clear directive to migrate them.
- Prefer Buefy components (`b-*`) and Bulma utility classes over writing custom CSS whenever possible to ensure visual consistency and maintain responsiveness.
- Adhere strictly to Vue I18n conventions: new user-facing strings must be added to localization JSON files and referenced via `$t()`, avoiding hardcoded English text in templates.

**Go Backend (GORM & Mux):**
- Use GORM for database migrations and queries rather than raw SQL. This ensures cross-database compatibility, specifically since SQLite is heavily utilized here.
- API Handlers should extract route variables exclusively via `mux.Vars(r)` when defining dynamic path parameters.
- Background tasks and cron jobs must use the established `robfig/cron/v3` patterns found in the project.

### Testing Rules

**Test Organization:**
- Tests for Go backend code should live alongside the files they test (e.g., `file_test.go`).
- Ginkgo/Gomega BDD-style testing is the standard for complex behaviors in the backend based on project dependencies. 

**Mock Usage & Databases:**
- Do not make external API calls during unit tests. Utilize standard Go mocking interfaces or `httptest` servers for HTTP clients.
- If relying on database queries in tests, ensure a temporary SQLite in-memory database (`file::memory:?cache=shared`) is used rather than touching persistent database files.

**Integration vs Unit Test Rules:**
- Keep backend Unit Tests fast by avoiding disk I/O and heavy DB operations unless strictly testing the GORM integration layers.
- Currently, there are no dedicated frontend testing frameworks (like Jest/Vitest) established in the `package.json`. Therefore, testing efforts should focus heavily on Go API stability and behavior.

### Code Quality & Style Rules

**Linting & Formatting:**
- Use standard `gofmt` (or `goimports`) for all Go files. Never deviate from standard Go formatting conventions.
- For JavaScript and Vue files in the `ui/` directory, strictly abide by the established ESLint configurations (`@vue/standard`). Do not disable ESLint rules inline without an explicit, well-documented justification.

**Code Organization:**
- Place reusable backend utility functions in the `pkg/` directory if they are broadly applicable and not coupled strictly to a single domain.
- Frontend components should remain cleanly structured within `ui/src/components/`, with distinct top-level views mapping directly to Vue Router paths.

**Documentation Requirements:**
- Every exported Go function, type, and interface should have an explanatory GoDoc comment.
- Avoid overly complex logic within Vue templates. Always extract complex template expressions into Vue computed properties or methods to keep the UI layer declarative and easy to read.

### Development Workflow Rules

**Build & Execution:**
- Use `npm run dev` to start the frontend and backend servers concurrently (`air` is used for Go live reload).
- Ensure any new dependencies added to the Go backend are tracked correctly in `go.mod` by running `go mod tidy` after modifying dependencies.

**Repository Patterns:**
- Ensure all CI/CD workflows (GitHub Actions) remain untouched unless there is a specific request to modify pipeline behavior.
- When generating builds for distribution, respect the `.goreleaser.yml` configurations for the Go binaries and do not attempt manual cross-compilation without checking existing scripts (like `build-local.sh`).

### Critical Don't-Miss Rules

**Anti-Patterns to Avoid:**
- **Mixing Vue APIs:** Do not introduce Vue 3 Composition API (`setup()`) or `<script setup>` syntax into the Vue 2.7 codebase, as this fragments the frontend design and confuses state management.
- **Raw SQL:** Do not bypass GORM with raw SQL queries unless absolutely necessary, to ensure SQLite compatibility is strictly preserved.
- **Asset Serving:** Do not hardcode static paths for frontend assets in Go; always rely on the established embedding mechanisms (`fs.go`).

**Security & Edge Cases:**
- Never expose sensitive environment variables or server paths directly in the frontend Vue templates or Vuex stores.
- Given the presence of media processing dependencies (`ffprobe`, `go-whisper`, `imaging`), ensure that any heavy media tasks are handled asynchronously in the Go backend (e.g., via goroutines) without blocking the main HTTP handlers.

---

## Usage Guidelines

**For AI Agents:**
- Read this file before implementing any code
- Follow ALL rules exactly as documented
- When in doubt, prefer the more restrictive option
- Update this file if new patterns emerge

**For Humans:**
- Keep this file lean and focused on agent needs
- Update when technology stack changes
- Review quarterly for outdated rules
- Remove rules that become obvious over time

Last Updated: 2026-07-25
