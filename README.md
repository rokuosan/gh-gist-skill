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
✓ Linked: ~/.claude/skills/japanese-tech-writing
```

The skill is added with `git submodule add https://gist.github.com/<id>.git .agents/skills/<name>` and staged together with `.gitmodules` — review and commit as usual. It is pinned to the current commit and travels with your repository; on another machine, `git submodule update --init` restores it. If the destination path already exists, `add` refuses and asks you to remove it first.

`add` currently supports project scope only (inside a git repository). Running it elsewhere is an error; user-scope clone mode is planned (M3).

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
3. Copies all gist files into `.agents/skills/<name>/` (no `.git`, not tracked)
4. Symlinks `~/.claude/skills/<name>` to the copy

The snapshot is plain files — it does not appear in any manifest and there is no update tracking. To update a skill, run `copy` again; it replaces the existing snapshot atomically.

### Flags (both commands)

| Flag | Description |
| --- | --- |
| `--path <dir>` | Destination directory (default: `.agents/skills`) |
| `--no-link` | Skip creating the `~/.claude/skills` symlink |

### Notes

- The gist must contain a `SKILL.md` with a spec-valid `name:` in its frontmatter. Spec-invalid skills are rejected with a pointer to the spec — that is for the skill author to fix.
- Gists are flat (no directories), so skills that need `scripts/` or `references/` cannot be published as gists in the first place. Files like `references__foo.md` are copied as-is, without expanding `__` into directories.

## Roadmap

- [x] M1: `copy` — snapshot install
- [x] M2: `add` — install as a git submodule inside a repository (pinned to a commit, distributable with the repo)
- [ ] M3: `update` / `remove` / `list`, clone mode for user scope
- [ ] M4: multi-agent support via a config file

## Development

```console
$ go test ./...
$ go build && gh extension install .
```
