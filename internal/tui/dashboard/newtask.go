package dashboard

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// repoConfig holds per-repo branch configuration for the new task wizard.
type repoConfig struct {
	Alias      string
	Branch     string
	BaseBranch string
	SwitchTo   string // if non-empty, checkout this branch in main repo before creating worktree
}

// newTaskStep tracks the current wizard step.
type newTaskStep int

const (
	stepPickRepos  newTaskStep = iota
	stepTaskName
	stepConfigRepo
	stepCreating
	stepDone
)

// newTaskModel is the overlay model for the new task wizard.
type newTaskModel struct {
	ws *workspace.Workspace

	step    newTaskStep
	form    *huh.Form
	spinner spinner.Model

	// Resume-from-PR mode (non-nil = resume mode)
	resumePR           *service.StandalonePR
	needsSwitchForPR   bool   // true when the PR repo branch is checked out and needs a switch first
	needsSwitchForRepo bool   // true when a non-PR repo's selected branch is checked out in main repo
	pendingBranch      string // the intended branch while awaiting the needsSwitchForRepo form

	// Collected data
	selectedRepos []workspace.Repo
	taskName      string
	configs       []repoConfig
	configIdx     int // which repo we're configuring (branch + base)

	// Form value bindings (huh writes into these)
	selectedAliases []string // bound to MultiSelect in stepPickRepos
	curBranch       string
	curBaseBranch   string

	// Creation results
	createdDirs []string
	progress    []string
	createErr   error

	width  int
	height int
}

func newNewTaskModel(ws *workspace.Workspace) *newTaskModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	return &newTaskModel{
		ws:      ws,
		spinner: s,
	}
}

func newResumeFromPRModel(ws *workspace.Workspace, pr *service.StandalonePR) *newTaskModel {
	m := newNewTaskModel(ws)
	m.resumePR = pr
	m.taskName = pr.HeadBranch
	m.selectedAliases = []string{pr.RepoAlias}
	return m
}

// initPickRepos creates the repo selection form.
func (m *newTaskModel) initPickRepos() tea.Cmd {
	m.step = stepPickRepos

	options := make([]huh.Option[string], len(m.ws.Repos))
	for i, r := range m.ws.Repos {
		label := fmt.Sprintf("%s  — %s", r.Alias, r.Description)
		options[i] = huh.NewOption(label, r.Alias)
	}

	// In resume-from-PR mode, keep pre-selected aliases; otherwise reset.
	if m.resumePR == nil {
		m.selectedAliases = nil
	}

	validate := func(s []string) error {
		if len(s) == 0 {
			return fmt.Errorf("select at least one repo (space to toggle)")
		}
		if m.resumePR != nil {
			found := false
			for _, a := range s {
				if a == m.resumePR.RepoAlias {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("%s must be selected (it has the PR)", m.resumePR.RepoAlias)
			}
		}
		return nil
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which repo(s) are you working in?").
				Options(options...).
				Value(&m.selectedAliases).
				Validate(validate),
		),
	).WithTheme(ui.HuhTheme()).
		WithWidth(m.formWidth()).
		WithShowHelp(true)

	return m.form.Init()
}

// initTaskName creates the task name input form.
func (m *newTaskModel) initTaskName() tea.Cmd {
	m.step = stepTaskName

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Task name").
				Placeholder("e.g. auth-refactor, remove-parse-session").
				Value(&m.taskName),
		),
	).WithTheme(ui.HuhTheme()).
		WithWidth(m.formWidth()).
		WithShowHelp(true)

	return m.form.Init()
}

