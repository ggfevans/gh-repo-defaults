<p align="center">
  <img src="static/hero.gif" alt="gh mint demo" width="720">
</p>

<h1 align="center">gh mint</h1>

<p align="center">
A GitHub CLI extension that creates repos and applies settings, labels, boilerplate, and branch protection from named YAML profiles.
</p>

<p align="center">
  <a href="#installation">Installation</a> &middot;
  <a href="#usage">Usage</a> &middot;
  <a href="#configuration">Configuration</a> &middot;
  <a href="#built-in-profiles">Profiles</a> &middot;
  <a href="#templates">Templates</a>
</p>

---

## Why

I got tired of clicking through the same GitHub UI checkboxes every time I made a new repo. Disable wiki, disable projects, delete branch on merge, squash only, replace the default labels, add a license, add CI, set up branch protection... you know the drill.

So now there's this.

## What it does

Define a profile once in YAML, then run one command to get a fully configured repository:

- **Repository settings** &mdash; wiki, projects, discussions, merge strategies
- **Labels** &mdash; clear GitHub's defaults, apply your own with colors and descriptions
- **Boilerplate files** &mdash; LICENSE, .gitignore, CONTRIBUTING.md, CI workflows, whatever you want
- **Branch protection** &mdash; required reviews, dismiss stale reviews, status checks

Works as both an interactive TUI and as scriptable CLI subcommands.

## Installation

Requires the [GitHub CLI](https://cli.github.com/) (`gh`).

```bash
gh extension install ggfevans/gh-mint
```

## Usage

### Interactive (TUI)

```bash
gh mint
```

Launches a terminal UI where you can select a mode, fill out a form, and watch it go.

### Create a new repo

```bash
gh mint create my-project --profile oss --public
gh mint create my-project --profile personal --private --description "A thing"
```

| Flag | Description |
|------|-------------|
| `-p, --profile` | Profile to apply (default: `default_profile` from config) |
| `--public` | Create a public repository |
| `--private` | Create a private repository |
| `--description` | Repository description |

### Apply a profile to an existing repo

```bash
gh mint apply ggfevans/some-repo --profile oss
```

Updates settings, syncs labels, and applies branch protection to a repo that already exists.

### List profiles

```bash
gh mint profiles list
```

```
NAME       DESCRIPTION                          LABELS  DEFAULT
personal   Personal project - minimal config    3       *
oss        Open source project defaults         6
action     GitHub Action defaults               3
```

### Show profile details

```bash
gh mint profiles show oss
```

## Configuration

Config lives at `~/.config/gh-mint/config.yaml`. If the file doesn't exist, built-in defaults are used.

```yaml
default_profile: oss
default_owner: ""  # leave empty for personal account

profiles:
  my-profile:
    description: "What this profile is for"

    settings:
      has_wiki: false
      has_projects: false
      has_discussions: false
      delete_branch_on_merge: true
      allow_squash_merge: true
      allow_merge_commit: false
      allow_rebase_merge: false
      squash_merge_commit_title: "PR_TITLE"
      squash_merge_commit_message: "PR_BODY"

    labels:
      clear_existing: true  # remove GitHub's default labels first
      items:
        - name: bug
          color: "d73a4a"
          description: "Something isn't working"
        - name: enhancement
          color: "a2eeef"

    boilerplate:
      license: MIT
      gitignore: Go
      files:
        - src: contributing.md    # template filename
          dest: CONTRIBUTING.md   # path in repo

    branch_protection:
      branch: main
      required_reviews: 1
      dismiss_stale_reviews: true
      require_status_checks: false
```

## Built-in profiles

Three profiles ship out of the box:

| Profile | Description | Labels | Merge strategy | Branch protection |
|---------|-------------|--------|----------------|-------------------|
| `personal` | Minimal personal project setup | bug, enhancement, chore | squash + merge commit | none |
| `oss` | Open source project defaults | bug, enhancement, documentation, good first issue, help wanted, wontfix | squash only | main (dismiss stale reviews) |
| `action` | GitHub Action project | bug, enhancement, breaking change | squash | none |

All three disable wiki and projects, enable delete-branch-on-merge, and include an MIT license.

## Templates

Boilerplate files are embedded in the binary. You can override any template by placing a file with the same name in `~/.config/gh-mint/templates/`.

**Included templates:**

| File | Used by |
|------|---------|
| `contributing.md` | `oss` profile |
| `ci.yml` | `oss` profile |
| `action.yml` | `action` profile |
| `action-ci.yml` | `action` profile |
| `action-release.yml` | `action` profile |

User-provided templates in the config directory take precedence over embedded ones.

## Disclaimer

This was made for my own use. It scratches my itch and encodes my opinions about how repos should be set up. Your mileage may vary.

That said, I'm more than happy to take issues and PRs. If something is broken, missing, or could be better, open an issue and let's talk about it.

## AI use disclosure

This project was built with assistance from Claude Code (Anthropic). The architecture, implementation, and documentation were developed collaboratively between a human and an AI. All code has been reviewed and tested by the maintainer.

## License

MIT
