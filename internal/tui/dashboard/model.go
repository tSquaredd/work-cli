package dashboard

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/claude"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/session"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
)

const refreshInterval = 5 * time.Second

// Model is the root Bubble Tea model for the dashboard.
type Model struct {
	svc       *service.WorkService
	taskList  taskListModel
	detail    detailModel
	statusBar statusBarModel

	width  int
	height int

	// State
	filtering   bool
	filterInput string
	confirming  bool
	confirmTask string
	quitting    bool
}

// New creates a new dashboard model.
func New(svc *service.WorkService) Model {
	return Model{
		svc:       svc,
		taskList:  newTaskListModel(),
		detail:    newDetailModel(),
		statusBar: newStatusBarModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadTasks(m.svc),
		tickCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		return m, nil

	case tasksLoadedMsg:
		m.taskList.setTasks(msg.tasks)
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case diffLoadedMsg:
		if sel := m.taskList.selected(); sel != nil && sel.Name == msg.taskName {
			m.detail.diffText = msg.diff
			m.detail.diffDir = msg.dir
			m.detail.showDiff = true
			m.updateStatusBar()
		}
		return m, nil

	case actionResultMsg:
		m.statusBar.message = msg.message
		m.confirming = false
		m.confirmTask = ""
		// Clear message after a delay and refresh
		return m, tea.Batch(
			loadTasks(m.svc),
			clearMessageCmd(),
		)

	case clearMessageMsg:
		m.statusBar.message = ""
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			loadTasks(m.svc),
			tickCmd(),
		)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle filter mode
	if m.filtering {
		return m.handleFilterKey(msg)
	}

	// Handle confirmation mode
	if m.confirming {
		return m.handleConfirmKey(msg)
	}

	// Handle diff scroll mode
	if m.detail.showDiff {
		switch key {
		case "j", "down":
			m.detail.scrollDown()
			return m, nil
		case "k", "up":
			m.detail.scrollUp()
			return m, nil
		case "d", "esc":
			m.detail.showDiff = false
			m.detail.scroll = 0
			m.updateStatusBar()
			return m, nil
		case "q":
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "j", "down":
		m.taskList.moveDown()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "k", "up":
		m.taskList.moveUp()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "enter":
		m.taskList.toggleExpand()
		return m, nil

	case "r":
		return m.handleResume()

	case "d":
		return m.handleDiff()

	case "c":
		return m.handleClean()

	case "a":
		return m.handleAttach()

	case "/":
		m.filtering = true
		m.filterInput = ""
		m.statusBar.filtering = true
		m.statusBar.filter = ""
		return m, nil
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "enter":
		m.filtering = false
		m.taskList.filter = m.filterInput
		m.taskList.cursor = 0
		m.statusBar.filtering = false
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "esc":
		m.filtering = false
		m.filterInput = ""
		m.taskList.filter = ""
		m.taskList.cursor = 0
		m.statusBar.filtering = false
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "backspace":
		if len(m.filterInput) > 0 {
			m.filterInput = m.filterInput[:len(m.filterInput)-1]
			m.statusBar.filter = m.filterInput
		}
		return m, nil

	default:
		if len(key) == 1 {
			m.filterInput += key
			m.statusBar.filter = m.filterInput
		}
		return m, nil
	}
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "y", "Y":
		taskName := m.confirmTask
		return m, func() tea.Msg {
			err := m.cleanTask(taskName)
			if err != nil {
				return actionResultMsg{message: fmt.Sprintf("Error: %s", err), isError: true}
			}
			return actionResultMsg{message: fmt.Sprintf("Cleaned %s", taskName)}
		}

	case "n", "N", "esc":
		m.confirming = false
		m.confirmTask = ""
		m.statusBar.message = ""
		return m, nil
	}

	return m, nil
}

// Action handlers

func (m Model) handleResume() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil {
		return m, nil
	}

	// Build the claude command to run in a new tab
	dirs := sel.Dirs()
	if len(dirs) == 0 {
		return m, nil
	}

	// Prepare claude files
	cfg := claude.LaunchConfig{
		Workspace: m.svc.Workspace,
		TaskName:  sel.Name,
		Dirs:      dirs,
	}

	if err := claude.Prepare(cfg); err != nil {
		m.statusBar.message = fmt.Sprintf("Error preparing: %s", err)
		return m, clearMessageCmd()
	}

	// Build command string
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		m.statusBar.message = "claude not found in PATH"
		return m, clearMessageCmd()
	}

	systemPrompt := claude.BuildSystemPrompt(cfg)
	var cmdParts []string
	cmdParts = append(cmdParts, fmt.Sprintf("cd %q", dirs[0]))

	args := []string{claudePath}
	for i, d := range dirs {
		if i > 0 {
			args = append(args, "--add-dir", d)
		}
	}
	args = append(args, "--append-system-prompt", fmt.Sprintf("%q", systemPrompt))
	cmdParts = append(cmdParts, strings.Join(args, " "))

	command := strings.Join(cmdParts, " && ")
	tabTitle := "work: " + sel.Name

	// Spawn in new terminal tab
	opener := session.DetectTerminal()
	pid, err := opener.OpenTab(command, tabTitle)
	if err != nil {
		m.statusBar.message = fmt.Sprintf("Tab spawn failed: %s", err)
		return m, clearMessageCmd()
	}

	// Register session
	if m.svc.Tracker != nil {
		rec := session.SessionRecord{
			TaskName:      sel.Name,
			PID:           pid,
			Dirs:          dirs,
			LaunchedAt:    time.Now(),
			TerminalTab:   tabTitle,
			WorkspaceRoot: m.svc.Workspace.Root,
		}
		_ = m.svc.Tracker.Register(rec)
	}

	m.statusBar.message = fmt.Sprintf("Launched %s in new tab", sel.Name)
	return m, tea.Batch(
		loadTasks(m.svc),
		clearMessageCmd(),
	)
}

