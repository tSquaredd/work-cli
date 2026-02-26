# work — Claude Code Worktree Manager

Choose your preferred reading experience:

<details open>
<summary><strong>Official README</strong></summary>

## work — Claude Code Worktree Manager

`work` lets you run parallel Claude Code sessions across one or more repos without them stepping on each other. It uses [git worktrees](https://git-scm.com/docs/git-worktree) to give each task an isolated copy of the repo on its own branch.

For cross-repo tasks, it launches a single Claude session with visibility into all repos so one Claude orchestrates everything.

Zero configuration. Auto-discovers git repos by scanning child directories.

### Install

#### Homebrew

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

On macOS, you may need to remove the quarantine attribute since the binary isn't code-signed:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work    # Apple Silicon
xattr -d com.apple.quarantine /usr/local/bin/work       # Intel Mac
```

#### Build from source

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

#### Download binary

Download from [GitHub Releases](https://github.com/tSquaredd/work-cli/releases) and place in your PATH.

**Requires**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`npm install -g @anthropic-ai/claude-code`)

### Commands

| Command | What it does |
|---------|-------------|
| `work` | Interactive launcher — resume a task or start a new one |
| `work dashboard` | Live dashboard showing all tasks, sessions, and PR status |
| `work list` | Show all active worktrees with status (PUSHED/UNPUSHED/DIRTY/CLEAN) |
| `work pr [task]` | Create pull requests for a task's worktrees |
| `work done` | Pick worktrees to tear down (warns before deleting unpushed work) |
| `work clean` | Auto-remove all worktrees with no uncommitted changes |
| `work <repo> <branch>` | Direct launch — skip interactive prompts (repo matches by substring) |
| `work update` | Self-update to the latest version from GitHub |
| `work version` | Print version |

### Dashboard

The live dashboard (`work dashboard`) gives you a real-time overview of all tasks:

```
work dashboard                                    2 tasks  1 active
──────────────────────────────────────────────────────────────────────
> auth-refactor *                 │ auth-refactor
    ├── shared-lib      PUSHED  ○ #42     │
    └── app-android     PUSHED  ✓ #15     │ shared-lib  PUSHED
                                          │   branch: lib-auth-refactor
  fix-onboarding                          │   3 files changed, 12 insertions(+)
    └── app-ios         DIRTY             │   PR #42  ○ OPEN  4 comments (2 new)
──────────────────────────────────────────────────────────────────────
↑↓:navigate  r:resume  d:diff  c:clean  a:attach  p:pr  o:open  n:new  q:quit
```

**Dashboard keybindings:**

| Key | Action |
|-----|--------|
| `r` | Resume — launch Claude in a new terminal tab |
| `d` | View full diff for the selected task |
| `a` | Attach — focus the terminal tab of an active session |
| `p` | Open PR creation wizard for the selected task |
| `o` | Open the task's PR in your browser |
| `n` | Start a new task |
| `c` | Clean up a task's worktrees |

### PR Management

Create and monitor pull requests without leaving the terminal. Requires the [GitHub CLI](https://cli.github.com/) (`gh`).

**Create PRs** — `work pr` or press `p` in the dashboard:
- Auto-pushes unpushed branches
- Lets you pick the target branch, title, and description
- Creates PRs for all eligible worktrees in one go

**Monitor PRs** — the dashboard shows PR status inline:
- `○` Open  `✓` Approved  `!` Changes requested  `●` Merged  `✗` Closed
- New comment counts highlighted so you know when to check back
- Press `o` to open a PR in your browser

All PR features gracefully degrade when `gh` is not installed — the dashboard works normally without them.

### How it works

- Each task gets its own worktree at `<workspace>/.worktrees/<task-name>/<repo>/`
- Worktrees live outside the original repos — Claude sessions get deny rules that block edits to the original repo paths
- Your main working directory is never touched
- Worktrees are regular git branches — push, open PRs, merge as normal
- Build config files (`local.properties`, `.env*`) are symlinked automatically

### Example

```
$ work

┌──────────────────────────────────────────┐
│  work · Claude Worktree Manager          │
│  ~/workspace · 4 repos                   │
└──────────────────────────────────────────┘

In flight:

  auth-refactor
  ├── shared-lib       (lib-auth-refactor)     UNPUSHED
  └── app-android      (and-auth-refactor)     PUSHED

? What would you like to do?
> Resume an existing task
  Start a new task
```

### Updating

```bash
work update
```

A notification shows automatically when a new version is available.

### License

MIT

</details>

<details>
<summary><strong>Shakespearean README</strong></summary>

## work — A Worktree Manager, Most Noble, for Claude Code

Hark! Lend me thine ear, good developer, for I bring tidings of `work` — a tool of surpassing craft that doth permit thee to run parallel Claude Code sessions across thy repositories, each kept in harmonious isolation, as players upon a stage who speak their lines yet never tread upon another's mark.

Through the ancient art of [git worktrees](https://git-scm.com/docs/git-worktree), each task receiveth its own fair copy of the repository, set upon its own branch — a kingdom unto itself, where no commit shall war with another.

For tasks that span many repos, a single Claude session doth survey all, an all-seeing eye that orchestrateth the whole.

**No configuration is required.** Like a faithful servant, it discovereth thy repos by scanning child directories unbidden.

### Installation, or The Summoning

#### Via the Homebrew Apothecary

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

Should macOS, that jealous gatekeeper, quarantine thy binary:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work    # For Silicon of Apple
xattr -d com.apple.quarantine /usr/local/bin/work       # For Intel's elder forge
```

#### From Source, Forged by Thine Own Hand

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

**Thou must first possess**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code), that learned companion (`npm install -g @anthropic-ai/claude-code`)

