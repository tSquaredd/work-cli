package dashboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/tSquaredd/work-cli/internal/claude"
	"github.com/tSquaredd/work-cli/internal/service"
	"github.com/tSquaredd/work-cli/internal/session"
	"github.com/tSquaredd/work-cli/internal/settings"
	"github.com/tSquaredd/work-cli/internal/ui"
	"github.com/tSquaredd/work-cli/internal/worktree"
)

const refreshInterval = 5 * time.Second

// Model is the root Bubble Tea model for the dashboard.
type Model struct {
	svc        *service.WorkService
	prEnricher *service.PREnricher
	taskList   taskListModel
	detail     detailModel
	statusBar  statusBarModel

	width  int
	height int

	// Comment viewer overlay
	comments     commentViewModel
	showComments bool

	// Diff viewer overlay
	diffView     diffViewModel
	showDiffView bool

	// New task overlay
	newTaskView  *newTaskModel
	showNewTask  bool

	// Resume confirm overlay (--dangerously-skip-permissions)
	showResumeConfirm     bool
	resumeConfirmTask     *service.TaskView
	resumeConfirmForm     *huh.Form
	resumeDangerouslySkip bool

	// Settings overlay
	showSettings bool
	settingsView *settingsModel

	// Standalone PRs
	myPRs    []service.StandalonePR
	otherPRs []service.StandalonePR

	// State
	confirming    bool
	confirmTask   string
	confirmAction string // "clean", "test", or "testPR"
	confirmPR     *service.StandalonePR // set when confirmAction == "testPR"
	quitting    bool
	newTask     bool // set when user presses 'n' to start a new task
	openPR               bool   // set when user presses 'p' to open PR wizard
	openPRWorktreeAlias  string // when at repo level, only open PR for this worktree
	prTick               int  // counter for throttled PR polling (every 6th tick = 30s)
	prDiscovered         bool // whether initial PR discovery has been done
	prStandaloneLoaded   bool // whether initial standalone PR fetch has been triggered
}

// New creates a new dashboard model.
func New(svc *service.WorkService) Model {
	return Model{
		svc:       svc,
		taskList:  newTaskListModel(),
		detail:    newDetailModel(),
		statusBar: newStatusBarModel(),
		diffView:  newDiffViewModel(),
	}
}

