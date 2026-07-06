# gh-gist-skill

A [gh](https://cli.github.com/) extension that installs [Agent Skills](https://agentskills.io/specification) published as GitHub Gists.

People increasingly publish Agent Skills as gists, but installing one by hand is tedious: open the gist, copy the URL, place the files, and make sure the directory name matches the `name:` in `SKILL.md` frontmatter (required by the spec). This extension automates that. Because it runs through `gh`, it reuses your existing login, so secret gists work too.

## Installation

```console
$ gh extension install rokuosan/gh-gist-skill
```

## Usage

### `add` — install as a git submodule

Run inside a git repository (your dotfiles, a project, …):

```console
$ gh gist-skill add https://gist.github.com/k16shikano/fd287c3133457c4fd8f5601d34aa817d
✓ Resolved gist: fd287c3133457c4fd8f5601d34aa817d
✓ Detected skill name from SKILL.md: japanese-tech-writing
✓ Added submodule: .agents/skills/japanese-tech-writing
✓ Linked: .claude/skills/japanese-tech-writing
```

The skill is added with `git submodule add https://gist.github.com/<id>.git .agents/skills/<name>` at the repository root and staged together with `.gitmodules` — review and commit as usual. It is pinned to the current commit and travels with your repository; on another machine, `git submodule update --init` restores it. If the destination path already exists, `add` refuses and asks you to remove it first.

Project-scope installs stay inside the repository: the only link is a committable relative symlink `./.claude/skills/<name>` → `../../.agents/skills/<name>`, and nothing is written to your home directory.

Outside a git repository (or with `--scope user`), `add` instead clones the gist into `$XDG_DATA_HOME/gh-gist-skill/skills/<name>` (default `~/.local/share/...`) and links it into `~/.agents/skills` and `~/.claude/skills`. Each skill is an independent clone; no parent repository is created.

### `update` / `remove` / `list` — manage installed skills

```console
$ gh gist-skill list
NAME                   SCOPE    COMMIT   STATUS      PATH
japanese-tech-writing  project  5ed08e4  up to date  .agents/skills/japanese-tech-writing

$ gh gist-skill update                  # or: update <name>
✓ japanese-tech-writing (project): 5ed08e4 -> 9c2f1ab
Note: updated submodules are not committed; review and commit to pin the new versions.

$ gh gist-skill remove japanese-tech-writing
✓ Removed submodule: .agents/skills/japanese-tech-writing (commit the change to finish)
```

These commands cover both scopes: project-scope submodules under `.agents/skills/` in the current repository, and user-scope clones in the data directory. `update` uses `git submodule update --remote` for submodules (the new pin needs a commit) and `git pull --ff-only` for clones. `remove` also deletes the symlinks the tool created and cleans up `.git/modules`. `list --no-status` skips the network check for upstream updates.

Skills installed with `copy` do not appear here — a copy is a plain snapshot with no tracking, by design.

### `copy` — snapshot install

```console
$ gh gist-skill copy https://gist.github.com/k16shikano/fd287c3133457c4fd8f5601d34aa817d
✓ Resolved gist: fd287c3133457c4fd8f5601d34aa817d
✓ Detected skill name from SKILL.md: japanese-tech-writing
✓ Copied snapshot: .agents/skills/japanese-tech-writing
✓ Linked: ~/.claude/skills/japanese-tech-writing
```

`copy` takes a fire-and-forget snapshot of the gist:

1. Resolves the gist from a URL (with or without `#file-` fragment) or a bare gist ID
2. Reads the `name:` from `SKILL.md` frontmatter and validates it against the Agent Skills spec
3. Copies all gist files (no `.git`, not tracked) following the same scope rules as `add`:
   - inside a git repository → `<root>/.agents/skills/<name>/` with a relative `./.claude/skills/<name>` link
   - outside → `~/.agents/skills/<name>/` with a `~/.claude/skills/<name>` link
   - with `--path <dir>` → `<dir>/<name>/`, no links

The snapshot is plain files — it does not appear in any manifest and there is no update tracking. To update a skill, run `copy` again; it replaces the existing snapshot atomically.

### Flags

| Command | Flag | Description |
| --- | --- | --- |
| `add` | `--scope <s>` | `auto` (default), `project` (submodule), or `user` (clone) |
| `add`, `copy` | `--no-link` | Skip creating symlinks into agent skill directories |
| `copy` | `--path <dir>` | Custom destination directory (skips symlinks) |
| `remove` | `--scope <s>` | Disambiguate when a skill exists in both scopes |
| `list` | `--no-status` | Skip the network check for upstream updates |

`add` always installs to `.agents/skills/<name>` (project) or the data directory (user) so that `list` / `update` / `remove` can find what it installed; only the untracked `copy` takes a free-form `--path`.

### Notes

- The gist must contain a `SKILL.md` with a spec-valid `name:` in its frontmatter. Spec-invalid skills are rejected with a pointer to the spec — that is for the skill author to fix.
- Gists are flat (no directories), so skills that need `scripts/` or `references/` cannot be published as gists in the first place. Files like `references__foo.md` are copied as-is, without expanding `__` into directories.

## Roadmap

- [x] M1: `copy` — snapshot install
- [x] M2: `add` — install as a git submodule inside a repository (pinned to a commit, distributable with the repo)
- [x] M3: `update` / `remove` / `list`, clone mode for user scope
- [ ] M4: multi-agent support via a config file

## Development

```console
$ go test ./...
$ go build && gh extension install .
```
