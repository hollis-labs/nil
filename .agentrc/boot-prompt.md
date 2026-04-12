# Nil — Boot Prompt

## Active priorities

### Frontend biome cleanup (blocks clean commits)
- `npx biome check src/` reports ~403 errors + 189 warnings in `frontend/src/`
- Most are accessibility rules: `useKeyWithClickEvents`, `noStaticElementInteractions`, `useUniqueElementIds`
- Pre-existing, unrelated to any single feature — accumulated drift
- Currently bypassed via `--no-verify`; commits touching frontend files will fail the `frontend-lint` pre-commit hook until resolved
- **Action:** dedicated cleanup pass. Consider splitting by rule family and dispatching parallel agents. Some rules may warrant a biome config override if they conflict with the app's keyboard-first design (e.g., click handlers on styled `div`s are intentional).