// SetPREnricher configures the PR enricher for live PR status.
func (m *Model) SetPREnricher(enricher *service.PREnricher) {
	m.prEnricher = enricher
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
		if m.showNewTask {
			m.newTaskView.width = msg.Width
			m.newTaskView.height = msg.Height
		}
		if m.showSettings && m.settingsView != nil {
			m.settingsView.width = msg.Width
			m.settingsView.height = msg.Height
		}
		return m, nil

	case tasksLoadedMsg:
		m.taskList.setTasks(msg.tasks)
		m.updateDetail()
		m.updateStatusBar()
		var cmds []tea.Cmd
		// On first load, trigger PR discovery and standalone PR fetch
		if !m.prDiscovered && m.prEnricher != nil {
			m.prDiscovered = true
			cmds = append(cmds, discoverPRs(m.prEnricher, msg.tasks))
		}
		if !m.prStandaloneLoaded && m.svc.GHAvailable {
			m.prStandaloneLoaded = true
			cmds = append(cmds, loadStandalonePRs(m.svc, msg.tasks))
		}
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case prStatusLoadedMsg:
		m.taskList.setTasks(msg.tasks)
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case standalonePRsLoadedMsg:
		if msg.err != nil {
			m.taskList.setPRError(fmt.Sprintf("PR list: %s", msg.err))
			return m, nil
		}
		m.myPRs = msg.mine
		m.otherPRs = msg.others
		m.taskList.setStandalonePRs(msg.mine, msg.others)
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case prDiffLoadedMsg:
		m.statusBar.loading = false
		if msg.err != nil {
			m.statusBar.message = fmt.Sprintf("Error loading diff: %s", msg.err)
			return m, clearMessageCmd()
		}
		m.diffView.setData(msg.repoDir, msg.repoAlias, msg.prNumber, msg.prTitle, msg.headSHA, msg.isMine, msg.diff)
		m.diffView.width = m.width
		m.diffView.height = m.height
		m.showDiffView = true
		m.updateStatusBar()
		return m, nil

	case reviewCommentPostedMsg:
		if msg.err != nil {
			m.diffView.message = fmt.Sprintf("Error: %s", msg.err)
		} else {
			m.diffView.message = "Comment posted"
			m.diffView.mode = diffViewBrowse
			m.diffView.commentBuf = nil
		}
		return m, nil

	case openBrowserMsg:
		openURL(msg.url)
		return m, nil

	case diffLoadedMsg:
		if sel := m.taskList.selected(); sel != nil && sel.Name == msg.taskName {
			m.detail.diffText = msg.diff
			m.detail.diffDir = msg.dir
			m.detail.showDiff = true
			m.updateStatusBar()
		}
		return m, nil

	case commentsLoadedMsg:
		m.statusBar.loading = false
		if msg.err != nil {
			m.statusBar.message = fmt.Sprintf("Error loading comments: %s", msg.err)
			return m, clearMessageCmd()
		}
		m.statusBar.message = ""
		m.comments.setData(msg.taskName, msg.repoAlias, msg.prNumber, msg.dir, msg.comments)
		m.comments.width = m.width
		m.comments.height = m.height
		m.showComments = true
		m.updateStatusBar()
		return m, nil

	case commentRepliedMsg:
		m.statusBar.loading = false
		if msg.err != nil {
			m.comments.replyErr = msg.err.Error()
			m.comments.mode = commentModeBrowse
			return m, nil
		}
		m.comments.cancelReply()
		m.statusBar.message = "Reply posted"
		// Refresh comments
		return m, tea.Batch(
			loadComments(m.comments.taskName, m.comments.repoAlias, m.comments.worktreeDir, m.comments.prNumber),
			clearMessageCmd(),
		)

	case actionResultMsg:
		m.statusBar.loading = false
		m.statusBar.message = msg.message
		m.confirming = false
		m.confirmTask = ""
		// Clear message after a delay and refresh
		return m, tea.Batch(
			loadTasks(m.svc),
			clearMessageCmd(),
		)

	case clearMessageMsg:
		m.statusBar.loading = false
		m.statusBar.message = ""
		return m, nil

	case tickMsg:
		m.prTick++
		cmds := []tea.Cmd{loadTasks(m.svc), tickCmd()}
		// Poll PR status every 6th tick (30s at 5s interval)
		if m.prEnricher != nil && m.prTick%6 == 0 {
			cmds = append(cmds, pollPRStatus(m.prEnricher, m.taskList.tasks))
		}
		// Refresh standalone PRs every 60th tick (5min)
		if m.svc.GHAvailable && m.prTick%60 == 0 {
			cmds = append(cmds, loadStandalonePRs(m.svc, m.taskList.tasks))
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		if m.statusBar.loading {
			var cmd tea.Cmd
			m.statusBar.spinner, cmd = m.statusBar.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case newTaskFormCancelMsg:
		m.showNewTask = false
		m.updateStatusBar()
		return m, nil

	case settingsSavedMsg:
		if err := settings.Save(msg.settings); err != nil {
			m.statusBar.message = fmt.Sprintf("Error saving settings: %s", err)
		} else {
			m.svc.RefreshSettings()
			m.statusBar.message = "Settings saved"
		}
		m.showSettings = false
		m.settingsView = nil
		m.updateStatusBar()
		return m, clearMessageCmd()

	case settingsCancelMsg:
		m.showSettings = false
		m.settingsView = nil
		m.updateStatusBar()
		return m, nil

	case newTaskCreatedMsg:
		m.newTaskView.step = stepDone
		m.newTaskView.createdDirs = msg.dirs
		m.newTaskView.progress = msg.progress
		m.newTaskView.createErr = msg.err
		if msg.err != nil {
			// Show error, user can press Esc to return
			return m, nil
		}
		// Launch Claude in a new terminal tab
		return m.launchNewTask()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward messages to new task overlay when active
	if m.showNewTask {
		cmd := m.newTaskView.update(msg)
		return m, cmd
	}

	// Forward messages to resume confirm overlay when active
	if m.showResumeConfirm && m.resumeConfirmForm != nil {
		model, cmd := m.resumeConfirmForm.Update(msg)
		m.resumeConfirmForm = model.(*huh.Form)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle new task overlay
	if m.showNewTask {
		return m.handleNewTaskKey(msg)
	}

	// Handle resume confirm overlay
	if m.showResumeConfirm {
		return m.handleResumeConfirmKey(msg)
	}

	// Handle settings overlay
	if m.showSettings {
		return m.handleSettingsKey(msg)
	}

	// Handle confirmation mode
	if m.confirming {
		return m.handleConfirmKey(msg)
	}

	// Handle diff viewer overlay
	if m.showDiffView {
		return m.handleDiffViewKey(msg)
	}

	// Handle comment viewer overlay
	if m.showComments {
		return m.handleCommentKey(msg)
	}

	// Handle diff scroll mode (inline detail panel diff)
	if m.detail.showDiff {
		switch key {
		case "down":
			m.detail.scrollDown()
			return m, nil
		case "up":
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

	// Check if cursor is on a repo header or standalone PR
	row := m.taskList.selectedRow()
	if row != nil && row.kind == rowRepoHeader {
		return m.handleRepoHeaderKey(msg)
	}
	if row != nil && (row.kind == rowMyPR || row.kind == rowOtherPR) {
		return m.handleStandalonePRKey(msg)
	}

	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "down":
		m.taskList.moveDown()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "up":
		m.taskList.moveUp()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "enter", "right":
		m.taskList.enterWorktrees()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "left":
		m.taskList.exitWorktrees()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "r":
		return m.handleResume()

	case "d":
		return m.handleDiff()

	case "D":
		return m.handlePRReview()

	case "c":
		return m.handleClean()

	case "t":
		return m.handleTest()

	case "a":
		return m.handleAttach()

	case "n":
		return m.startNewTask()

	case "p":
		return m.handleOpenPR()

	case "o":
		return m.handleBrowserOpen()

	case "m":
		return m.handleComments()

	case "s":
		return m.startSettings()
	}

	return m, nil
}

// handleResumeConfirmKey forwards keys to the resume confirm form, then advances
// or cancels the resume flow based on the form state.
func (m Model) handleResumeConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.showResumeConfirm = false
		m.resumeConfirmTask = nil
		m.resumeConfirmForm = nil
		m.updateStatusBar()
		return m, nil
	}

	if m.resumeConfirmForm == nil {
		m.showResumeConfirm = false
		m.updateStatusBar()
		return m, nil
	}

	model, cmd := m.resumeConfirmForm.Update(msg)
	m.resumeConfirmForm = model.(*huh.Form)

	switch m.resumeConfirmForm.State {
	case huh.StateAborted:
		m.showResumeConfirm = false
		m.resumeConfirmTask = nil
		m.resumeConfirmForm = nil
		m.updateStatusBar()
		return m, nil
	case huh.StateCompleted:
		sel := m.resumeConfirmTask
		m.showResumeConfirm = false
		m.resumeConfirmTask = nil
		m.resumeConfirmForm = nil
		m.updateStatusBar()
		if sel == nil {
			return m, nil
		}
		return m.resumeTask(sel)
	}

	return m, cmd
}

func (m Model) handleStandalonePRKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "down":
		m.taskList.moveDown()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "up":
		m.taskList.moveUp()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "r":
		row := m.taskList.selectedRow()
		if row != nil && row.kind == rowMyPR {
			return m.handleResumeFromPR()
		}
		return m, nil

	case "t":
		row := m.taskList.selectedRow()
		if row != nil && row.kind == rowMyPR {
			return m.handleTestStandalonePR()
		}
		return m, nil

	case "d":
		return m.handleStandalonePRDiff()

	case "m":
		return m.handleStandalonePRComments()

	case "o":
		return m.handleStandalonePRBrowserOpen()

	case "n":
		return m.startNewTask()
	}

	return m, nil
}