// initConfigRepo creates the branch config form for the current repo.
func (m *newTaskModel) initConfigRepo() tea.Cmd {
	m.step = stepConfigRepo

	// In resume-from-PR mode, auto-configure repos that have the PR branch.
	if m.resumePR != nil {
		for m.configIdx < len(m.selectedRepos) {
			repo := m.selectedRepos[m.configIdx]
			isPRRepo := repo.Alias == m.resumePR.RepoAlias
			if isPRRepo {
				// Fix 2: detect if PR branch is currently checked out in main repo.
				currentBranch := worktree.Branch(repo.Path)
				if currentBranch == m.resumePR.HeadBranch {
					// Branch is checked out — need user to pick a branch to switch to first.
					m.needsSwitchForPR = true
					break
				}
				// Branch not checked out — auto-configure.
				m.configs = append(m.configs, repoConfig{
					Alias:      repo.Alias,
					Branch:     m.resumePR.HeadBranch,
					BaseBranch: m.resumePR.HeadBranch,
				})
				m.configIdx++
				continue
			}
			if worktree.HasRemoteBranch(repo.Path, m.resumePR.HeadBranch) {
				m.configs = append(m.configs, repoConfig{
					Alias:      repo.Alias,
					Branch:     m.resumePR.HeadBranch,
					BaseBranch: m.resumePR.HeadBranch,
				})
				m.configIdx++
				continue
			}
			break // this repo needs manual config (Fix 1)
		}
		// If all repos were auto-configured, jump to creation.
		if m.configIdx >= len(m.selectedRepos) {
			m.step = stepCreating
			m.form = nil
			return tea.Batch(
				m.spinner.Tick,
				createWorktrees(m.ws, m.taskName, m.configs),
			)
		}
	}

	repo := m.selectedRepos[m.configIdx]
	// Fetch remote refs so AllBranches can include remote-only branches.
	if m.resumePR != nil || m.needsSwitchForPR || m.needsSwitchForRepo {
		worktree.FetchAll(repo.Path)
	}

	// Fix 2: show "switch to branch" form for PR repo when branch is checked out.
	if m.needsSwitchForPR {
		allBranches := worktree.AllBranches(repo.Path)
		var opts []huh.Option[string]
		for _, b := range allBranches {
			if b != m.resumePR.HeadBranch {
				opts = append(opts, huh.NewOption(b, b))
			}
		}
		m.curBranch = ""
		if len(opts) > 0 {
			m.curBranch = opts[0].Value
		}
		m.form = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("%s — Switch to branch", repo.Alias)).
					Description(fmt.Sprintf("%s is checked out. Pick a branch to switch %s to so the worktree can be created.", m.resumePR.HeadBranch, repo.Alias)).
					Options(opts...).
					Value(&m.curBranch).
					Filtering(true).
					Height(10),
			),
		).WithTheme(ui.HuhTheme()).WithWidth(m.formWidth()).WithShowHelp(true)
		return m.form.Init()
	}

	// needsSwitchForRepo: show "switch to branch" form for non-PR repo when selected branch is checked out.
	if m.needsSwitchForRepo {
		allBranches := worktree.AllBranches(repo.Path)
		worktreeBranches := worktree.WorktreeBranches(repo.Path)
		var opts []huh.Option[string]
		for _, b := range allBranches {
			if b != m.pendingBranch && !worktreeBranches[b] {
				opts = append(opts, huh.NewOption(b, b))
			}
		}
		m.curBranch = ""
		if len(opts) > 0 {
			m.curBranch = opts[0].Value
		}
		m.form = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("%s — Switch to branch", repo.Alias)).
					Description(fmt.Sprintf("%s is checked out. Pick a branch to switch %s to so the worktree can be created.", m.pendingBranch, repo.Alias)).
					Options(opts...).
					Value(&m.curBranch).
					Filtering(true).
					Height(10),
			),
		).WithTheme(ui.HuhTheme()).WithWidth(m.formWidth()).WithShowHelp(true)
		return m.form.Init()
	}

	// Fix 1: for non-PR repos in resume mode, show a local branch selector.
	if m.resumePR != nil {
		allBranches := worktree.AllBranches(repo.Path)
		currentBranch := worktree.CurrentBranch(repo.Path)
		worktreeBranches := worktree.WorktreeBranches(repo.Path)
		m.curBranch = currentBranch
		m.curBaseBranch = ""

		var opts []huh.Option[string]
		for _, b := range allBranches {
			if worktreeBranches[b] && b != currentBranch {
				// Already in a worktree — skip entirely
				continue
			}
			label := b
			if b == currentBranch {
				label = b + " (current)"
			}
			opts = append(opts, huh.NewOption(label, b))
		}

		if len(opts) > 0 {
			m.form = huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title(fmt.Sprintf("%s — Branch", repo.Alias)).
						Description("Select existing branch to include in this task").
						Options(opts...).
						Value(&m.curBranch).
						Filtering(true).
						Height(10),
				),
			).WithTheme(ui.HuhTheme()).WithWidth(m.formWidth()).WithShowHelp(true)
			return m.form.Init()
		}
		// fallthrough to normal form if no opts found
	}

	// Default branch name
	m.curBranch = repo.Prefix + "-" + m.taskName

	// Get recent branches for base selection
	recentBranches := worktree.RecentBranches(repo.Path, 15)
	currentBranch := worktree.CurrentBranch(repo.Path)
	m.curBaseBranch = currentBranch

	var fields []huh.Field

	// Branch name input
	fields = append(fields,
		huh.NewInput().
			Title(fmt.Sprintf("%s — Branch name", repo.Alias)).
			Value(&m.curBranch),
	)

	// Base branch selection
	if len(recentBranches) > 0 {
		branchOptions := make([]huh.Option[string], 0, len(recentBranches))
		for _, b := range recentBranches {
			label := b
			if b == currentBranch {
				label = b + " (current)"
			}
			branchOptions = append(branchOptions, huh.NewOption(label, b))
		}
		fields = append(fields,
			huh.NewSelect[string]().
				Title(fmt.Sprintf("%s — Base branch", repo.Alias)).
				Description("Fetches latest before creating worktree").
				Options(branchOptions...).
				Value(&m.curBaseBranch).
				Height(8),
		)
	}

	m.form = huh.NewForm(
		huh.NewGroup(fields...),
	).WithTheme(ui.HuhTheme()).
		WithWidth(m.formWidth()).
		WithShowHelp(true)

	return m.form.Init()
}

