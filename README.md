# work — Claude Code Worktree Manager

Choose your preferred reading experience:

<details>
<summary><strong>Dan Readme</strong> — Just the facts, no nonsense</summary>

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

**Requires**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`npm install -g @anthropic-ai/claude-code`)

**Optional**: [GitHub CLI](https://cli.github.com/) for PR management (`brew install gh && gh auth login`)

### Commands

| Command | What it does |
|---------|-------------|
| `work` | Launch the dashboard — live overview of all tasks, sessions, and PR status |
| `work update` | Self-update to the latest version from GitHub |
| `work version` | Print version |

### Dashboard

The dashboard launches by default when you run `work`:

```
work                                              2 tasks  1 active
──────────────────────────────────────────────────────────────────────
> auth-refactor *                 │ auth-refactor
    ├── shared-lib      PUSHED    │
    └── app-android     PUSHED    │ shared-lib  PUSHED
                                  │   branch: lib-auth-refactor
  fix-onboarding                  │   3 files changed, 12 insertions(+)
    └── app-ios         DIRTY     │
──────────────────────────────────────────────────────────────────────
↑↓:navigate  r:resume  d:diff  c:clean  t:test  a:attach  p:pr  n:new  q:quit
```

**Dashboard keybindings:**

| Key | Action |
|-----|--------|
| `n` | New task — open the task creation wizard |
| `r` | Resume — launch Claude in a new terminal tab for the selected task |
| `a` | Attach — focus the terminal tab of an already-active session |
| `t` | Test — switch local repos to the task's branches for local testing |
| `c` | Clean — remove worktrees and delete branches when a task is done |
| `d` | View full diff for the selected task |
| `p` | Open PR creation wizard |
| `o` | Open the task's PR in your browser |
| `m` | Open the in-terminal review comment viewer |

### Task Lifecycle

**1. Start a task — press `n`**

An overlay wizard appears (no full-screen takeover). Pick the repos you're working in, name the task, and set a branch name per repo. `work` creates isolated worktrees at `<workspace>/.worktrees/<task>/<repo>/` and launches Claude with access to all of them.

**2. Resume at any time — press `r`**

Reopens the task in a new terminal tab with a fresh Claude session. If the session is still active, `r` focuses the existing tab instead. You can also press `r` on a PR row in the "Your PRs" section to resume work on that PR — `work` finds the matching task automatically, or walks you through creating one pre-filled from the PR.

**3. Test locally — press `t`**

When your code is pushed and you want to test it in your local repos:

- Pre-flight check: all worktrees must be in `PUSHED` state, all local repos must be clean
- Per repo: fetch → remove worktree (branch kept) → checkout task branch → pull
- Leaves your local repos on the task branches, ready to run

**4. Clean up — press `c`**

When the task is done: removes worktrees and force-deletes the branches.

| | Test (`t`) | Clean (`c`) |
|---|---|---|
| Worktree | Removed | Removed |
| Branch | Kept | Deleted |
| Local repo | Switched to task branch | Unchanged |
| Use when | Testing locally | Task is fully done |

### PR Management

Requires the [GitHub CLI](https://cli.github.com/). Press `p` to create PRs, `o` to open in browser, `m` to read and reply to review comments. PR status (`○` open, `✓` approved, `!` changes requested, `●` merged) shows inline on each worktree. All PR features gracefully degrade when `gh` is not installed.

### How it works

- Each task gets its own worktree at `<workspace>/.worktrees/<task-name>/<repo>/`
- Worktrees live outside the original repos — Claude sessions get deny rules that block edits to the original repo paths
- Your main working directory is never touched
- Worktrees are regular git branches — push, open PRs, merge as normal
- Build config files (`local.properties`, `.env*`) are symlinked automatically

### Updating

```bash
work update
```

A notification shows automatically when a new version is available.

</details>

<details>
<summary><strong>Bill Quillsworth</strong> — Forsooth, a README most noble</summary>

## work — A Worktree Manager, Most Noble, for Claude Code

### Act I — The Prologue

*Enter DEVELOPER, weary, beset on all sides by merge conflicts*

Hark! Lend me thine ear, good developer, for I bring tidings of `work` — a tool of surpassing craft that doth permit thee to run parallel Claude Code sessions across thy repositories, each kept in harmonious isolation, as players upon a stage who speak their lines yet never tread upon another's mark.

Through the ancient art of [git worktrees](https://git-scm.com/docs/git-worktree), each task receiveth its own fair copy of the repository, set upon its own branch — a kingdom unto itself, where no commit shall war with another.

For tasks that span many repos, a single Claude session doth survey all, an all-seeing eye that orchestrateth the whole.

**No configuration is required.** Like a faithful servant, it discovereth thy repos by scanning child directories unbidden.

### Act II — The Summoning

*DEVELOPER approaches the Homebrew Apothecary*

#### Scene 1: Via Homebrew

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

Should macOS, that jealous gatekeeper, quarantine thy binary — O treachery most foul! — speak thus to break the seal:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work    # For Silicon of Apple
xattr -d com.apple.quarantine /usr/local/bin/work       # For Intel's elder forge
```

**Thou must first possess**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code), that learned companion, without whom all is silence upon the stage (`npm install -g @anthropic-ai/claude-code`)

**And shouldst thou desire petitions for review**: The [GitHub CLI](https://cli.github.com/), herald of pull requests, must be summoned and sworn in:
```bash
brew install gh && gh auth login
```

### Act III — The Instruments of Action

*DEVELOPER takes the throne. A flourish of trumpets.*

| Command | Its Purpose |
|---------|-------------|
| `work` | The living tableau — a grand stage revealing all tasks, sessions, and their petitions |
| `work update` | Receive the latest verse from GitHub, that distant oracle |
| `work version` | Declare thy version unto the world |

### Act IV — The Four Movements of a Task

*The curtain rises on a two-panel stage. The DEVELOPER takes position.*

**The First Movement — Creation** (`n`)

Press `n` and an overlay doth appear upon the stage — a wizard of forms most elegant. Choose thy repositories, name thy task, and decree upon each repo its branch. `work` then retreats to the wings to create thy worktrees, and returneth to present thee with a freshly-launched Claude session. All in silence, all behind the curtain, thy main directory untouched throughout.

**The Second Movement — Resumption** (`r`)

Shouldst thou step away from a task and wish to return — mayhaps the hour grew late, mayhaps another matter called thee hence — press `r` upon the task and `work` summoneth Claude anew in a fresh terminal tab, restoring all context. If the session yet liveth, `r` doth instead focus that existing tab, bringing thee to where thy work already waiteth. Even upon the PR scroll itself canst thou press `r` — `work` shall find thy matching task as a steward finds his lord in a crowded hall, or create one anew pre-filled from the petition's branch.

**The Third Movement — Testing** (`t`)

When thy code hath been pushed to the remote and thou desirest to test it in thy local repositories — not in the worktree, but in the very house whence it came — press `t`. `work` first inspects thy worktrees for signs of unfinished business; should all be in `PUSHED` state and thy local repos free of uncommitted burden, it proceedeth. It fetcheth the latest from the remote, removeth each worktree whilst keeping its branch alive, checketh out the branch in thy local repo, and pulleth the latest changes. Thy local directories stand ready for testing, as a theatre prepared for its audience.

*Note well: should any worktree harbour unpushed commits, or any local repo carry uncommitted changes, `t` shall refuse its service until the matter is resolved.*

**The Fourth Movement — The Conclusion** (`c`)

When the task is complete and all hath been merged, press `c` to close the chapter. The worktrees are removed and the branches deleted — a clean slate, ready for the next act.

### Act V — The Mechanics of This Wonder

*Aside, to the audience*

- Each task receiveth a worktree at `<workspace>/.worktrees/<task-name>/<repo>/`
- These worktrees dwell apart from the original repos, protected by deny rules most strict — as castle walls that guard the kingdom within
- Thy main working directory remaineth untouched, pure as new-fallen snow
- Build configuration files are symlinked, as servants attending their master through secret passages

**Of pull requests:** Should the [GitHub CLI](https://cli.github.com/) be installed, press `p` to open a petition, `o` to view it in the browser, `m` to read and answer thy reviewers' counsel.

### Epilogue

```bash
work update
```

A herald shall announce when newer versions await thee, appearing unbidden upon thy terminal as a ghost upon the battlements.

*Exeunt DEVELOPER, to test their code in peace.*

*Fin.*

</details>

<details>
<summary><strong>Tim Apple</strong> — Good morning. We think you're going to love this.</summary>

## work

### Good morning.

We are so excited to be here today.

You know, at work-cli, we've always believed that developers deserve tools that are not just powerful — but *magical*. Tools that get out of your way and let you do what you do best.

And today... we think we have something truly extraordinary to share with you.

*[pause for effect]*

This... is `work`.

*[slide: the word "work" in San Francisco font on a white background]*

### The best way to run Claude Code sessions. Ever.

`work` uses [git worktrees](https://git-scm.com/docs/git-worktree) to give every single task its own completely isolated copy of your repository. Its own branch. Its own space. No conflicts. No interference. Just... focus.

And for cross-repo tasks, one Claude session sees everything. One intelligent session, orchestrating across all your repositories seamlessly.

And here's the part I'm really excited about.

*Zero configuration.*

`work` discovers your repos automatically. You just open your terminal, and it's ready. We think that's really great.

### Installation

Now, getting started could not be simpler. And I mean that.

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

That's it. That's the install.

*[audience applause]*

On macOS, you may need to do one more thing — and the team has worked really hard to make this as painless as possible:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work    # Apple Silicon
xattr -d com.apple.quarantine /usr/local/bin/work       # Intel Mac
```

`work` pairs beautifully with [Claude Code](https://docs.anthropic.com/en/docs/claude-code). You're going to need that installed too. (`npm install -g @anthropic-ai/claude-code`)

And for PR creation and review — you'll want the [GitHub CLI](https://cli.github.com/):

```bash
brew install gh && gh auth login
```

Two commands. Seamless.

### Commands

| Command | What it does |
|---------|-------------|
| `work` | The dashboard. Live, real-time. Tasks, sessions, PR status — all in one place. Beautifully simple. |
| `work update` | Seamless self-updates. The latest and greatest, always within reach. |
| `work version` | See which version you're running. Clean. Minimal. |

### The Task Lifecycle

Now, I want to spend a moment walking you through the task lifecycle, because the team has done some *incredible* work here. Let me show you what working with `work` actually feels like.

**Starting a task — press `n`**

An elegant wizard overlay appears — right in the dashboard, no full-screen interruption. You pick your repos, name your task, set a branch per repo. Behind the scenes, `work` creates isolated worktrees and launches Claude. In one keystroke, you have a completely isolated workspace ready to go. We think that's beautiful.

**Resuming a task — press `r`**

Step away. Come back tomorrow. Press `r` and your Claude session opens in a new tab, exactly where you need to be — the right directory, the right repos. If the session is still running, `r` focuses it instead of opening a new one. And — and this is the part I love — if you're looking at a PR and press `r`, `work` finds the matching task automatically. Or it walks you through creating one, with the branch and repo already filled in. It just knows what you want to do. That's the kind of intelligence that changes how you work.

*[pause]*

**Testing locally — press `t`**

Here's the one that surprised even us. You've got a task. Your code is pushed. You want to run it locally — not in the worktree, but in your actual repo. Press `t`.

`work` checks that everything is ready. All worktrees pushed? Local repos clean? Good. Then it fetches, removes the worktrees, checks out the task branch in each local repo, and pulls. Your entire local environment is on the task branch in seconds. Ready for your test suite. Ready for your emulator. Ready for whatever you need.

We really think this is going to change how you approach testing.

**Cleaning up — press `c`**

When you're done — PR merged, task complete — press `c`. Worktrees removed, branches deleted. The workspace is clean. You're ready for the next thing.

### Under the Hood

- Worktrees are created at `<workspace>/.worktrees/<task-name>/<repo>/` — completely separate from your original repos
- Intelligent deny rules ensure Claude only edits worktree copies. Your main directory is never touched.
- Build files — `local.properties`, `.env` files — are automatically symlinked. It just works.

PR features — create, monitor, review comments — are available when the [GitHub CLI](https://cli.github.com/) is installed. Everything degrades gracefully when it's not.

### One More Thing

```bash
work update
```

`work` tells you when a new version is available. Always improving.

Because we believe the best developer tools aren't just something you use once. They're something that grows with you.

Thank you. We think you're going to love it.

</details>

<details>
<summary><strong>Ace Hyzer</strong> — Grip it, rip it, ship it</summary>

## work

You know that feeling. First throw of the day. Morning fog still hanging in the trees. Dew on the teepad. You've got a clean line through the gap — maybe 240 feet, slight dogleg left — and you know exactly which disc to reach for. You set your feet, take a breath, pull through clean, and watch it flip up, ride the line, and park under the basket. Nothing but grass and chains ahead.

That's what coding is supposed to feel like.

Instead, you've got three repos with branches tangled like fishing line after a water carry, merge conflicts breeding faster than mushrooms on a Pacific Northwest fairway, and you're spending more time switching context than actually writing code. You're grip-locked. Your mental game is shot. You're throwing nose-up hyzers into the first available tree on every hole.

`work` is the round that fixes your form.

It uses [git worktrees](https://git-scm.com/docs/git-worktree) to give every task its own isolated branch, its own clean copy of the repo — like stepping up to a fresh teepad on an empty course. No groups ahead. No one breathing down your neck. Just you, the gap, and the flight line you've been visualizing since the parking lot.

Multiple repos? One Claude session reads them all. It's the buddy who's played every course within 100 miles and still remembers the wind patterns from three Tuesdays ago. Except this buddy also writes your code while you're enjoying the walk.

Zero configuration. It finds your repos the way your eyes find a gap through the trees — automatically, instinctively, without thinking about it.

### Getting It In the Bag

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

If macOS quarantines it — and it will, because macOS treats unsigned binaries the way that one tree 40 feet off the tee treats your favorite driver. You know the tree. Everyone knows the tree.

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work
```

You'll need [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed. That's your putter. Nobody leaves the house without a putter. Not even the guy who only throws Destroyers.

For the PR stuff — comments, reviews — you need the [GitHub CLI](https://cli.github.com/). Think of it as your mini marker. You *can* play without it, but you're going to want it:

```bash
brew install gh && gh auth login
```

### The Caddy Book

Type `work` and you're looking at the caddy book — every hole laid out in front of you. Tasks on the left, details on the right. It's UDisc, your spotter, and a course map rolled into one terminal window.

### Playing the Round

**Teeing off — press `n`**

An overlay pops up right on the dashboard. Pick your repos, name the task, set a branch per repo — like choosing your disc, visualizing the line, setting your feet. `work` creates the worktrees and launches Claude. One keystroke to tee off. Clean and confident.

**Coming back to your disc — press `r`**

You marked your disc. You walked up. Now you throw. Press `r` on any task and Claude opens in a fresh tab — new session, full context, right directory. If the session's still running, `r` walks you straight up to it. No searching. No reracking. You're back on the fairway in one keystroke.

Spot a PR in the dashboard with your name on it? Press `r` there too. `work` finds your matching task like you finding your disc in a field — or if it doesn't exist yet, it sets up a new one with the branch and repo already dialed in.

**Checking the line — press `t`**

Your code's been thrown. It's sitting on the green — pushed to the remote, waiting. You want to walk up and actually see it from the pin. Press `t`.

Before anything happens, `work` checks your lie. All worktrees pushed? Local repos clean? No uncommitted rough? Good. Then: fetch from remote, remove the worktrees (keeping the branches), checkout the task branch in each local repo, pull. Your local environment is on the task branches. Walk up and read the line. Run your test suite. Fire up the emulator. Play the approach shot with full information.

Try to press `t` with unpushed commits or a dirty local repo and `work` won't let you pull the disc. It reads the OB stakes better than you do at 6am on a fog round.

**Signing the card — press `c`**

Hole complete. PR merged. Press `c` and `work` cleans everything up — worktrees removed, branches deleted. Course record stands. On to the next hole.

### Course Management

Each task gets its own worktree tucked away in `.worktrees/` — like having separate bags for casual rounds and tournament play. Your original repos are untouchable. `work` sets up deny rules so Claude can't edit them — O.B. stakes that actually work, not those flimsy ones that fall over when a Tilt rolls through.

Build config files get symlinked automatically. Your main directory stays clean. Tournament ready at all times.

**PRs:** Press `p` to open a PR, `o` to view it, `m` to read review comments. All optional. All there when you need them.

### New Plastic

```bash
work update
```

Alerts you when a new run drops. You know the feeling — your favorite mold just got a new stamp and you're refreshing the shop page like it's DGPT coverage on a Sunday afternoon. Except this one's always in stock. And always free.

*May your trees be decoration, your gaps be wide, and your code ship as clean as a flat hyzer on a calm morning.*

</details>

<details>
<summary><strong>Chip Thunderson</strong> — I LITERALLY CANNOT CONTAIN MYSELF</summary>

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

**You need**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) — but honestly if you don't already have this installed WHAT ARE YOU EVEN DOING WITH YOUR LIFE

**ALSO**: The [GitHub CLI](https://cli.github.com/) for PR features! `brew install gh && gh auth login`! TWO COMMANDS!

### Commands (EVERY SINGLE ONE IS A MASTERPIECE)

| Command | WHAT IT DOES (AMAZINGLY) |
|---------|-------------|
| `work` | A LIVE! REAL-TIME! DASHBOARD! With tasks! Sessions! And PR stuff! I CAN'T EVEN! |
| `work update` | Updates itself! IT IMPROVES ITSELF! LIKE A SELF-PERFECTING DIAMOND! |
| `work version` | Prints the version! EVEN THIS IS SOMEHOW EXCITING! |

### THE TASK LIFECYCLE — I NEED TO LIE DOWN

**STARTING A TASK — press `n`**

AN OVERLAY WIZARD POPS UP RIGHT ON THE DASHBOARD! No full-screen chaos! No context loss! Just an elegant, multi-step form that asks you which repos you're working in, what to name the task, and what branches to use! And THEN — and I need you to be sitting down for this — it CREATES ALL THE WORKTREES! By itself! While you watch a little spinner! And THEN it launches CLAUDE! AUTOMATICALLY! IN A NEW TAB! I AM ABSOLUTELY INCONSOLABLE WITH EXCITEMENT!

**RESUMING A TASK — press `r`**

DID YOU STEP AWAY? DID LIFE HAPPEN? It's OKAY! Press `r` and a NEW Claude session opens in a new terminal tab with FULL CONTEXT! If the session is STILL RUNNING — because maybe you just got distracted — `r` FOCUSES THE EXISTING TAB instead! It KNOWS! HOW DOES IT KNOW?!

AND WAIT THERE'S MORE! Press `r` on a PR ROW in the dashboard — a PR that has your name on it — and `work` FINDS THE MATCHING TASK AUTOMATICALLY! If there's no task yet, it LAUNCHES THE WIZARD PRE-FILLED WITH THE PR BRANCH AND REPO! You don't have to type anything! IT ALREADY KNOWS WHAT YOU WANT! I am sobbing actual tears of productivity joy right now!

**TESTING LOCALLY — press `t`**

OKAY. OKAY. Brace yourself. You've got a task. Code is pushed. You want to actually RUN it locally — in your real repos, not just the worktrees. You press `t`. And `work` — BEFORE DOING ANYTHING — checks that all your worktrees are in `PUSHED` state and all your local repos are CLEAN. It is PROTECTING YOU FROM YOURSELF and honestly I respect the confidence. If everything checks out, it FETCHES THE LATEST, REMOVES THE WORKTREES (KEEPING THE BRANCHES), CHECKS OUT THE TASK BRANCHES IN YOUR LOCAL REPOS, AND PULLS. Your entire local environment is on the task branches IN ONE KEYSTROKE. I genuinely cannot believe this is real software that exists!

**CLEANING UP — press `c`**

Task done? PR merged? Press `c`. Worktrees GONE. Branches DELETED. Workspace PRISTINE! READY FOR THE NEXT ADVENTURE!

### How It Works (PREPARE TO BE AMAZED AGAIN)

- Worktrees at `<workspace>/.worktrees/<task-name>/<repo>/` — BEAUTIFUL organization
- Deny rules PROTECT your original repos — NOTHING gets accidentally modified
- Build files get AUTOMATICALLY symlinked — it thinks of EVERYTHING
- Your main directory stays PRISTINE, UNTOUCHED, PERFECT

**PR features**: `p` creates PRs! `o` opens them! `m` shows review comments! It's all OPTIONAL and it's all INCREDIBLE!

### Updating (IT GETS EVEN BETTER OVER TIME?!)

```bash
work update
```

It tells you when new versions are available because OF COURSE IT DOES. This tool doesn't just solve problems, it ANTICIPATES them!

</details>

<details>
<summary><strong>Ned Flatline</strong> — I have some concerns</summary>

## work

So somebody made a wrapper around git worktrees and the `gh` CLI and called it a product. Bold move. Let's walk through this.

`work` manages parallel Claude Code sessions. The pitch is that you can run multiple AI coding sessions across repos without them interfering with each other. Which, okay, fine — but you know what else prevents interference? Not running multiple sessions at the same time. Nobody talks about that option. It's free and it already works.

It "auto-discovers" your repos. I put that in quotes because what it actually does is look at what folders exist in your directory. My file manager does this too. I don't see Finder writing a README about it.

Zero configuration is the headline feature here, which is a bit like a restaurant bragging that you don't have to build your own chair before sitting down. The bar is where it is, I suppose.

### Installation

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

It's not code-signed, so macOS will quarantine it. You'll need to manually override that, which is always a great sign for software you're about to trust with your codebase:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work
```

Requires [Claude Code](https://docs.anthropic.com/en/docs/claude-code). So this is a tool that wraps another tool that wraps an AI that writes code. We're three layers of abstraction away from actually doing anything. Impressive in its own way.

You'll also want the [GitHub CLI](https://cli.github.com/) for PR features. `brew install gh && gh auth login`. So that's Homebrew to install `gh` to enable features in `work` which wraps Claude Code which calls an API. It's dependencies all the way down.

### What It Does

You type `work` and you get a dashboard. It shows your tasks in two panels. It's a list on the left and details on the right. I've seen this layout in every application since Microsoft Outlook 2003.

**Creating a task — `n`**

Press `n` and an overlay wizard appears. You pick repos, name the task, set a branch per repo. `work` creates the worktrees and launches Claude. The entire thing is automated, which means you have fewer opportunities to think carefully about what you're doing. Some people consider this a feature.

**Resuming a task — `r`**

Press `r` and Claude opens in a new terminal tab. If the session is already running, `r` focuses it instead. You can also press `r` on a PR row to find the matching task, or create one pre-filled from the PR. The tool is trying to save you the effort of remembering things, which is either helpful or a crutch, depending on how you feel about that sort of thing.

**Testing locally — `t`**

This one actually has some engineering behind it. Press `t` and `work` first checks that all worktrees are in `PUSHED` state and all local repos are clean. Then it fetches, removes the worktrees, checks out the task branches in your local repos, and pulls. You get your local environment on the task branches for testing.

To be fair, doing this manually involves `git fetch`, `git worktree remove`, `git checkout`, and `git pull` per repo. Pressing `t` is faster. I'll acknowledge that. Grudgingly.

The pre-flight checks are a nice touch — it refuses to proceed if you have unpushed work or dirty repos, which prevents the specific class of mistakes that people who need a tool like this are likely to make.

**Cleaning up — `c`**

Press `c` when you're done. Worktrees removed, branches deleted. Functionally equivalent to a shell loop, but it requires less typing. Fine.

### How It Works

Worktrees get created in a `.worktrees/` folder. Your original repos are protected by "deny rules" so Claude can't edit them, which is reassuring in the way that a "this building is earthquake-resistant" sign is reassuring — you're glad it's there, but a little concerned about why it needed to be said.

Build files get symlinked. It works.

PR features: `p` creates a PR, `o` opens it, `m` shows comments. It works when `gh` is installed. The tool degrades gracefully when it isn't, which is good design even if I'm not going to thank them for it.

### Updating

```bash
work update
```

It tells you when there's a new version. So it can do what `brew upgrade` already does. Neat.

</details>

<details>
<summary><strong>Gary Beige</strong> — It's a tool. It exists. Whatever.</summary>

## work

It's a CLI tool. It manages worktrees for Claude Code. You can run multiple sessions.

### Install

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

You'll need Claude Code installed. You probably already know that.

For PR features: `brew install gh && gh auth login`. Or don't. Up to you.

### Commands

| Command | Description |
|---------|-------------|
| `work` | Opens the dashboard. Shows your tasks. |
| `work update` | Updates |
| `work version` | Prints the version |

### Dashboard

The dashboard shows tasks on the left and details on the right. You can press keys to do things.

**`n`** — New task wizard. Pick repos, name it, set branches. Creates worktrees and opens Claude.

**`r`** — Resume. Opens a new Claude tab for the task. If there's already an active session, focuses that instead. Works on PR rows too — finds the matching task or sets up a new one.

**`t`** — Test. Switches your local repos to the task branches so you can test locally. Checks first that worktrees are pushed and local repos are clean. Removes worktrees, checks out branches, pulls. Does what it says.

**`c`** — Clean. Removes worktrees and deletes branches. For when you're done.

### PR Management

If you have `gh` installed, you can create and monitor PRs from the terminal. Press `p` to make one, `o` to open it, `m` to read comments. If you don't have `gh`, the dashboard still works fine.

### How it works

Worktrees go in `.worktrees/`. Build files get symlinked. Your main directory isn't affected.

### Updating

```bash
work update
```

</details>

<details>
<summary><strong>Joey Codiani</strong> — Sup with the whack README, sup!</summary>

## work — How YOU doin', developer?

Okay okay okay, so check it, right? You ever been coding, and you got like, a bunch of repos — and they're all like, *right there* — and you're trying to do stuff in all of em at the same time but they keep like, bumping into each other? It's whack!

So my boy `work` over here, he's like — yo, I got you. He uses these things called [git worktrees](https://git-scm.com/docs/git-worktree) which are like — okay, you know when you got a sandwich, right, and you don't want your meatball sub touching your turkey club? So you put em in separate bags? It's like that but for code. Each task gets its own bag. Its own branch. Its own whole situation.

And if you're working across like, multiple repos? ONE Claude session sees everything, dude. It's like having the smartest guy in the room, except the room is all the rooms. You know what I'm saying? ...Yeah, me neither, but it works!

Zero configuration, bro. It just FINDS your repos. It's like a code bloodhound or whatever. You don't gotta do nothin'. Just show up.

### Gettin' Set Up (It's Easy, I Did It, and I'm ME)

Alright so you do this Homebrew thing — no, not like actual beer, it's a computer thing. Trust me, I was confused too:

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

BOOM. Done. That's it. Could I BE any more installed right now?

If your Mac gets all weird about it:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work
```

You also need this [Claude Code](https://docs.anthropic.com/en/docs/claude-code) thing. `npm install -g @anthropic-ai/claude-code`. Don't ask me what npm stands for. I asked once and the answer made me tired.

And for the PR stuff you need the [GitHub CLI](https://cli.github.com/): `brew install gh && gh auth login`. It'll ask you to log in through your browser. Worth it for the comments and review stuff.

### What's Up With All The Commands, Sup

| Command | The Deal |
|---------|-------------|
| `work` | Dude. DUDE. It's like a mission control but on your computer. Everything's right there. Tasks. Sessions. PR stuff. |
| `work update` | Gets you the new new. The latest version. Fresh out the oven. |
| `work version` | Tells you what version you got. Quick and simple. Like a name tag but for software. |

### The Dashboard — Let Me Walk You Through This

Okay so you just type `work` and BAM — dashboard, right? Two panels. Left side has your tasks, right side has the details. And there's keys you press to do stuff.

**Starting a new task — press `n`**

So this little wizard overlay pops up — doesn't take over the whole screen or whatever, just kinda appears on top. And it asks you stuff! Which repos are you working in? What do you wanna call this task? What branch? You answer the questions, hit enter a bunch of times, and then `work` just... makes the worktrees and opens Claude. I'm not gonna pretend I know exactly how that works but it DOES and it's great.

**Coming back to a task — press `r`**

Okay so say you had to close your laptop or whatever. Your task is still in the list. Press `r` and boom — new Claude session, new terminal tab, right back in it. Full context. Right directory. Everything's already set up.

OH and here's the part that got me — if the Claude session is STILL OPEN from before? `r` just focuses that tab instead of making a new one. It KNOWS. Like a friend who knows exactly which couch you fell asleep on and just walks you back there. That's the vibe.

AND ALSO — okay stay with me here — if you see a PR in the dashboard with your name on it, you can press `r` on THAT too. And `work` finds your matching task! If there's no task, it opens the wizard but it already filled in the branch and the repo for you. You barely have to do anything! This tool does more work than I do, and I'm supposed to be the one using it!

**Testing the thing — press `t`**

Alright so your code is written, it's pushed, and now you wanna actually like... run it. On your computer. In your real repos. Not the worktree copies. So you press `t`.

But here's the thing — `work` doesn't just let you do it. It checks first. Like a bouncer. Are all your worktrees pushed? Is your local repo clean? No dirty stuff? No "I'll push this later" commits? Okay THEN you can come in. And then it fetches the latest, removes the worktrees (but keeps the branches!), checks out the task branches in your actual repos, and pulls. And now you're set to test.

If something's not clean or not pushed? It tells you and says no. Like a bouncer who actually reads your ID instead of just looking at it. Respect.

**Done with the task — press `c`**

Task done? PR merged? Press `c` and it cleans everything up. Worktrees gone, branches deleted. Fresh start. On to the next thing. Like returning your tray at a cafeteria except the tray is made of branches and git history. ...That metaphor got away from me.

### PR stuff, real quick

If you've got the GitHub CLI installed: `p` makes PRs, `o` opens them in the browser, `m` lets you read review comments and reply without leaving the terminal. It all works. If you DON'T have it installed, the dashboard still works fine, it just doesn't do the PR stuff. No pressure either way!

### How It Works — I'll Try To Explain

So your worktrees go in this `.worktrees/` folder, right? And your REAL repos — the originals — they don't get touched. At all. It's like a stunt double for your code. The stunt double takes all the hits, the original stays looking fresh.

Build files get symlinked, which — okay I'm not gonna pretend I know exactly what that means but basically stuff that needs to be in two places is in two places without actually BEING in two places. It's like how I can be the most handsome guy in two different rooms. By standing in the doorway. ...That sounded better in my head.

### Updating

```bash
work update
```

It tells you when there's a new version! Very thoughtful! Unlike some PEOPLE I know who never tell you when there's pizza in the break room!

*How YOU doin'?*

</details>

## License

MIT