func (m Model) handleRepoHeaderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "down":
		m.taskList.moveDown()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "up":
		m.taskList.moveUp()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "enter", "right":
		m.taskList.expandRepoHeader()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "left":
		m.taskList.collapseRepoHeader()
		m.updateDetail()
		m.updateStatusBar()
		return m, nil

	case "n":
		return m.startNewTask()
	}

	return m, nil
}

func (m Model) handleDiffViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Comment post
	if m.diffView.mode == diffViewCommenting && key == "enter" {
		body := m.diffView.commentBody()
		if body == "" {
			m.diffView.mode = diffViewBrowse
			m.diffView.commentBuf = nil
			return m, nil
		}
		filePath, startLine, endLine, ok := m.diffView.selectedFileAndLines()
		if !ok {
			m.diffView.message = "Could not determine file/line for comment"
			m.diffView.mode = diffViewBrowse
			return m, nil
		}
		m.diffView.mode = diffViewBrowse
		return m, postReviewComment(
			m.diffView.repoDir, m.diffView.prNumber, m.diffView.headSHA,
			filePath, startLine, endLine, "RIGHT", body,
		)
	}

	// Claude launch
	if m.diffView.mode == diffViewClaudePrompt && key == "enter" {
		filePath, startLine, endLine, ok := m.diffView.selectedFileAndLines()
		if !ok {
			filePath = ""
			startLine = 0
			endLine = 0
		}
		m.diffView.claudeRequested = true
		m.diffView.claudeContext = &claudeReviewContext{
			FilePath:  filePath,
			StartLine: startLine,
			EndLine:   endLine,
			DiffText:  m.diffView.selectedDiffText(),
		}
		return m, tea.Quit
	}

	// Close diff viewer
	if key == "esc" && m.diffView.mode == diffViewBrowse {
		m.showDiffView = false
		m.updateStatusBar()
		return m, nil
	}
	if key == "q" && (m.diffView.mode == diffViewBrowse || m.diffView.mode == diffViewSelecting) {
		m.quitting = true
		return m, tea.Quit
	}

	// Open in browser
	if key == "o" && m.diffView.mode == diffViewBrowse {
		pr := m.findDiffViewPR()
		if pr != nil && pr.URL != "" {
			return m, openBrowser(pr.URL)
		}
		return m, nil
	}

	// Delegate to diffView
	m.diffView.handleKey(key)
	return m, nil
}

// findDiffViewPR locates the StandalonePR matching the current diff viewer context.
func (m Model) findDiffViewPR() *service.StandalonePR {
	for i := range m.myPRs {
		if m.myPRs[i].Number == m.diffView.prNumber && m.myPRs[i].RepoDir == m.diffView.repoDir {
			return &m.myPRs[i]
		}
	}
	for i := range m.otherPRs {
		if m.otherPRs[i].Number == m.diffView.prNumber && m.otherPRs[i].RepoDir == m.diffView.repoDir {
			return &m.otherPRs[i]
		}
	}
	return nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "y", "Y":
		taskName := m.confirmTask
		action := m.confirmAction
		m.confirming = false
		m.confirmTask = ""
		m.confirmAction = ""

		if action == "testPR" {
			pr := m.confirmPR
			m.confirmPR = nil
			m.statusBar.message = fmt.Sprintf("Switching %s to %s...", pr.RepoAlias, pr.HeadBranch)
			m.statusBar.loading = true
			return m, tea.Batch(m.statusBar.spinner.Tick, func() tea.Msg {
				err := m.testStandalonePR(pr)
				if err != nil {
					return actionResultMsg{message: fmt.Sprintf("Error: %s", err), isError: true}
				}
				return actionResultMsg{message: fmt.Sprintf("Switched %s to %s", pr.RepoAlias, pr.HeadBranch)}
			})
		}

		if action == "test" {
			m.statusBar.message = fmt.Sprintf("Switching repos for %s...", taskName)
			m.statusBar.loading = true
			return m, tea.Batch(m.statusBar.spinner.Tick, func() tea.Msg {
				err := m.testTask(taskName)
				if err != nil {
					return actionResultMsg{message: fmt.Sprintf("Error: %s", err), isError: true}
				}
				return actionResultMsg{message: fmt.Sprintf("Switched local repos for %s", taskName)}
			})
		}

		m.statusBar.message = fmt.Sprintf("Cleaning %s...", taskName)
		m.statusBar.loading = true
		return m, tea.Batch(m.statusBar.spinner.Tick, func() tea.Msg {
			err := m.cleanTask(taskName)
			if err != nil {
				return actionResultMsg{message: fmt.Sprintf("Error: %s", err), isError: true}
			}
			return actionResultMsg{message: fmt.Sprintf("Cleaned %s", taskName)}
		})

	case "n", "N", "esc":
		m.confirming = false
		m.confirmTask = ""
		m.confirmAction = ""
		m.confirmPR = nil
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
	return m.startResumeFlow(sel)
}