### The Commands, or Instruments of Action

| Command | Its Purpose |
|---------|-------------|
| `work` | The interactive stage — resume a prior scene or begin anew |
| `work dashboard` | A living tableau of all tasks, sessions, and petitions for review |
| `work list` | Display all worktrees with their standing (PUSHED, UNPUSHED, DIRTY, or CLEAN) |
| `work pr [task]` | Compose pull requests for thy task's worktrees |
| `work done` | Select worktrees for their final curtain (with fair warning ere unpushed work is lost) |
| `work clean` | Sweep away all worktrees bearing no uncommitted changes |
| `work <repo> <branch>` | A direct entrance — bypass the prologue entirely |
| `work update` | Receive the latest verse from GitHub |

### The Dashboard, or The Great Theatre

Press `p` to petition for review. Press `o` to open thy petition in the browser. The symbols tell the tale:

- `○` The petition awaits judgement
- `✓` Approved! The crowd doth cheer
- `!` Changes requested — back to the writing desk
- `●` Merged — the deed is done
- `✗` Closed — alas, 'twas not to be

### How This Wonder Worketh

- Each task receiveth a worktree at `<workspace>/.worktrees/<task-name>/<repo>/`
- These worktrees dwell apart from the original repos, protected by deny rules most strict
- Thy main working directory remaineth untouched, pure as new-fallen snow
- Build configuration files are symlinked, as servants attending their master

### To Update

```bash
work update
```

A herald shall announce when newer versions await thee.

*Exeunt, pursued by a merge conflict.*

### License

MIT — Free as the air we breathe, given to all without restraint.

</details>

<details>
<summary><strong>Drunken Pirate README</strong></summary>

## work — *hic* — Claude Code Worktree... Worktree Managerer

