# do

A developer workflow CLI that connects git worktrees, tmux, and Claude Code into a single fast loop. Each feature branch gets its own isolated worktree, its own tmux window, and an AI assistant ready to start — all from one command.

```
do branch feature/user-auth
```

---

## Why

Working with multiple features in parallel usually means constant `git stash`, context switching, and lost flow. `do` solves this by giving each branch its own directory (via git worktrees) and its own terminal window (via tmux), with Claude Code automatically opened and ready to go.

The full loop:

```
do spec "user authentication"   # write a structured spec
do start specs/user-auth.md     # spin up worktree + tmux + Claude
# ... implement ...
do commit                        # Claude writes the commit message
do pr                            # Claude writes the PR description
do review                        # Claude checks the spec criteria
do clean                         # remove merged worktrees
```

---

## Requirements

- Go 1.21+
- Git
- [tmux](https://github.com/tmux/tmux)
- [Claude Code](https://github.com/anthropics/claude-code) (`claude` in PATH)
- [GitHub CLI](https://cli.github.com/) (`gh`) — required for `do pr`

---

## Install

### From source

```bash
git clone https://github.com/erickmob/cli
cd cli
go build -o do .
sudo mv do /usr/local/bin/do
```

### Quick one-liner

```bash
go install github.com/erickmob/cli@latest
```

> The binary is named `do`. Make sure `$GOPATH/bin` (or `~/go/bin`) is in your `$PATH`.

---

## Commands

### `do branch`

Creates a git worktree for a branch and opens a new tmux window with Claude Code running inside it.

```bash
do branch feature/login
do branch fix/payment-bug --from develop
do branch experiment/new-ui --here
do branch feature/api --tmux session --session my-api
do branch feature/auth --path ~/worktrees/auth
```

**Branch resolution order:**
1. Branch exists locally → use it directly
2. Branch exists on remote → check out from `origin/<branch>`
3. Branch doesn't exist anywhere → create from `origin/main` (fetches first)

**Worktree path:** sibling directory of the current repo, named `<repo>-<branch>`.
Example: `/projects/my-app` + `feature/login` → `/projects/my-app-feature-login`

| Flag | Short | Description |
|---|---|---|
| `--from <branch>` | `-f` | Base new branch on this branch instead of `origin/main` |
| `--here` | `-H` | Base new branch on current HEAD |
| `--tmux <mode>` | `-t` | `window` (default) or `session` |
| `--path <path>` | `-p` | Custom worktree path |
| `--session <name>` | `-s` | tmux session name (only for `--tmux session`) |

---

### `do commit`

Commits staged changes. If no message is given, Claude generates one in [Conventional Commits](https://www.conventionalcommits.org/) format.

```bash
do commit                              # Claude generates the message
do commit "feat: add login endpoint"  # use your own message
```

Fails early with a clear error if there are no staged changes. Use `git add` first.

---

### `do pr`

Creates a GitHub Pull Request. Claude analyzes the diff and commit history to write the title and PR body. If a spec file exists for the current branch, it's included as context.

```bash
do pr
do pr --title "feat: user authentication"
do pr --base develop
```

The generated PR body includes:
- **Summary** — 2–3 bullet points describing what changed
- **Test Plan** — a checklist of things to verify

| Flag | Short | Description |
|---|---|---|
| `--base <branch>` | `-b` | Base branch (default: repo default) |
| `--title <title>` | `-t` | PR title (Claude generates if omitted) |

Requires `gh` to be installed and authenticated (`gh auth login`).

---

### `do spec`

Creates a structured Markdown spec file in the `specs/` directory of the current repo.

```bash
do spec "user authentication"
# → specs/user-authentication.md

do spec "Implement Payment Gateway"
# → specs/implement-payment-gateway.md
```

The generated file contains sections for context, objective, functional requirements, non-functional requirements, acceptance criteria, out of scope, and technical notes — ready to fill in.

---

### `do start`

Reads a spec file, derives a branch name from the title, and runs the full `do branch` flow with the spec content sent to Claude as initial context.

```bash
do start specs/user-authentication.md
```

The spec title is converted to a branch name:
`# Implement User Authentication` → `feature/implement-user-authentication`

Claude Code opens in the new tmux window with the full spec already loaded as context.

---

### `do review`

Compares the current implementation against the branch's spec file and asks Claude to evaluate every acceptance criterion.

```bash
do review
```

Output per criterion:
- `✅` — implemented
- `❌` — not implemented
- `⚠️` — partially implemented

Also surfaces unimplemented functional requirements and any gaps worth noting. Requires a spec file in `specs/` matching the current branch name.

---

### `do list`

Lists all active git worktrees with status information.

```bash
do list
```

Example output:

```
main                          ◀ current
  path:   /projects/my-app
  commit: a1b2c3d initial commit

feature-login  *
  path:   /projects/my-app-feature-login
  commit: 9f3e21a feat: add login form

feature-payments
  path:   /projects/my-app-feature-payments
  commit: 4c8a10b feat: stripe integration
```

- Current branch shown in **green**
- Worktrees with uncommitted changes marked with `*` in yellow

---

### `do clean`

Removes worktrees whose branches have already been merged into `origin/main`.

```bash
do clean
```

Shows a list of eligible worktrees and asks for confirmation before removing anything. For each confirmed removal: deletes the worktree directory and the local branch.

---

### `do sync`

Rebases the current branch on top of `origin/main`.

```bash
do sync
```

Fetches `origin/main` first, then runs `git rebase origin/main`. On conflict, aborts the rebase and prints clear instructions for resolving manually. Blocked on `main`/`master` branches.

---

## Typical Workflow

```bash
# 1. Write a spec for the feature
do spec "password reset flow"

# 2. Fill in specs/password-reset-flow.md, then start
do start specs/password-reset-flow.md
# → creates feature/password-reset-flow worktree
# → opens tmux window with Claude already reading the spec

# 3. Implement the feature in the new worktree window
#    (Claude Code is already open and has the spec as context)

# 4. Keep the branch up to date
do sync

# 5. Stage your changes, then commit
git add -p
do commit

# 6. Check if spec criteria are met
do review

# 7. Open a PR
do pr

# 8. After merge, clean up
do clean
```

---

## Project Structure

```
cli/
├── main.go
├── go.mod
└── cmd/
    ├── root.go      # Cobra setup, command registration
    ├── git.go       # Shared git utilities
    ├── branch.go    # do branch
    ├── commit.go    # do commit
    ├── pr.go        # do pr
    ├── spec.go      # do spec
    ├── start.go     # do start
    ├── list.go      # do list
    ├── clean.go     # do clean
    ├── sync.go      # do sync
    └── review.go    # do review
```

---

## Dependencies

| Package | Purpose |
|---|---|
| [spf13/cobra](https://github.com/spf13/cobra) | CLI framework |
| [fatih/color](https://github.com/fatih/color) | Colored terminal output |
| [briandowns/spinner](https://github.com/briandowns/spinner) | Progress spinner for long operations |

---

## License

MIT