// startResumeFlow either launches the resume confirm overlay (when settings
// require asking) or proceeds straight to resumeTask. handleAttach short-circuits
// when a session is already active — no Claude is spawned in that case, so no
// confirm is needed.
func (m Model) startResumeFlow(sel *service.TaskView) (tea.Model, tea.Cmd) {
	if sel.HasSession {
		return m.handleAttach()
	}

	prompt, value := settings.ResolveDangerouslySkip(m.svc.Settings)
	if !prompt {
		m.resumeDangerouslySkip = value
		return m.resumeTask(sel)
	}

	m.resumeDangerouslySkip = false
	m.resumeConfirmTask = sel
	m.resumeConfirmForm = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(settings.DangerouslySkipPromptTitle).
				Description(settings.DangerouslySkipPromptDescription).
				Affirmative("Yes").
				Negative("No").
				Value(&m.resumeDangerouslySkip),
		),
	).WithTheme(ui.HuhTheme()).
		WithWidth(confirmFormWidth(m.width)).
		WithShowHelp(true)
	m.showResumeConfirm = true
	m.updateStatusBar()
	return m, m.resumeConfirmForm.Init()
}

func (m Model) resumeTask(sel *service.TaskView) (tea.Model, tea.Cmd) {
	// If session is already active, focus the existing window
	if sel.HasSession {
		return m.handleAttach()
	}

	// Build the claude command to run in a new tab
	dirs := sel.Dirs()
	if len(dirs) == 0 {
		return m, nil
	}

	// Prepare claude files
	cfg := claude.LaunchConfig{
		Workspace:                  m.svc.Workspace,
		TaskName:                   sel.Name,
		Dirs:                       dirs,
		DangerouslySkipPermissions: m.resumeDangerouslySkip,
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
	// Single repo: launch in the repo's worktree directory.
	// Multiple repos: launch in the workspace root.
	launchDir := m.svc.Workspace.Root
	if len(dirs) == 1 {
		launchDir = dirs[0]
	}
	cmdParts = append(cmdParts, fmt.Sprintf("cd %q", launchDir))

	args := []string{claudePath}
	for _, d := range dirs {
		args = append(args, "--add-dir", d)
	}
	args = append(args, "--append-system-prompt", fmt.Sprintf("%q", systemPrompt))
	if cfg.DangerouslySkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	cmdParts = append(cmdParts, strings.Join(args, " "))

	command := strings.Join(cmdParts, " && ")

	// Wrap command with shell PID tracking
	if m.svc.Tracker != nil {
		command = m.svc.Tracker.WrapCommand(sel.Name, command)
	}

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

func (m Model) findTaskForPR(pr *service.StandalonePR) *service.TaskView {
	for i, task := range m.taskList.tasks {
		for _, wt := range task.Worktrees {
			if wt.Alias == pr.RepoAlias && wt.Branch == pr.HeadBranch {
				return &m.taskList.tasks[i]
			}
		}
	}
	return nil
}

func (m Model) handleResumeFromPR() (tea.Model, tea.Cmd) {
	pr := m.taskList.selectedStandalonePR()
	if pr == nil {
		return m, nil
	}

	// Case 1: matching task exists — resume it directly (with confirm overlay if needed)
	if task := m.findTaskForPR(pr); task != nil {
		return m.startResumeFlow(task)
	}

	// Case 2: no match — launch resume-from-PR wizard
	return m.startResumeFromPR(pr)
}

func (m Model) startResumeFromPR(pr *service.StandalonePR) (tea.Model, tea.Cmd) {
	m.newTaskView = newResumeFromPRModel(m.svc.Workspace, m.svc.Settings, pr)
	m.newTaskView.width = m.width
	m.newTaskView.height = m.height
	m.showNewTask = true
	m.updateStatusBar()
	cmd := m.newTaskView.initPickRepos()
	return m, cmd
}

func (m Model) handleDiff() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil {
		return m, nil
	}

	// If at repo level, diff the focused worktree
	if wt := m.taskList.focusedWorktree(); wt != nil {
		return m, loadDiff(sel.Name, wt.Dir)
	}

	// Otherwise diff the first worktree
	if len(sel.Worktrees) > 0 {
		dir := sel.Worktrees[0].Dir
		return m, loadDiff(sel.Name, dir)
	}

	return m, nil
}

func (m Model) handlePRReview() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil || !sel.HasPRs {
		return m, nil
	}

	// Find first worktree with a PR
	for _, wt := range sel.Worktrees {
		if wt.PR != nil && wt.PR.Number > 0 {
			m.statusBar.message = fmt.Sprintf("Loading PR #%d diff...", wt.PR.Number)
			m.statusBar.loading = true
			headSHA := "" // will be fetched in the diff loader if needed
			return m, tea.Batch(m.statusBar.spinner.Tick, loadPRDiff(wt.Dir, wt.PR.Number, wt.Alias, wt.PR.Title, headSHA, true))
		}
	}

	return m, nil
}