func (m Model) handleDiff() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil {
		return m, nil
	}

	// Load diff for first worktree (or all combined)
	if len(sel.Worktrees) > 0 {
		dir := sel.Worktrees[0].Dir
		return m, loadDiff(sel.Name, dir)
	}

	return m, nil
}

func (m Model) handleClean() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil {
		return m, nil
	}

	// Check if any worktree has dirty changes
	hasDirty := false
	for _, wt := range sel.Worktrees {
		if wt.Status == worktree.StatusDirty {
			hasDirty = true
			break
		}
	}

	if hasDirty {
		m.confirming = true
		m.confirmTask = sel.Name
		m.statusBar.message = fmt.Sprintf("Clean %s? Has uncommitted changes! (y/n)", sel.Name)
		return m, nil
	}

	m.confirming = true
	m.confirmTask = sel.Name
	m.statusBar.message = fmt.Sprintf("Clean %s? (y/n)", sel.Name)
	return m, nil
}

func (m Model) handleAttach() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil || !sel.HasSession {
		return m, nil
	}

	if m.svc.Tracker != nil {
		err := session.Attach(m.svc.Tracker, sel.Name)
		if err != nil {
			m.statusBar.message = fmt.Sprintf("Attach failed: %s", err)
			return m, clearMessageCmd()
		}
		m.statusBar.message = fmt.Sprintf("Focused %s", sel.Name)
		return m, clearMessageCmd()
	}

	return m, nil
}

func (m Model) cleanTask(taskName string) error {
	detail := m.svc.TaskDetail(taskName)
	if detail == nil {
		return fmt.Errorf("task %q not found", taskName)
	}

	ws := m.svc.Workspace
	for _, wt := range detail.Worktrees {
		repo := ws.RepoByAlias(wt.Alias)
		if repo == nil {
			continue
		}
		if err := worktree.Remove(repo.Path, wt.Dir, true); err != nil {
			return fmt.Errorf("removing %s/%s: %w", taskName, wt.Alias, err)
		}
	}

	worktree.CleanupTaskDir(ws.Root, taskName)

	// Unregister session if tracked
	if m.svc.Tracker != nil {
		_ = m.svc.Tracker.Unregister(taskName)
	}

	return nil
}

// Layout and rendering

func (m *Model) updateLayout() {
	// Distribute width: ~35% task list, ~65% detail
	listWidth := m.width * 35 / 100
	if listWidth < 25 {
		listWidth = 25
	}
	detailWidth := m.width - listWidth - 3 // 3 for divider + padding

	contentHeight := m.height - 4 // header + status bar + borders

	m.taskList.width = listWidth
	m.taskList.height = contentHeight
	m.detail.width = detailWidth
	m.detail.height = contentHeight
	m.statusBar.width = m.width
}

func (m *Model) updateDetail() {
	m.detail.setTask(m.taskList.selected())
}

func (m *Model) updateStatusBar() {
	sel := m.taskList.selected()
	m.statusBar.hasTask = sel != nil
	m.statusBar.hasActive = sel != nil && sel.HasSession
	m.statusBar.showDiff = m.detail.showDiff
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Loading..."
	}

	// Narrow terminal: single panel mode
	if m.width < 60 {
		return m.singlePanelView()
	}

	return m.twoPanelView()
}

func (m Model) twoPanelView() string {
	contentHeight := m.height - 4

	// Header
	header := headerLine(m.taskList.filteredTasks(), m.width)

	// Divider under header
	divider := lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		Render(strings.Repeat("─", m.width))

	// Left panel
	leftStyle := lipgloss.NewStyle().
		Width(m.taskList.width).
		Height(contentHeight).
		MaxHeight(contentHeight)

	// Right panel
	rightStyle := lipgloss.NewStyle().
		Width(m.detail.width).
		Height(contentHeight).
		MaxHeight(contentHeight).
		PaddingLeft(2)

	// Vertical divider
	vertDiv := lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		Render("│")

	leftContent := leftStyle.Render(m.taskList.view())
	rightContent := rightStyle.Render(m.detail.view())

	// Join panels horizontally
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftContent, vertDiv, rightContent)

	// Status bar
	statusDivider := divider
	statusContent := m.statusBar.view()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		divider,
		panels,
		statusDivider,
		statusContent,
	)
}

func (m Model) singlePanelView() string {
	contentHeight := m.height - 4

	header := headerLine(m.taskList.filteredTasks(), m.width)
	divider := lipgloss.NewStyle().
		Foreground(ui.ColorMuted).
		Render(strings.Repeat("─", m.width))

	content := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(m.taskList.view())

	statusContent := m.statusBar.view()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		divider,
		content,
		divider,
		statusContent,
	)
}

// Helper commands

type clearMessageMsg struct{}

func clearMessageCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}
