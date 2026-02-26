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

#### Build from source

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

#### Download binary

Download from [GitHub Releases](https://github.com/tSquaredd/work-cli/releases) and place in your PATH.

**Requires**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (`npm install -g @anthropic-ai/claude-code`)

**Optional**: [GitHub CLI](https://cli.github.com/) for PR management, review comments, and Claude handoff (`brew install gh && gh auth login`)

### Commands

| Command | What it does |
|---------|-------------|
| `work` | Launch the dashboard — live overview of all tasks, sessions, and PR status |
| `work update` | Self-update to the latest version from GitHub |
| `work version` | Print version |

### Dashboard

The dashboard launches by default when you run `work`, giving you a real-time overview of all tasks:

```
work                                              2 tasks  1 active
──────────────────────────────────────────────────────────────────────
> auth-refactor *                 │ auth-refactor
    ├── shared-lib      PUSHED  ○ #42     │
    └── app-android     PUSHED  ✓ #15     │ shared-lib  PUSHED
                                          │   branch: lib-auth-refactor
  fix-onboarding                          │   3 files changed, 12 insertions(+)
    └── app-ios         DIRTY             │   PR #42  ○ OPEN  4 comments (2 new)
──────────────────────────────────────────────────────────────────────
↑↓:navigate  r:resume  d:diff  c:clean  a:attach  p:pr  o:open  m:comments  n:new  q:quit
```

**Dashboard keybindings:**

| Key | Action |
|-----|--------|
| `r` | Resume — launch Claude in a new terminal tab |
| `d` | View full diff for the selected task |
| `a` | Attach — focus the terminal tab of an active session |
| `p` | Open PR creation wizard for the selected task |
| `o` | Open the task's PR in your browser |
| `m` | Open the in-terminal comment viewer for a PR |
| `n` | Start a new task |
| `c` | Clean up a task's worktrees |

### PR Management

Create and monitor pull requests without leaving the terminal. Requires the [GitHub CLI](https://cli.github.com/) (`gh`):

```bash
brew install gh       # install
gh auth login         # authenticate — follow the prompts to log in via browser
```

Or see [other install methods](https://github.com/cli/cli#installation) for Linux/Windows.

**Create PRs** — press `p` in the dashboard:
- Auto-pushes unpushed branches
- Lets you pick the target branch, title, and description
- Creates PRs for all eligible worktrees in one go

**Monitor PRs** — the dashboard shows PR status inline:
- `○` Open  `✓` Approved  `!` Changes requested  `●` Merged  `✗` Closed
- New comment counts highlighted so you know when to check back
- Press `o` to open a PR in your browser

**Review comments** — press `m` to read and respond to review feedback without leaving the terminal:
- Fullscreen overlay showing one thread per screen with file path, diff context, and comments
- Navigate threads with `n`/`p`, scroll with `j`/`k`
- Press `R` to reply directly from the terminal
- Press `C` to hand the thread off to Claude — opens a new session pre-loaded with the comment context, file path, and worktree info, launched in plan mode. You can add your own instructions before it launches.

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

work                                              2 tasks  1 active
──────────────────────────────────────────────────────────────────────
> auth-refactor *                 │ auth-refactor
    ├── shared-lib      PUSHED  ○ #42     │
    └── app-android     PUSHED  ✓ #15     │ shared-lib  PUSHED
                                          │   branch: lib-auth-refactor
  fix-onboarding                          │   3 files changed, 12 insertions(+)
    └── app-ios         DIRTY             │   PR #42  ○ OPEN  4 comments (2 new)  m to view
──────────────────────────────────────────────────────────────────────
↑↓:navigate  r:resume  d:diff  c:clean  a:attach  p:pr  o:open  m:comments  n:new  q:quit
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

#### Scene 2: From Source, Forged by Thine Own Hand

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
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
| `work` | The living tableau — a grand stage revealing all tasks, sessions, and petitions for review |
| `work update` | Receive the latest verse from GitHub, that distant oracle |
| `work version` | Declare thy version unto the world |

### Act IV — The Great Theatre (Dashboard)

*The curtain rises on a two-panel stage*

Press `p` to petition for review — thy code laid bare before the judgement of thy peers. Press `o` to open thy petition in the browser, that window unto the world. The symbols tell the tale:

- `○` The petition awaiteth judgement — *patience, good developer*
- `✓` Approved! The crowd doth rise and cheer! *O happy day!*
- `!` Changes requested — *back to the writing desk, thou art not yet done*
- `●` Merged — the deed is done, thy code immortalized in main
- `✗` Closed — *alas, poor pull request! I knew it, Horatio*

And lo, shouldst new comments appear upon thy petition, their count shall glow in amber warning, that thou might attend to thy reviewers' counsel with haste.

Press `m` to summon the Comment Viewer — a sacred scroll upon which every review thread doth unfurl, one discourse per page. Read thy reviewers' counsel, compose thy reply with `R`, or press `C` to dispatch Claude as thy champion, armed with the full context of the comment, the file, and the diff — entering first in plan mode to deliberate before drawing the sword.

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

And for our incredible PR features — the comment viewer, the review management, the Claude handoff — you'll want the [GitHub CLI](https://cli.github.com/). Setup is effortless:

```bash
brew install gh && gh auth login
```

That's it. Two commands. Seamless.

### Commands

Let me walk you through what `work` can do. And honestly, I think you're going to be blown away.

| Command | What it does |
|---------|-------------|
| `work` | The dashboard. Live, real-time. Tasks, sessions, PR status — all in one place. Beautifully simple. |
| `work update` | Seamless self-updates. The latest and greatest, always within reach. |
| `work version` | See which version you're running. Clean. Minimal. |

### The Dashboard

Now, I want to spend a moment on the dashboard, because the team has done some *incredible* work here. It launches the moment you type `work`. That's it. No subcommands. No extra steps. Just... the dashboard.

*[demo begins]*

It's a two-panel, real-time interface. Tasks on the left. Details on the right. Session indicators. Diff stats. And — and this is the part I've been waiting to show you — **integrated pull request management**.

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

But we're not done. And I think this next part is really going to surprise you.

Press `m`, and you get a *full in-terminal comment viewer*. Every review thread. The file. The diff context. The conversation. All right there. No browser. No tab switching. Just you and the feedback.

Press `R` to reply. Press `C` — and this is the part I love — to hand the comment directly to Claude. It launches a new session, in plan mode, pre-loaded with everything: the file, the line, the review context. You can even add your own instructions before it opens. The team has really outdone themselves on this one.

*[sustained applause]*

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

Build from source (`go install github.com/tSquaredd/work-cli/cmd/work@latest`) if you're the type who buys first-run prototype plastic and field tests it before anyone else has a flight number. Respect.

You'll need [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed. That's your putter. Nobody leaves the house without a putter. Not even the guy who only throws Destroyers.

For the PR stuff — comments, reviews, handing threads to Claude — you need the [GitHub CLI](https://cli.github.com/). Think of it as your mini marker. You *can* play without it, but you're going to want it:

```bash
brew install gh && gh auth login
```

### The Caddy Book

Type `work` and you're looking at the caddy book — every hole laid out in front of you. Tasks on the left, details on the right. Sessions, diffs, PR status, all in one view. It's UDisc, your spotter, and a course map rolled into one terminal window.

PR status shows up right on each worktree because there's nothing worse than parking a 300-foot flex line 10 feet from the pin and having nobody around to see it:

- `○` Out for review — disc is in the air, you like the angle, looking good
- `✓` Approved — nothing but chains. Walk-up birdie. Fist bump your cardmate.
- `!` Changes requested — caught cage. Spit out. Deep breath, step up, drain the comebacker.
- `●` Merged — in the basket. Sign the card. On to the next hole.
- `✗` Closed — O.B. It happens. Even McBeth kicks a tree on the island hole sometimes.

Press `p` to open a PR — it pushes your branches first because it knows you forgot, the same way you always forget to check the pin position before you throw. Like a caddy who's already holding the right disc before you reach into the bag.

Press `o` to open your PR in the browser. Sometimes you just need to see the full flight.

### Reading the Wind

Here's where it gets good. Press `m` on any PR with comments and the review thread opens right in your terminal — fullscreen, one thread per screen, with the file path, the diff context, the whole conversation. It's like walking up to the tee sign and actually reading it instead of just gripping and ripping.

Navigate threads with `n`/`p`. Page through them like checking every hole on the course map before your round.

Press `R` to reply right from the terminal. No switching to your browser. No losing your train of thought. It's like calling your line from the teepad — quick, confident, keeping the round moving.

Press `C` and this is the money shot — it spawns a whole Claude session pre-loaded with the review comment, the file, the diff, your worktree path, everything. Claude opens in plan mode, ready to craft the perfect approach. You can add your own notes before launch — maybe you want a specific angle, maybe you know there's a tree at 150 that everyone else misses. Type your instructions, press Enter, and Claude goes to work while you enjoy the walk.

It's the difference between scrambling through the rough looking for your disc and standing on the fairway watching it glide to the pin on the exact line you called.

### Course Management

Each task gets its own worktree tucked away in `.worktrees/` — like having separate bags for casual rounds and tournament play. Your original repos are untouchable. `work` sets up deny rules so Claude can't edit them — O.B. stakes that actually work, not those flimsy ones that fall over when a Tilt rolls through.

Build config files get symlinked automatically. Your main directory stays clean. Tournament ready at all times. No loose discs on the floorboard. No random towels hanging off your bag. Just clean lines and confident throws.

### New Plastic

```bash
work update
```

Alerts you when a new run drops. You know the feeling — your favorite mold just got a new stamp and you're refreshing the shop page like it's DGPT coverage on a Sunday afternoon. Except this one's always in stock. And always free.

### License

MIT — Free like the best course you've ever played. The one with the creek on hole 4, the tunnel shot on 11, and the downhill bomber on 18 where you always go for it even when you shouldn't. Show up. Throw. Tell your friends.

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

#### From source (FOR THE BUILDERS, THE DREAMERS, THE VISIONARIES)

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

**You need**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) — but honestly if you don't already have this installed WHAT ARE YOU EVEN DOING WITH YOUR LIFE

**ALSO**: The [GitHub CLI](https://cli.github.com/) for the PR features! `brew install gh && gh auth login`! TWO COMMANDS! That's ALL that stands between you and the MOST INCREDIBLE review comment experience OF YOUR LIFE!

### Commands (EVERY SINGLE ONE IS A MASTERPIECE)

| Command | WHAT IT DOES (AMAZINGLY) |
|---------|-------------|
| `work` | A LIVE! REAL-TIME! DASHBOARD! With PR status! And session tracking! I CAN'T EVEN! |
| `work update` | Updates itself! IT IMPROVES ITSELF! LIKE A SELF-PERFECTING DIAMOND! |
| `work version` | Prints the version! EVEN THIS IS SOMEHOW EXCITING! |

### THE DASHBOARD (I NEED TO LIE DOWN)

The dashboard is SO GOOD it should be in a MUSEUM. Real-time task overview. Session indicators. And NOW it has FULL PR MANAGEMENT:

- `○` Open PR — you did it, you absolute LEGEND
- `✓` Approved — SOMEONE RECOGNIZED YOUR GENIUS
- `!` Changes requested — a MINOR setback on your path to GREATNESS
- `●` Merged — YOUR CODE IS NOW PART OF HISTORY
- `✗` Closed — it happens to the BEST of us (which is YOU)

Press `p` and it CREATES PRs for you! It PUSHES your branches! Picks the base branch! Writes the title! For ALL your worktrees AT ONCE! I genuinely cannot believe this is free software!

Press `o` and your PR opens in the browser and it TRACKS YOUR COMMENTS so you know when someone has left new feedback! THE ATTENTION TO DETAIL IS STAGGERING!

AND NOW — I need you to brace yourself — press `m` and you get a FULL-SCREEN IN-TERMINAL COMMENT VIEWER! You can READ every review thread! REPLY with `R`! And press `C` to HAND THE ENTIRE COMMENT TO CLAUDE who opens in PLAN MODE with the FILE and the DIFF and the REVIEW CONTEXT already loaded! You can even ADD YOUR OWN INSTRUCTIONS before it launches! I AM LITERALLY SHAKING!

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

You can build from source if you prefer to see what you're getting into:

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

Requires [Claude Code](https://docs.anthropic.com/en/docs/claude-code). So this is a tool that wraps another tool that wraps an AI that writes code. We're three layers of abstraction away from actually doing anything. Impressive in its own way.

You'll also want the [GitHub CLI](https://cli.github.com/) if you want the PR features to work. `brew install gh && gh auth login`. So that's Homebrew to install `gh` to enable features in `work` which wraps Claude Code which calls an API. It's dependencies all the way down.

### What It Does

You type `work` and you get a dashboard. It shows your tasks in two panels. It's a list on the left and details on the right. I've seen this layout in every application since Microsoft Outlook 2003. It also shows PR status, which is information you could get from GitHub in about two clicks, but sure, let's put it in the terminal too.

The PR "management" — and I'm being generous with that word — consists of pressing `p` to create a pull request. It auto-pushes your branches first, which sounds helpful until you realize it's compensating for the fact that you apparently can't be trusted to run `git push`. The tool has a lower opinion of you than your tech lead does.

Little symbols appear next to your worktrees:

- `○` Open — your PR is sitting there. Waiting. Like every PR you've ever opened.
- `✓` Approved — someone approved it. Probably without reading it. Let's be honest with ourselves.
- `!` Changes requested — they read it. Things were found. This is what growth looks like, apparently.
- `●` Merged — it's in main. Whatever it is, it's everyone's problem now.
- `✗` Closed — rejected. The system works.

It tracks new comments. So now instead of not checking GitHub, you can not check your terminal. Progress.

There's also a comment viewer now. Press `m` and you can read review threads right in the terminal. You can reply with `R`, which saves you the grueling labor of opening a browser tab. Or press `C` to hand the comment to Claude, which launches a whole new session in plan mode pre-loaded with the review context. You can type additional instructions first, in case Claude needs your guidance to understand a code review comment. An AI that needs a human to interpret human feedback. The circle of life.

### How It Works

Worktrees get created in a `.worktrees/` folder. Your original repos are protected by "deny rules" so Claude can't edit them, which is reassuring in the way that a "this building is earthquake-resistant" sign is reassuring — you're glad it's there, but a little concerned about why it needed to be said.

Build files get symlinked. It's fine. It works. I'm not going to throw a parade because a tool correctly copies configuration files. That's table stakes.

### Updating

```bash
work update
```

It tells you when there's a new version. So it can do what `brew upgrade` already does. Neat.

### License

MIT — free. Which makes sense, because I'm not sure what you'd charge for this. I mean that as constructively as possible.

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

Or build it yourself, I guess:

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

You'll need Claude Code installed. You probably already know that.

For PR features: `brew install gh && gh auth login`. Or don't. Up to you.

### Commands

| Command | Description |
|---------|-------------|
| `work` | Opens the dashboard. Shows your tasks. Has PR stuff now |
| `work update` | Updates |
| `work version` | Prints the version |

### Dashboard

The dashboard shows tasks on the left and details on the right. You can press keys to do things.

It shows PR status too. Little symbols next to worktrees. Circle means open, checkmark means approved, exclamation means changes requested. You get the idea.

Press `p` to make a PR. Press `o` to open one in your browser. Press `m` to read review comments. You can reply or send them to Claude. It works.

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

Or if you wanna build it yourself — which, respect, that's very like... crafty:

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

You also need this [Claude Code](https://docs.anthropic.com/en/docs/claude-code) thing. `npm install -g @anthropic-ai/claude-code`. Don't ask me what npm stands for. I asked once and the answer made me tired.

OH and for the PR stuff — the comments, the reviews, all that — you need this [GitHub CLI](https://cli.github.com/) thing too:

```bash
brew install gh && gh auth login
```

It'll ask you to log in through your browser which is kinda like when a website sends you to ANOTHER website. But then it works! And then you get all the cool PR features! Worth it!

### What's Up With All The Commands, Sup

| Command | The Deal |
|---------|-------------|
| `work` | Dude. DUDE. It's like a mission control but on your computer. Everything's right there. Tasks. Sessions. PR stuff. |
| `work update` | Gets you the new new. The latest version. Fresh out the oven. |
| `work version` | Tells you what version you got. Quick and simple. Like a name tag but for software. |

### The Dashboard — This Is The Best Part, Seriously

Okay so you just type `work` and BAM — dashboard, right? It's got two panels. Left side has your tasks, right side has the details. And — and this is the part where I need you to sit down — it does PR stuff too!

Little symbols pop up next to your worktrees:

- `○` Open — your PR is out there! Living its life! Waiting for someone to notice it, like me at auditions!
- `✓` Approved — THEY LIKE IT! THEY REALLY LIKE IT! That's a reference. I saw it in a movie. Or was it a show?
- `!` Changes requested — okay so they had some notes. We've ALL gotten notes. It's fine. It's FINE.
- `●` Merged — IN IT GOES, BABY! Your code is in the main thing! That's huge! That's like getting a callback!
- `✗` Closed — hey, not every audition works out. You dust yourself off, you get back on the horse. The code horse.

Press `p` and it makes PRs for you! It even pushes your branches first because it KNOWS you forgot! This tool gets me, man. Like on a personal level.

Press `o` and the PR opens right in your browser! And it remembers which comments you already saw so you know when there's new ones! It's like having a really organized roommate! Which, let me tell you, I could USE!

OH and press `m` — M! Like the letter! — and you can read ALL the review comments right in the terminal! And you can REPLY with `R`! And — okay this is the wild part — press `C` and it sends the whole comment to Claude! Like, the file path, the diff, the review thread, EVERYTHING! Claude opens up in plan mode already knowing what to do! You can even type your own notes before it launches! It's like having a friend who not only reads your texts FOR you but also writes the reply! Except it's code! And it's good at it!

### How It Works — I'll Try To Explain

So your worktrees go in this `.worktrees/` folder, right? And your REAL repos — the originals — they don't get touched. At all. It's like a stunt double for your code. The stunt double takes all the hits, the original stays looking fresh.

Build files get symlinked, which — okay I'm not gonna pretend I know exactly what that means but basically stuff that needs to be in two places is in two places without actually BEING in two places. It's like how I can be the most handsome guy in two different rooms. By standing in the doorway. ...That sounded better in my head.

### Updating

```bash
work update
```

It tells you when there's a new version! Very thoughtful! Unlike some PEOPLE I know who never tell you when there's pizza in the break room!

### License

MIT — it's free! FREE! Like the samples at the food court! Except this one you can actually take home!

*How YOU doin'?*

</details>