func (m Model) handleStandalonePRDiff() (tea.Model, tea.Cmd) {
	pr := m.taskList.selectedStandalonePR()
	if pr == nil {
		return m, nil
	}

	m.statusBar.message = fmt.Sprintf("Loading PR #%d diff...", pr.Number)
	m.statusBar.loading = true
	return m, tea.Batch(m.statusBar.spinner.Tick, loadPRDiff(pr.RepoDir, pr.Number, pr.RepoAlias, pr.Title, pr.HeadSHA, pr.IsMine))
}

func (m Model) handleStandalonePRComments() (tea.Model, tea.Cmd) {
	pr := m.taskList.selectedStandalonePR()
	if pr == nil {
		return m, nil
	}

	m.statusBar.message = fmt.Sprintf("Loading comments for PR #%d...", pr.Number)
	m.statusBar.loading = true
	taskName := fmt.Sprintf("pr-%d", pr.Number) // synthetic task name for comment viewer
	return m, tea.Batch(m.statusBar.spinner.Tick, loadComments(taskName, pr.RepoAlias, pr.RepoDir, pr.Number))
}

func (m Model) handleStandalonePRBrowserOpen() (tea.Model, tea.Cmd) {
	pr := m.taskList.selectedStandalonePR()
	if pr == nil || pr.URL == "" {
		return m, nil
	}

	m.statusBar.message = fmt.Sprintf("Opened PR #%d in browser", pr.Number)
	return m, tea.Batch(
		openBrowser(pr.URL),
		clearMessageCmd(),
	)
}

func (m Model) handleTestStandalonePR() (tea.Model, tea.Cmd) {
	pr := m.taskList.selectedStandalonePR()
	if pr == nil {
		return m, nil
	}
	if worktree.IsDirty(pr.RepoDir) {
		m.statusBar.message = fmt.Sprintf("Cannot switch: %s local repo has uncommitted changes", pr.RepoAlias)
		return m, clearMessageCmd()
	}
	m.confirming = true
	m.confirmTask = ""
	m.confirmPR = pr
	m.confirmAction = "testPR"
	m.statusBar.message = fmt.Sprintf("Test %s #%d? Will checkout %s locally. (y/n)", pr.RepoAlias, pr.Number, pr.HeadBranch)
	return m, nil
}

func (m Model) testStandalonePR(pr *service.StandalonePR) error {
	if err := worktree.Fetch(pr.RepoDir, pr.HeadBranch); err != nil {
		return fmt.Errorf("%s: fetch failed: %w", pr.RepoAlias, err)
	}
	if err := worktree.Checkout(pr.RepoDir, pr.HeadBranch); err != nil {
		return fmt.Errorf("%s: %w", pr.RepoAlias, err)
	}
	if err := worktree.Pull(pr.RepoDir); err != nil {
		return fmt.Errorf("%s: %w", pr.RepoAlias, err)
	}
	return nil
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
		m.confirmAction = "clean"
		m.statusBar.message = fmt.Sprintf("Clean %s? Has uncommitted changes! (y/n)", sel.Name)
		return m, nil
	}

	m.confirming = true
	m.confirmTask = sel.Name
	m.confirmAction = "clean"
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

func (m Model) handleTest() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil {
		return m, nil
	}

	// Pre-flight: all worktrees must be PUSHED (fresh inspection)
	for _, wt := range sel.Worktrees {
		status := worktree.InspectStatus(wt.Dir)
		if status != worktree.StatusPushed {
			m.statusBar.message = fmt.Sprintf("Cannot switch: %s is %s", wt.Alias, status)
			return m, clearMessageCmd()
		}
	}

	// Pre-flight: local repos must be clean
	ws := m.svc.Workspace
	for _, wt := range sel.Worktrees {
		repo := ws.RepoByAlias(wt.Alias)
		if repo == nil {
			continue
		}
		if worktree.IsDirty(repo.Path) {
			m.statusBar.message = fmt.Sprintf("Cannot switch: %s local repo has uncommitted changes", wt.Alias)
			return m, clearMessageCmd()
		}
	}

	m.confirming = true
	m.confirmTask = sel.Name
	m.confirmAction = "test"
	m.statusBar.message = fmt.Sprintf("Test %s? Will switch local repos and remove worktrees. (y/n)", sel.Name)
	return m, nil
}

func (m Model) testTask(taskName string) error {
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

		// Fetch latest
		if err := worktree.Fetch(repo.Path, wt.Branch); err != nil {
			return fmt.Errorf("%s: fetch failed: %w", wt.Alias, err)
		}

		// Remove worktree but keep the branch
		if err := worktree.Remove(repo.Path, wt.Dir, false); err != nil {
			return fmt.Errorf("%s: remove worktree failed: %w", wt.Alias, err)
		}

		// Checkout branch in local repo
		if err := worktree.Checkout(repo.Path, wt.Branch); err != nil {
			return fmt.Errorf("%s: %w", wt.Alias, err)
		}

		// Pull latest
		if err := worktree.Pull(repo.Path); err != nil {
			return fmt.Errorf("%s: %w", wt.Alias, err)
		}
	}

	// Cleanup task directory
	worktree.CleanupTaskDir(ws.Root, taskName)

	// Unregister session if tracked
	if m.svc.Tracker != nil {
		_ = m.svc.Tracker.Unregister(taskName)
	}

	return nil
}

func (m Model) handleOpenPR() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil {
		return m, nil
	}

	if !m.svc.GHAvailable {
		m.statusBar.message = "gh CLI not configured — install gh and run: gh auth login"
		return m, clearMessageCmd()
	}

	// At repo level, only open PR for the focused worktree
	if wt := m.taskList.focusedWorktree(); wt != nil {
		m.openPRWorktreeAlias = wt.Alias
	}

	m.openPR = true
	return m, tea.Quit
}

