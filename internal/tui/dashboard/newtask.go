package dashboard

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
	"github.com/tSquaredd/work-cli/internal/workspace"
)

// repoConfig holds per-repo branch configuration for the new task wizard.
type repoConfig struct {
	Alias      string
	Branch     string
	BaseBranch string
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

func newNewTaskModel(ws *workspace.Workspace) newTaskModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	return newTaskModel{
		ws:      ws,
		spinner: s,
	}
}

// initPickRepos creates the repo selection form.
func (m *newTaskModel) initPickRepos() tea.Cmd {
	m.step = stepPickRepos

	options := make([]huh.Option[string], len(m.ws.Repos))
	for i, r := range m.ws.Repos {
		label := fmt.Sprintf("%s  — %s", r.Alias, r.Description)
		options[i] = huh.NewOption(label, r.Alias)
	}

	m.selectedAliases = nil
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which repo(s) are you working in?").
				Options(options...).
				Value(&m.selectedAliases),
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
	repo := m.selectedRepos[m.configIdx]

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
		// Save config for current repo
		branch := m.curBranch
		if branch == "" {
			branch = m.selectedRepos[m.configIdx].Prefix + "-" + m.taskName
		}
		baseBranch := m.curBaseBranch
		if baseBranch == "" {
			baseBranch = worktree.CurrentBranch(m.selectedRepos[m.configIdx].Path)
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
	b.WriteString(titleStyle.Render("New Task"))
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