// advanceStep collects data from the completed form and moves to the next step.
// Returns a tea.Cmd for the next form init (or creation command).
func (m *newTaskModel) advanceStep() tea.Cmd {
	switch m.step {
	case stepPickRepos:
		// Collect selected repos (huh wrote into m.selectedAliases)
		m.selectedRepos = nil
		for _, alias := range m.selectedAliases {
			if r := m.ws.RepoByAlias(alias); r != nil {
				m.selectedRepos = append(m.selectedRepos, *r)
			}
		}

		if len(m.selectedRepos) == 0 {
			return func() tea.Msg { return newTaskFormCancelMsg{} }
		}
		return m.initTaskName()

	case stepTaskName:
		m.taskName = sanitizeTaskName(m.taskName)
		if m.taskName == "" {
			return func() tea.Msg { return newTaskFormCancelMsg{} }
		}
		m.configIdx = 0
		m.configs = make([]repoConfig, 0, len(m.selectedRepos))
		return m.initConfigRepo()

	case stepConfigRepo:
		// Fix 2: handle the PR repo branch-switch form.
		if m.needsSwitchForPR {
			m.configs = append(m.configs, repoConfig{
				Alias:      m.selectedRepos[m.configIdx].Alias,
				Branch:     m.resumePR.HeadBranch,
				BaseBranch: m.resumePR.HeadBranch,
				SwitchTo:   m.curBranch,
			})
			m.needsSwitchForPR = false
			m.configIdx++
			if m.configIdx < len(m.selectedRepos) {
				return m.initConfigRepo()
			}
			m.step = stepCreating
			m.form = nil
			return tea.Batch(m.spinner.Tick, createWorktrees(m.ws, m.taskName, m.configs))
		}

		// Handle the non-PR repo branch-switch form.
		if m.needsSwitchForRepo {
			m.configs = append(m.configs, repoConfig{
				Alias:      m.selectedRepos[m.configIdx].Alias,
				Branch:     m.pendingBranch,
				BaseBranch: m.pendingBranch,
				SwitchTo:   m.curBranch,
			})
			m.needsSwitchForRepo = false
			m.pendingBranch = ""
			m.configIdx++
			if m.configIdx < len(m.selectedRepos) {
				return m.initConfigRepo()
			}
			m.step = stepCreating
			m.form = nil
			return tea.Batch(m.spinner.Tick, createWorktrees(m.ws, m.taskName, m.configs))
		}

		// Save config for current repo
		branch := m.curBranch
		if branch == "" {
			branch = m.selectedRepos[m.configIdx].Prefix + "-" + m.taskName
		}
		baseBranch := m.curBaseBranch
		if baseBranch == "" {
			baseBranch = worktree.CurrentBranch(m.selectedRepos[m.configIdx].Path)
		}

		// In resume mode, if user selected the currently checked-out branch, need a SwitchTo first.
		if m.resumePR != nil && branch == worktree.CurrentBranch(m.selectedRepos[m.configIdx].Path) {
			m.needsSwitchForRepo = true
			m.pendingBranch = branch
			return m.initConfigRepo()
		}

		m.configs = append(m.configs, repoConfig{
			Alias:      m.selectedRepos[m.configIdx].Alias,
			Branch:     branch,
			BaseBranch: baseBranch,
		})

		m.configIdx++
		if m.configIdx < len(m.selectedRepos) {
			// More repos to configure
			return m.initConfigRepo()
		}

		// All repos configured — start creation
		m.step = stepCreating
		m.form = nil
		return tea.Batch(
			m.spinner.Tick,
			createWorktrees(m.ws, m.taskName, m.configs),
		)
	}

	return nil
}