func (m Model) handleBrowserOpen() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil || !sel.HasPRs {
		return m, nil
	}

	// Find first PR URL for the selected task
	for _, wt := range sel.Worktrees {
		if wt.PR != nil && wt.PR.URL != "" {
			// Mark as viewed
			if m.svc.PRStore != nil {
				_ = m.svc.PRStore.MarkViewed(sel.Name, wt.Alias, wt.PR.CommentCount)
			}
			m.statusBar.message = fmt.Sprintf("Opened PR #%d in browser", wt.PR.Number)
			return m, tea.Batch(
				openBrowser(wt.PR.URL),
				clearMessageCmd(),
			)
		}
	}

	m.statusBar.message = "No PR found for this task"
	return m, clearMessageCmd()
}

func (m Model) handleComments() (tea.Model, tea.Cmd) {
	sel := m.taskList.selected()
	if sel == nil || !sel.HasPRs {
		return m, nil
	}

	// Collect worktrees with PRs
	type prEntry struct {
		alias    string
		dir      string
		prNumber int
	}
	var prs []prEntry
	for _, wt := range sel.Worktrees {
		if wt.PR != nil && wt.PR.Number > 0 {
			prs = append(prs, prEntry{alias: wt.Alias, dir: wt.Dir, prNumber: wt.PR.Number})
		}
	}

	if len(prs) == 0 {
		return m, nil
	}

	// Auto-select if only one PR
	entry := prs[0]
	m.statusBar.message = fmt.Sprintf("Loading comments for PR #%d...", entry.prNumber)
	m.statusBar.loading = true
	return m, tea.Batch(m.statusBar.spinner.Tick, loadComments(sel.Name, entry.alias, entry.dir, entry.prNumber))
}