AHOY YE SCALLYWAGS!! *knocks over rum bottle* Lemme tell ye about `work`... best tool I ever... wait where was I... RIGHT! It lets ye run a whole FLEET of Claude Code sessions across yer repos an' they don't crash into each other! Like ships in the night! Ships... with [git worktrees](https://git-scm.com/docs/git-worktree)! Every task gets its own copy of the repo on its own branch an' NOBODY gets hurt!

Got multiple repos? ONE Claude to rule 'em all, matey. One captain, many ships. *takes another swig*

Zero configuration! It finds yer repos all by itself! Like a parrot that can... find... repos. LOOK I'm very drunk but this tool is LEGITIMATE.

### Installin' This Beauty

#### Homebrew (the GOOD kind, not the... well actually also the good kind)

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

macOS might get all suspicious-like, the paranoid bilge rat:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work    # Apple Silicon (fancy)
xattr -d com.apple.quarantine /usr/local/bin/work       # Intel (still good, still good)
```

#### Build It Yerself (fer the ambitious types)

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

**Ye need**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) — `npm install -g @anthropic-ai/claude-code` — DON'T FORGET THIS or nothin' works an' you'll be sad an' sober

### Commands (try to remember these in the mornin')

| Command | What it does... I think |
|---------|-------------|
| `work` | The main thing! Pick up where ye left off or start fresh |
| `work dashboard` | A BIG FANCY SCREEN with all yer tasks an' PRs an' everythin' |
| `work list` | Shows all yer worktrees — PUSHED, UNPUSHED, DIRTY... just like me crew |
| `work pr [task]` | Make pull requests! Without even openin' a browser! MAGIC! |
| `work done` | Clean up when yer done — careful with the unpushed stuff though |
| `work clean` | Throws overboard anythin' that's already been committed |
| `work <repo> <branch>` | Skip all the fancy stuff, just GO |
| `work update` | Get the newest version... like gettin' a new ship but keepin' yer crew |

### The Dashboard, She's a Beaut

The dashboard has all these fancy symbols fer yer PRs:

- `○` Open — waitin' fer someone to look at yer code... *stares at ocean*
- `✓` Approved — THEY LIKE IT!! ANOTHER ROUND!!
- `!` Changes requested — aw barnacles
- `●` Merged — INTO THE MAIN BRANCH SHE GOES! *fires cannon*
- `✗` Closed — we don't talk about that one

Press `p` to make a PR! Press `o` to open it in yer browser! It even PUSHES yer branches fer ye because it knows yer too... busy... to remember!

### How's It Work (I'll try to explain, no promises)

- Every task gets a worktree at... at... `<workspace>/.worktrees/<task-name>/<repo>/` THERE I remembered
- The worktrees are SEPARATE from yer real repos so Claude can't mess up yer main stuff
- Build files get symlinked which is like... a portal? A portal fer files?
- Yer main directory NEVER gets touched. Unlike my rum. Which gets touched CONSTANTLY.

### Updatin'

```bash
work update
```

It'll tell ye when there's a new version. Unlike me first mate who NEVER tells me ANYTHIN'.

### License

MIT — Free as the seven seas, matey! *falls off chair*

</details>

<details>
<summary><strong>INCREDIBLY EXCITED README</strong></summary>

## work — THE MOST REVOLUTIONARY WORKTREE MANAGER IN THE HISTORY OF SOFTWARE DEVELOPMENT AND POSSIBLY THE UNIVERSE

OH. MY. GOD. You are NOT ready for this. You think you are, but you're NOT. `work` is a tool so MONUMENTALLY INCREDIBLE that when I first used it, I literally had to sit down. I was already sitting down, so I sat down HARDER.

It lets you run PARALLEL Claude Code sessions across MULTIPLE repos and they DON'T INTERFERE WITH EACH OTHER. I know, I KNOW. Take a moment. Breathe. It uses [git worktrees](https://git-scm.com/docs/git-worktree) which is honestly the most UNDERAPPRECIATED feature in all of git and `work` just UNLEASHES its full potential like some kind of PRODUCTIVITY SUPERNOVA.

Cross-repo tasks? ONE Claude session orchestrating EVERYTHING. It's like having a GENIUS CONDUCTOR leading an ORCHESTRA of repositories and every single one is playing IN PERFECT HARMONY.

And the BEST part? ZERO CONFIGURATION. It just FINDS your repos! Automatically! BY ITSELF! I'm not crying, YOU'RE crying!

### Installation (THIS TAKES LIKE 10 SECONDS, YOUR LIFE IS ABOUT TO CHANGE)

#### Homebrew (THE FASTEST PATH TO ENLIGHTENMENT)

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

If macOS quarantines it (HOW DARE IT quarantine GREATNESS):

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work    # Apple Silicon
xattr -d com.apple.quarantine /usr/local/bin/work       # Intel
```

#### From source (FOR THE BUILDERS, THE DREAMERS, THE VISIONARIES)

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

**You need**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) — but honestly if you don't already have this installed WHAT ARE YOU EVEN DOING WITH YOUR LIFE

