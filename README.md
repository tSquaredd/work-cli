# work — Claude Code Worktree Manager

Choose your preferred reading experience:

<details>
<summary><strong>Joe Friday</strong> — Just the facts, no nonsense</summary>

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
<summary><strong>William Shakespeare</strong> — Forsooth, a README most noble</summary>

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

#### Scene 2: From Source, Forged by Thine Own Hand

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

**Thou must first possess**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code), that learned companion, without whom all is silence upon the stage (`npm install -g @anthropic-ai/claude-code`)

### Act III — The Instruments of Action

*DEVELOPER takes the throne. A flourish of trumpets.*

| Command | Its Purpose |
|---------|-------------|
| `work` | The interactive stage — resume a prior scene or begin anew |
| `work dashboard` | A living tableau of all tasks, sessions, and petitions for review |
| `work list` | Display all worktrees with their standing (PUSHED, UNPUSHED, DIRTY, or CLEAN) |
| `work pr [task]` | Compose pull requests — thy petition to the court of reviewers |
| `work done` | Select worktrees for their final curtain (with fair warning ere unpushed work is lost) |
| `work clean` | Sweep away all worktrees bearing no uncommitted changes, as a groundskeeper clearing the stage |
| `work <repo> <branch>` | A direct entrance — bypass the prologue entirely, for those who know their part |
| `work update` | Receive the latest verse from GitHub, that distant oracle |

### Act IV — The Great Theatre (Dashboard)

*The curtain rises on a two-panel stage*

Press `p` to petition for review — thy code laid bare before the judgement of thy peers. Press `o` to open thy petition in the browser, that window unto the world. The symbols tell the tale:

- `○` The petition awaiteth judgement — *patience, good developer*
- `✓` Approved! The crowd doth rise and cheer! *O happy day!*
- `!` Changes requested — *back to the writing desk, thou art not yet done*
- `●` Merged — the deed is done, thy code immortalized in main
- `✗` Closed — *alas, poor pull request! I knew it, Horatio*

And lo, shouldst new comments appear upon thy petition, their count shall glow in amber warning, that thou might attend to thy reviewers' counsel with haste.

### Act V — The Mechanics of This Wonder

*Aside, to the audience*

- Each task receiveth a worktree at `<workspace>/.worktrees/<task-name>/<repo>/`
- These worktrees dwell apart from the original repos, protected by deny rules most strict — as castle walls that guard the kingdom within
- Thy main working directory remaineth untouched, pure as new-fallen snow
- Build configuration files are symlinked, as servants attending their master through secret passages

### Epilogue

```bash
work update
```

A herald shall announce when newer versions await thee, appearing unbidden upon thy terminal as a ghost upon the battlements.

*Exeunt DEVELOPER, pursued by a merge conflict.*

*Fin.*

### License

MIT — Free as the air we breathe, given to all without restraint, as a sonnet cast upon the wind.

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

You can also build from source:

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