func (m Model) handleCommentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Claude prompt editing mode
	if m.comments.mode == commentModeClaudePrompt {
		switch key {
		case "esc":
			m.comments.cancelClaudePrompt()
			return m, nil
		case "enter":
			if t := m.comments.currentThread(); t != nil {
				m.comments.claudeRequested = true
				m.comments.claudeThread = t
				return m, tea.Quit
			}
			return m, nil
		case "backspace":
			if len(m.comments.claudePromptBuf) > 0 {
				m.comments.claudePromptBuf = m.comments.claudePromptBuf[:len(m.comments.claudePromptBuf)-1]
			}
			return m, nil
		default:
			if len(key) == 1 {
				m.comments.claudePromptBuf = append(m.comments.claudePromptBuf, rune(key[0]))
			} else if key == "space" {
				m.comments.claudePromptBuf = append(m.comments.claudePromptBuf, ' ')
			}
			return m, nil
		}
	}

	// Reply mode input
	if m.comments.mode == commentModeReply {
		switch key {
		case "esc":
			m.comments.cancelReply()
			return m, nil
		case "enter":
			body := strings.TrimSpace(string(m.comments.replyBuf))
			if body == "" {
				m.comments.cancelReply()
				return m, nil
			}
			m.comments.mode = commentModeBrowse
			m.statusBar.message = "Posting reply..."
			m.statusBar.loading = true
			if m.comments.isOnThread() {
				commentID := m.comments.lastCommentID()
				return m, tea.Batch(m.statusBar.spinner.Tick, replyToComment(m.comments.worktreeDir, m.comments.prNumber, commentID, body))
			}
			return m, tea.Batch(m.statusBar.spinner.Tick, replyToIssueComment(m.comments.worktreeDir, m.comments.prNumber, body))
		case "backspace":
			if len(m.comments.replyBuf) > 0 {
				m.comments.replyBuf = m.comments.replyBuf[:len(m.comments.replyBuf)-1]
			}
			return m, nil
		default:
			// Append printable characters
			if len(key) == 1 {
				m.comments.replyBuf = append(m.comments.replyBuf, rune(key[0]))
			} else if key == "space" {
				m.comments.replyBuf = append(m.comments.replyBuf, ' ')
			}
			return m, nil
		}
	}

	// Browse mode
	switch key {
	case "n", "down":
		m.comments.next()
		return m, nil
	case "p", "up":
		m.comments.prev()
		return m, nil
	case "J":
		m.comments.scrollDown()
		return m, nil
	case "K":
		m.comments.scrollUp()
		return m, nil
	case "R":
		m.comments.startReply()
		return m, nil
	case "C":
		if m.comments.currentThread() != nil {
			m.comments.startClaudePrompt()
			return m, nil
		}
		return m, nil
	case "o":
		// Open PR in browser
		sel := m.taskList.selected()
		if sel != nil {
			for _, wt := range sel.Worktrees {
				if wt.PR != nil && wt.PR.URL != "" {
					return m, openBrowser(wt.PR.URL)
				}
			}
		}
		// Also check standalone PRs
		pr := m.taskList.selectedStandalonePR()
		if pr != nil && pr.URL != "" {
			return m, openBrowser(pr.URL)
		}
		return m, nil
	case "esc", "m":
		m.showComments = false
		m.updateStatusBar()
		return m, nil
	case "q":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// CommentClaudeRequested returns true if the user pressed 'C' to hand off a thread to Claude.
func (m Model) CommentClaudeRequested() bool {
	return m.comments.claudeRequested
}

// CommentClaudeContext returns the context needed to launch Claude with the review comment.
func (m Model) CommentClaudeContext() *claude.CommentContext {
	if !m.comments.claudeRequested || m.comments.claudeThread == nil {
		return nil
	}
	t := m.comments.claudeThread
	return &claude.CommentContext{
		PRNumber:       m.comments.prNumber,
		FilePath:       t.Path,
		Line:           t.Line,
		DiffHunk:       t.DiffHunk,
		ThreadBody:     FormatThreadBody(t),
		WorktreeDir:    m.comments.worktreeDir,
		UserPrompt:     m.comments.claudeUserPrompt(),
	}
}

// CommentTaskName returns the task name from the comment viewer.
func (m Model) CommentTaskName() string {
	return m.comments.taskName
}

// CommentWorktreeDir returns the worktree directory from the comment viewer.
func (m Model) CommentWorktreeDir() string {
	return m.comments.worktreeDir
}

// DiffViewClaudeRequested returns true if the user pressed 'C' in the diff viewer to launch Claude.
func (m Model) DiffViewClaudeRequested() bool {
	return m.diffView.claudeRequested
}

// DiffViewClaudeContext returns the review context for Claude launch from the diff viewer.
func (m Model) DiffViewClaudeContext() *claude.ReviewContext {
	if !m.diffView.claudeRequested || m.diffView.claudeContext == nil {
		return nil
	}
	ctx := m.diffView.claudeContext
	return &claude.ReviewContext{
		PRNumber:   m.diffView.prNumber,
		PRTitle:    m.diffView.prTitle,
		RepoAlias:  m.diffView.repoAlias,
		RepoDir:    m.diffView.repoDir,
		FilePath:   ctx.FilePath,
		StartLine:  ctx.StartLine,
		EndLine:    ctx.EndLine,
		DiffLines:  ctx.DiffText,
		UserPrompt: m.diffView.claudeUserPrompt(),
	}
}

// DiffViewIsMine returns whether the diff viewer PR belongs to the current user.
func (m Model) DiffViewIsMine() bool {
	return m.diffView.isMine
}

// DiffViewRepoDir returns the repo directory from the diff viewer.
func (m Model) DiffViewRepoDir() string {
	return m.diffView.repoDir
}

// OpenPRRequested returns true if the user pressed 'p' to open the PR wizard.
func (m Model) OpenPRRequested() bool {
	return m.openPR
}

// SelectedTaskName returns the name of the currently selected task.
func (m Model) SelectedTaskName() string {
	if sel := m.taskList.selected(); sel != nil {
		return sel.Name
	}
	return ""
}

// SelectedWorktreeAlias returns the worktree alias to scope PR creation to,
// or "" if all worktrees should be included.
func (m Model) SelectedWorktreeAlias() string {
	return m.openPRWorktreeAlias
}

// Layout and rendering

func (m *Model) updateLayout() {
	// Distribute width: 50% task list, 50% detail
	listWidth := m.width * 50 / 100
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
	row := m.taskList.selectedRow()
	if row == nil {
		m.detail.setTask(nil)
		return
	}

	switch row.kind {
	case rowTask, rowWorktree:
		m.detail.setTask(m.taskList.selected())
	case rowRepoHeader:
		m.detail.setRepoGroup(row.repoAlias, row.section, m.myPRs, m.otherPRs)
	case rowMyPR, rowOtherPR:
		m.detail.setStandalonePR(m.taskList.selectedStandalonePR())
	default:
		m.detail.setTask(nil)
	}
}

func (m *Model) updateStatusBar() {
	sel := m.taskList.selected()
	row := m.taskList.selectedRow()

	m.statusBar.hasTask = sel != nil
	m.statusBar.hasActive = sel != nil && sel.HasSession
	m.statusBar.showDiff = m.detail.showDiff
	m.statusBar.showComments = m.showComments
	m.statusBar.showDiffView = m.showDiffView
	m.statusBar.showNewTask = m.showNewTask
	m.statusBar.showResumeConfirm = m.showResumeConfirm
	m.statusBar.showSettings = m.showSettings
	m.statusBar.hasPR = sel != nil && sel.HasPRs
	m.statusBar.hasComments = sel != nil && m.hasComments(sel)
	m.statusBar.ghAvailable = m.svc.GHAvailable

	// Check if cursor is on a standalone PR or repo header
	m.statusBar.standalonePR = row != nil && (row.kind == rowMyPR || row.kind == rowOtherPR)
	m.statusBar.isMyPR = row != nil && row.kind == rowMyPR
	m.statusBar.repoHeader = row != nil && row.kind == rowRepoHeader
	m.statusBar.inRepoLevel = m.taskList.navLevel == navRepo
	if m.showDiffView {
		m.statusBar.diffViewMode = m.diffView.mode
	}
}

func (m *Model) hasComments(sel *service.TaskView) bool {
	for _, wt := range sel.Worktrees {
		if wt.PR != nil && wt.PR.CommentCount > 0 {
			return true
		}
	}
	return false
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Loading..."
	}

	// Fullscreen new task overlay
	if m.showNewTask {
		return m.newTaskView.view()
	}

	// Fullscreen resume confirm overlay
	if m.showResumeConfirm {
		return m.resumeConfirmView()
	}

	// Fullscreen settings overlay
	if m.showSettings && m.settingsView != nil {
		return m.settingsView.view()
	}

	// Fullscreen diff viewer overlay
	if m.showDiffView {
		return m.diffView.view()
	}

	// Fullscreen comment viewer overlay
	if m.showComments {
		return m.comments.view()
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
	header := headerLine(m.taskList.tasks, m.width)

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

	header := headerLine(m.taskList.tasks, m.width)
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

// startSettings opens the settings overlay.
func (m Model) startSettings() (tea.Model, tea.Cmd) {
	m.settingsView = newSettingsModel(m.svc.Settings)
	m.settingsView.width = m.width
	m.settingsView.height = m.height
	m.showSettings = true
	m.updateStatusBar()
	if m.settingsView.form != nil {
		return m, m.settingsView.form.Init()
	}
	return m, nil
}

// handleSettingsKey forwards keys to the settings overlay form.
func (m Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.settingsView == nil {
		m.showSettings = false
		m.updateStatusBar()
		return m, nil
	}
	cmd := m.settingsView.update(msg)
	return m, cmd
}

// confirmFormWidth picks a sensible width for inline confirm/select forms in
// the dashboard. Mirrors the clamping used by the new-task wizard.
func confirmFormWidth(termWidth int) int {
	w := termWidth - 8
	if w < 40 {
		w = 40
	}
	if w > 80 {
		w = 80
	}
	return w
}

// resumeConfirmView renders the resume-with-skip-permissions confirm overlay.
func (m Model) resumeConfirmView() string {
	titleStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ui.ColorMuted)

	var b strings.Builder
	taskName := ""
	if m.resumeConfirmTask != nil {
		taskName = m.resumeConfirmTask.Name
	}
	b.WriteString(titleStyle.Render("Resume " + taskName))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width, 60))))
	b.WriteString("\n\n")
	if m.resumeConfirmForm != nil {
		b.WriteString(m.resumeConfirmForm.View())
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2, 4).
		Render(b.String())
}