### Commands (EVERY SINGLE ONE IS A MASTERPIECE)

| Command | WHAT IT DOES (AMAZINGLY) |
|---------|-------------|
| `work` | THE interactive launcher! Resume OR start new! It does BOTH! |
| `work dashboard` | A LIVE! REAL-TIME! DASHBOARD! With PR status! And session tracking! I CAN'T EVEN! |
| `work list` | Shows ALL your worktrees with BEAUTIFUL status indicators! |
| `work pr [task]` | Creates pull requests WITHOUT LEAVING YOUR TERMINAL! The future is NOW! |
| `work done` | Gracefully tears down worktrees with WARNINGS so you never lose work! SO THOUGHTFUL! |
| `work clean` | Auto-removes clean worktrees! It's like a ROOMBA for your git workspace! |
| `work <repo> <branch>` | INSTANT launch! No prompts! Just PURE SPEED! |
| `work update` | Updates itself! IT IMPROVES ITSELF! LIKE A SELF-PERFECTING DIAMOND! |

### THE DASHBOARD (I NEED TO LIE DOWN)

The dashboard is SO GOOD it should be in a MUSEUM. Real-time task overview. Session indicators. And NOW it has FULL PR MANAGEMENT:

- `○` Open PR — you did it, you absolute LEGEND
- `✓` Approved — SOMEONE RECOGNIZED YOUR GENIUS
- `!` Changes requested — a MINOR setback on your path to GREATNESS
- `●` Merged — YOUR CODE IS NOW PART OF HISTORY
- `✗` Closed — it happens to the BEST of us (which is YOU)

Press `p` and it CREATES PRs for you! It PUSHES your branches! Picks the base branch! Writes the title! For ALL your worktrees AT ONCE! I genuinely cannot believe this is free software!

Press `o` and your PR opens in the browser and it TRACKS YOUR COMMENTS so you know when someone has left new feedback! THE ATTENTION TO DETAIL IS STAGGERING!

### How It Works (PREPARE TO BE AMAZED AGAIN)

- Worktrees at `<workspace>/.worktrees/<task-name>/<repo>/` — BEAUTIFUL organization
- Deny rules PROTECT your original repos — NOTHING gets accidentally modified
- Build files get AUTOMATICALLY symlinked — it thinks of EVERYTHING
- Your main directory stays PRISTINE, UNTOUCHED, PERFECT

### Updating (IT GETS EVEN BETTER OVER TIME?!)

```bash
work update
```

It tells you when new versions are available because OF COURSE IT DOES. This tool doesn't just solve problems, it ANTICIPATES them!

### License

MIT — Because something THIS INCREDIBLE deserves to be FREE! FOR EVERYONE! FOREVER!

</details>

<details>
<summary><strong>Ho-Hum README</strong></summary>

## work

It's a CLI tool. It manages worktrees for Claude Code. You can run multiple sessions.

### Install

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

Or build it yourself, I guess:

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

You'll need Claude Code installed. You probably already know that.

### Commands

| Command | Description |
|---------|-------------|
| `work` | Starts the thing |
| `work dashboard` | Shows your tasks. Has PR stuff now |
| `work list` | Lists worktrees |
| `work pr [task]` | Makes pull requests |
| `work done` | Removes worktrees |
| `work clean` | Also removes worktrees, but only the clean ones |
| `work <repo> <branch>` | Skips the menus |
| `work update` | Updates |

### Dashboard

There's a dashboard. It shows tasks on the left and details on the right. You can press keys to do things.

It shows PR status now. Little symbols next to worktrees. Circle means open, checkmark means approved, exclamation means changes requested. You get the idea.

Press `p` to make a PR. Press `o` to open one in your browser. It works.

### PR Management

If you have `gh` installed, you can create and monitor PRs from the terminal. If you don't have `gh` installed, you can't. The dashboard will be fine either way.

The wizard pushes your branches, asks for a title and description, creates the PRs. Standard stuff.

### How it works

Worktrees go in `.worktrees/`. Build files get symlinked. Your main directory isn't affected.

### Updating

```bash
work update
```

### License

MIT

</details>