`work` pairs beautifully with [Claude Code](https://docs.anthropic.com/en/docs/claude-code). You're going to need that installed too. (`npm install -g @anthropic-ai/claude-code`)

### Commands

Let me walk you through what `work` can do. And honestly, I think you're going to be blown away.

| Command | What it does |
|---------|-------------|
| `work` | Your starting point. Resume a task or start a new one. Beautifully simple. |
| `work dashboard` | A live, real-time dashboard. Tasks, sessions, PR status — all in one place. |
| `work list` | See all your worktrees. PUSHED. UNPUSHED. DIRTY. CLEAN. At a glance. |
| `work pr [task]` | Create pull requests. Right from your terminal. We think this is a breakthrough. |
| `work done` | Thoughtfully tear down worktrees. It warns you before anything is lost. |
| `work clean` | Intelligently removes worktrees with no uncommitted changes. |
| `work <repo> <branch>` | Skip straight to what you need. Instant. |
| `work update` | Seamless self-updates. The latest and greatest, always within reach. |

### The Dashboard

Now, I want to spend a moment on the dashboard, because the team has done some *incredible* work here.

*[demo begins]*

It's a two-panel, real-time interface. Tasks on the left. Details on the right. Session indicators. Diff stats. And now — and this is the part I've been waiting to show you — **integrated pull request management**.

Let me show you what I mean.

- `○` Open — your PR is out for review
- `✓` Approved — and just look at that green checkmark
- `!` Changes requested — you'll know instantly
- `●` Merged — beautiful purple. Your code is in main.
- `✗` Closed

New comments appear highlighted. You always know when someone needs your attention.

Press `p` to create a PR. It pushes your branches, walks you through the title and description, and creates PRs across all your worktrees. In one flow.

Press `o` to open your PR in the browser. It even marks it as viewed. The little details matter, and we've sweated every single one.

*[pause]*

We really think this is going to change the way you work.

### Under the Hood

Now let me tell you a little about the technology.

- Worktrees are created at `<workspace>/.worktrees/<task-name>/<repo>/` — completely separate from your original repos
- Intelligent deny rules ensure Claude only edits worktree copies. Your main directory is never touched.
- Build files — `local.properties`, `.env` files — are automatically symlinked. It just works.

And it's all built on standard git. No proprietary formats. No vendor lock-in. Just git, the way it was meant to be used.

### One More Thing

```bash
work update
```

`work` tells you when a new version is available. Seamless updates. Always improving.

Because we believe the best developer tools aren't just something you use once. They're something that grows with you.

### Availability

`work` is available today. For free. MIT license.

*[thunderous applause]*

Thank you. We think you're going to love it.

</details>

<details>
<summary><strong>Ace Hyzer</strong> — Grip it, rip it, ship it</summary>

## work

Look, I'm going to be honest with you. I was mass-producing spaghetti code across three repos, branches tangled like headphone cables at the bottom of my disc bag, merge conflicts stacking up worse than a backup on hole 7 when the group ahead is putting with Bergs from 80 feet out. One by one. Into the wind.

Then I found `work` and it was like switching from a base-plastic Groove to a seasoned Halo Destroyer. Same arm. Completely different game.

`work` gives every task its own worktree — its own isolated branch, its own copy of the repo, nothing interfering with anything else. It's the coding equivalent of having the whole course to yourself on a Tuesday morning. No waiting. No distractions. Just clean lines and confident throws.

Got multiple repos? One Claude session sees all of them. It's like that buddy who somehow knows the line on every hole at every course within 50 miles. Except this buddy also writes your code.

### Getting It In the Bag

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

If macOS quarantines it — and it will, because macOS treats unsigned binaries the way a headwind treats your understable fairway driver:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work
```

You can also build from source (`go install github.com/tSquaredd/work-cli/cmd/work@latest`) if you're the type who field tests prototype plastic before it hits production. Respect.

You'll need [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed. That's your putter. You're not playing a round without a putter.

### Throwing a Round

`work` by itself drops you at the first tee pad — pick up where you left off or start fresh. `work dashboard` is the move though. Live leaderboard. Every task, every session, every PR, all right there. Two panels. It's like UDisc Live but for your codebase.

The dashboard now shows PR status right on each worktree, because nothing is more frustrating than parking your approach 10 feet out and having nobody see it. Little indicators tell you what's up:

- `○` Out for review — disc is in the air, looking good
- `✓` Approved — nothing but chains. Walk-up birdie.
- `!` Changes requested — caught cage. Kick-out. Gotta step up for the comebacker.
- `●` Merged — in the basket. Sign the card. Move on.
- `✗` Closed — O.B. It happens to everyone. Even McBeth shanks one now and then.

Press `p` to create a PR and `o` to open one in your browser. The tool auto-pushes your branches first because it knows you forgot. Like a caddy who hands you the right disc before you even reach into the bag.

### How It Plays

Each task gets its own worktree tucked away in `.worktrees/`. Your original repos are untouchable — `work` sets up deny rules so Claude can't edit them, like O.B. stakes that actually work. Build config files get symlinked so everything just runs. Your main directory stays clean. Tournament ready at all times.

### New Plastic

```bash
work update
```

Alerts you when a new version drops. You know the feeling — new run of your favorite mold just hit the shelves and you need to bag it before it sells out. Except this one's always in stock. And free.

### License

MIT — Free like a public course. Show up. Throw. Tell your friends.

*May all your putts be inside the circle and all your merge conflicts be fast-forwards.*

</details>

<details>
<summary><strong>Steve Ballmer</strong> — I LITERALLY CANNOT CONTAIN MYSELF</summary>

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
<summary><strong>Ben Stein</strong> — Let's not get carried away here</summary>

## work

Okay, so, I don't want to oversell this. It's a CLI tool. It manages git worktrees for Claude Code. That's... basically it. I know, I know, try to contain yourselves.

You can run multiple Claude sessions at the same time without them messing each other up. Which, to be fair, is something you could also achieve by just being more careful. But sure, this is fine too.

It auto-discovers your repos. That sounds fancy but it really just looks at what folders are in your directory. It's not exactly artificial intelligence. Well, actually, Claude is involved, so I guess technically — no, never mind. Let's move on.

### Installation

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

That's it. It's not a big deal. Homebrew does all the work. You could also build from source if you want to feel like you accomplished something today:

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

You need Claude Code installed. If you don't have it, this tool is just a fancy way to create git branches. Which, honestly, `git checkout -b` already does. But sure.

### What It Does

Type `work` and it asks if you want to resume something or start fresh. There's a `dashboard` that shows your tasks in two panels. It's... a dashboard. It has panels. The left one has a list, the right one has details. I'm not going to pretend this is groundbreaking UI design.

There's PR management now. You press `p` and it walks you through making a pull request. It pushes your branches for you first, which is nice I suppose, though one could argue it's enabling bad habits. Little icons show up next to your worktrees:

- `○` Open — someone might look at your code eventually
- `✓` Approved — they said it's fine. Don't read too much into it, they might have been in a hurry
- `!` Changes requested — don't take it personally. Okay, take it a little personally
- `●` Merged — congratulations, your code is now everyone's problem
- `✗` Closed — it's not rejection, it's... no, it's rejection

Press `o` to open a PR in your browser. It keeps track of comments so you know when there are new ones. Whether that's a feature or a source of anxiety depends on your team.

### How It Works

Worktrees go in `.worktrees/`. Your original repos don't get touched, which is probably the most useful thing about this whole operation if we're being honest. Build files get symlinked so things still compile. It's fine. It works.

### Updating

```bash
work update
```

It lets you know when there's a new version. Incrementally less disappointing each time, presumably.

### License

MIT — free, which, given everything I've just described, feels about right.

</details>

<details>
<summary><strong>Albert Camus</strong> — It's a tool. It exists. Whatever.</summary>

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
