# work — Claude Code Worktree Manager

Choose your preferred reading experience:

<details>
<summary><strong>The Straight Shooter</strong> — Just the facts, no nonsense</summary>

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
<summary><strong>The Bard</strong> — Forsooth, a README most noble</summary>

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
<summary><strong>The Free Spirit</strong> — Like, the code just flows through you, man</summary>

## work — A Worktree Manager for Claude Code, in Harmony with the Universe

Hey there, beautiful soul. Take a deep breath. Center yourself. Now... imagine a world where your code sessions exist in perfect balance. No conflicts. No chaos. Just pure, parallel harmony.

That's `work`.

It uses [git worktrees](https://git-scm.com/docs/git-worktree) — which, if you think about it, are really just the universe's way of letting your code exist in multiple dimensions simultaneously. Each task gets its own branch, its own space, its own *energy*. And they coexist peacefully, man. Like trees in a forest. Actually, they literally are trees. Worktrees. It's all connected.

For multi-repo tasks, one Claude session holds space for all of them. One consciousness, many repos. We're all one, you know?

Zero configuration. It discovers your repos organically, the way nature intended.

### Manifesting Your Installation

#### Through the Homebrew Collective

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

If macOS blocks your path — and isn't that just like the system, man — release the quarantine energy:

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work    # Apple Silicon
xattr -d com.apple.quarantine /usr/local/bin/work       # Intel
```

#### Growing It From Source (So Rewarding)

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

**You'll also want**: [Claude Code](https://docs.anthropic.com/en/docs/claude-code) — your AI companion on this journey (`npm install -g @anthropic-ai/claude-code`)

### The Pathways (Commands)

| Pathway | Where It Leads |
|---------|-------------|
| `work` | The crossroads — continue your journey or begin a new one |
| `work dashboard` | Your meditation space — a real-time visualization of all your creative energy |
| `work list` | Reflect on the state of your worktrees — PUSHED, UNPUSHED, DIRTY, or at peace (CLEAN) |
| `work pr [task]` | Send your creation out into the world for collective feedback |
| `work done` | Release a task back into the void with gratitude |
| `work clean` | Let go of what no longer serves you |
| `work <repo> <branch>` | Skip the ceremony, follow your intuition directly |
| `work update` | Receive the latest wisdom from the upstream collective |

### The Dashboard (Your Sacred Space)

The dashboard radiates the energy of your pull requests through sacred symbols:

- `○` Open — your offering has been placed upon the altar, awaiting the community's gaze
- `✓` Approved — the circle has blessed your contribution. Namaste
- `!` Changes requested — a gentle nudge from the universe to refine your craft
- `●` Merged — your code has become one with the main branch. You are the main branch. We all are
- `✗` Closed — every ending is a new beginning, friend

Press `p` to share your work with the community. Press `o` to visit your offering in the browser. It even pushes your branches for you, because `work` believes in supporting your journey, not burdening it.

### The Way It Flows

- Each task grows its own worktree at `<workspace>/.worktrees/<task-name>/<repo>/`
- The worktrees exist separately from the original repos — healthy boundaries are important, even in code
- Your main directory remains in its natural, untouched state
- Build files are symlinked — connected yet independent, like all of us

### Renewal

```bash
work update
```

Growth is a continuous process. The tool lets you know when a new version is ready to emerge.

### License

MIT — Because knowledge, like love, should be free.

Peace. :v:

</details>

<details>
<summary><strong>The Ballad</strong> — Sing along if you know the words</summary>

## The Ballad of `work`

### :musical_note: Verse 1

*Well I woke up this morning with repos on my mind,*
*Had branches all conflicting, of every shape and kind,*
*I needed parallel sessions but they kept stepping on their toes,*
*Then I found a little CLI and this is how it goes...*

### :musical_note: Chorus

*Oh, `work`! Sweet `work`!*
*With your worktrees standing tall,*
*You give each task its own little branch,*
*And isolation for them all!*
*Oh, `work`! Sweet `work`!*
*No configuration needed,*
*You scan my directories on your own,*
*My prayers have been heeded!*

### :musical_note: Verse 2 — The Installation Ballad

*Now if you want to join this song, the setup's pretty quick,*
*Just tap and install with Homebrew, it's a mighty simple trick:*

```bash
brew tap tSquaredd/homebrew-tap
brew install --cask work
```

*But if macOS gives you trouble, don't you shed a single tear,*
*Just clear that quarantine away and let the music clear:*

```bash
xattr -d com.apple.quarantine /opt/homebrew/bin/work
```

*Or build it from the source yourself, for those who like to craft:*

```bash
go install github.com/tSquaredd/work-cli/cmd/work@latest
```

*But don't forget your [Claude Code](https://docs.anthropic.com/en/docs/claude-code), friend, or you'll be up the mast!*

### :musical_note: Chorus (reprise)

*Oh, `work`! Sweet `work`!...*

### :musical_note: Verse 3 — The Commands

*Now let me sing the commands to you, each one a little gem,*
*Just type `work` to start it up, it's the finest of all them!*

*The `dashboard` shows your tasks alive, with PRs gleaming bright,*
*And `list` will show your worktrees, from DIRTY through to right,*
*Type `pr` to make a pull request without a browser tab,*
*And `done` will tear things down for you — don't worry, it won't grab*
*A single branch that's unpushed, no sir, it warns you first,*
*And `clean` sweeps up the tidy ones to quench your cleanup thirst!*

### :musical_note: Bridge — The Dashboard

*Now the dashboard, oh the dashboard, it's a sight to make you weep,*
*Two panels showing everything while your sessions run so deep,*
*A circle means it's open, and a checkmark means approved,*
*An exclamation? Changes wanted — but your spirit won't be moved!*
*A purple dot means merged, my friend, your code's in main at last,*
*And if you see that little x... well... let's not dwell on the past.*

*Press `p` to make a PR, press `o` to see it bloom,*
*Press `r` to resume a session, there's always enough room,*
*Press `n` to start a new task and `c` to clean your plate,*
*Press `d` to see the diff, and `a` to find your mate!*

### :musical_note: Verse 4 — How It Works

*Your worktrees live in `.worktrees/`, each task has got a home,*
*Deny rules guard the originals so Claude won't freely roam,*
*Your build files all get symlinked, and your main dir stays pristine,*
*It's the finest code arrangement that these eyes have ever seen!*

### :musical_note: Outro

*So if your branches tangle and your sessions start to fight,*
*Just `work update` and carry on, everything will be alright,*
*For `work` will keep on spinning up those worktrees one by one,*
*Until the last commit is pushed...*

*And the merge... is... done.*

:musical_note: :musical_note: :musical_note:

### License

*MIT — free for you, and free for me, from sea to shining sea!*

</details>

<details>
<summary><strong>The Hype Beast</strong> — I LITERALLY CANNOT CONTAIN MYSELF</summary>

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
<summary><strong>The Nihilist</strong> — It's a tool. It exists. Whatever.</summary>

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