// update handles a Bubble Tea message for the overlay.
func (m *newTaskModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return nil

	case spinner.TickMsg:
		if m.step == stepCreating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
		return nil
	}

	// Delegate to active form
	if m.form != nil {
		model, cmd := m.form.Update(msg)
		m.form = model.(*huh.Form)

		// Check if form completed
		if m.form.State == huh.StateCompleted {
			return m.advanceStep()
		}
		// Check if form was aborted (Esc)
		if m.form.State == huh.StateAborted {
			return func() tea.Msg { return newTaskFormCancelMsg{} }
		}

		return cmd
	}

	return nil
}

// view renders the overlay.
func (m *newTaskModel) view() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true)

	dimStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)

	var b strings.Builder

	// Title
	title := "New Task"
	if m.resumePR != nil {
		title = fmt.Sprintf("Resume PR #%d", m.resumePR.Number)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width, 60))))
	b.WriteString("\n\n")

	switch m.step {
	case stepPickRepos, stepTaskName, stepConfigRepo:
		if m.form != nil {
			b.WriteString(m.form.View())
		}

	case stepCreating:
		b.WriteString(m.spinner.View() + " Creating worktrees for " + ui.StyleInfo.Render(m.taskName) + "...\n")

	case stepDone:
		if m.createErr != nil {
			b.WriteString(ui.StyleDanger.Render("Error: " + m.createErr.Error()))
			b.WriteString("\n\n")
			b.WriteString(dimStyle.Render("Press Esc to return"))
		} else {
			for _, line := range m.progress {
				b.WriteString("  " + line + "\n")
			}
			b.WriteString("\n")
			b.WriteString(ui.StyleSuccess.Render("Launching Claude..."))
		}
	}

	// Pad to full screen
	content := b.String()
	padded := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2, 4).
		Render(content)

	return padded
}

func (m *newTaskModel) formWidth() int {
	w := m.width - 8 // padding
	if w < 40 {
		w = 40
	}
	if w > 80 {
		w = 80
	}
	return w
}

func sanitizeTaskName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	re := regexp.MustCompile(`[^a-z0-9-]`)
	return re.ReplaceAllString(name, "")
}

