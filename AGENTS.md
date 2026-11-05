# rexkit

A public Go library: the design system for terminal apps that stay open in Rex tabs and are used from a phone. Built on Charm v2. The README lists the packages and the design rules; follow those rules when you change or add anything.

## Rules

- **Public repo.** No internal hostnames, IPs, accounts or tokens. Not in code, tests, docs or commit messages.
- **Generic code only.** No package may import an app. An app-specific need becomes an option or a hook, not a special case.
- **Imports:** use Charm v2 (`charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`) and `github.com/charmbracelet/x/ansi`. Never the v1 `github.com/charmbracelet/...` paths.
- **Tests:** `mise run check` before every commit (gofmt, vet, tests). Tests never reach the network.
- **Output changes are breaking for apps.** Apps compare screen snapshots byte for byte. A change to what a component draws is a `feat` with a note in the PR, or `feat!` when callers must change.
- **Dependency versions:** keep `chroma` and `goldmark` in step with the apps. A version bump can change highlighted output.

## Commits and releases

- **Conventional Commits** for every commit and PR title (`feat:`, `fix:`, `docs:`, `chore:`, `ci:`, `test:`, `refactor:`). PRs are squash-merged, so the PR title is the changelog line.
- Name the Linear issue in the body (`Part of NIN-12` / `Fixes NIN-12`). The Linear project is rexkit.
- **Releases:** release-please keeps a release PR open, and merging it tags `vX.Y.Z`. Pre-1.0, `feat` bumps the minor version.
- **Actions** are pinned to commit SHAs with the version as a comment. Renovate keeps them current.