// startNewTask opens the new task wizard overlay.
func (m Model) startNewTask() (tea.Model, tea.Cmd) {
	m.newTaskView = newNewTaskModel(m.svc.Workspace, m.svc.Settings)
	m.newTaskView.width = m.width
	m.newTaskView.height = m.height
	m.showNewTask = true
	m.updateStatusBar()
	cmd := m.newTaskView.initPickRepos()
	return m, cmd
}

// handleNewTaskKey forwards key events to the active new task overlay form.
func (m Model) handleNewTaskKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Allow Esc at the done-with-error step to close overlay
	if m.newTaskView.step == stepDone && msg.String() == "esc" {
		m.showNewTask = false
		m.updateStatusBar()
		return m, nil
	}

	cmd := m.newTaskView.update(msg)
	return m, cmd
}

// launchNewTask spawns Claude in a new terminal tab after worktree creation.
func (m Model) launchNewTask() (tea.Model, tea.Cmd) {
	dirs := m.newTaskView.createdDirs
	taskName := m.newTaskView.taskName

	// Prepare claude files
	cfg := claude.LaunchConfig{
		Workspace:                  m.svc.Workspace,
		TaskName:                   taskName,
		Dirs:                       dirs,
		DangerouslySkipPermissions: m.newTaskView.dangerouslySkip,
	}

	if err := claude.Prepare(cfg); err != nil {
		m.statusBar.message = fmt.Sprintf("Error preparing: %s", err)
		m.showNewTask = false
		m.updateStatusBar()
		return m, clearMessageCmd()
	}

	// Build command string
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		m.statusBar.message = "claude not found in PATH"
		m.showNewTask = false
		m.updateStatusBar()
		return m, clearMessageCmd()
	}

	systemPrompt := claude.BuildSystemPrompt(cfg)
	var cmdParts []string
	// Single repo: launch in the repo's worktree directory.
	// Multiple repos: launch in the workspace root.
	launchDir := m.svc.Workspace.Root
	if len(dirs) == 1 {
		launchDir = dirs[0]
	}
	cmdParts = append(cmdParts, fmt.Sprintf("cd %q", launchDir))

	args := []string{claudePath}
	for _, d := range dirs {
		args = append(args, "--add-dir", d)
	}
	args = append(args, "--append-system-prompt", fmt.Sprintf("%q", systemPrompt))
	if cfg.DangerouslySkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}
	cmdParts = append(cmdParts, strings.Join(args, " "))

	command := strings.Join(cmdParts, " && ")

	// Wrap command with shell PID tracking
	if m.svc.Tracker != nil {
		command = m.svc.Tracker.WrapCommand(taskName, command)
	}

	tabTitle := "work: " + taskName

	// Spawn in new terminal tab
	opener := session.DetectTerminal()
	pid, err := opener.OpenTab(command, tabTitle)
	if err != nil {
		m.statusBar.message = fmt.Sprintf("Tab spawn failed: %s", err)
		m.showNewTask = false
		m.updateStatusBar()
		return m, clearMessageCmd()
	}

	// Register session
	if m.svc.Tracker != nil {
		rec := session.SessionRecord{
			TaskName:      taskName,
			PID:           pid,
			Dirs:          dirs,
			LaunchedAt:    time.Now(),
			TerminalTab:   tabTitle,
			WorkspaceRoot: m.svc.Workspace.Root,
		}
		_ = m.svc.Tracker.Register(rec)
	}

	m.showNewTask = false
	m.statusBar.message = fmt.Sprintf("Launched %s in new tab", taskName)
	m.updateStatusBar()
	return m, tea.Batch(
		loadTasks(m.svc),
		clearMessageCmd(),
	)
}

// NewTaskRequested returns true if the user pressed 'n' to start a new task.
// Kept for backward compatibility but should no longer trigger.
func (m Model) NewTaskRequested() bool {
	return m.newTask
}

// openURL opens a URL in the default browser.
func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		cmd = exec.Command("open", url)
	}
	_ = cmd.Start()
}
