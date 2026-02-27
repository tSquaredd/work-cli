package dashboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
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

	// Standalone PRs
	myPRs    []service.StandalonePR
	otherPRs []service.StandalonePR

	// State
	confirming  bool
	confirmTask string
	quitting    bool
	newTask     bool // set when user presses 'n' to start a new task
	openPR               bool   // set when user presses 'p' to open PR wizard
	openPRWorktreeAlias  string // when at repo level, only open PR for this worktree
	prTick      int  // counter for throttled PR polling (every 6th tick = 30s)
	prDiscovered bool // whether initial PR discovery has been done
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
		if m.svc.GHAvailable {
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
			m.statusBar.message = fmt.Sprintf("PR list error: %s", msg.err)
			return m, clearMessageCmd()
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
		// Refresh standalone PRs every 12th tick (60s)
		if m.svc.GHAvailable && m.prTick%12 == 0 {
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

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

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

	// Check if cursor is on a standalone PR
	row := m.taskList.selectedRow()
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

	case "a":
		return m.handleAttach()

	case "n":
		m.newTask = true
		return m, tea.Quit

	case "p":
		return m.handleOpenPR()

	case "o":
		return m.handleBrowserOpen()

	case "m":
		return m.handleComments()
	}

	return m, nil
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

	case "d":
		return m.handleStandalonePRDiff()

	case "m":
		return m.handleStandalonePRComments()

	case "o":
		return m.handleStandalonePRBrowserOpen()

	case "n":
		m.newTask = true
		return m, tea.Quit
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
		m.confirming = false
		m.confirmTask = ""
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
	row := m.taskList.selectedRow()
	if row == nil {
		m.detail.setTask(nil)
		return
	}

	switch row.kind {
	case rowTask, rowWorktree:
		m.detail.setTask(m.taskList.selected())
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
	m.statusBar.hasPR = sel != nil && sel.HasPRs
	m.statusBar.hasComments = sel != nil && m.hasComments(sel)
	m.statusBar.ghAvailable = m.svc.GHAvailable

	// Check if cursor is on a standalone PR
	m.statusBar.standalonePR = row != nil && (row.kind == rowMyPR || row.kind == rowOtherPR)
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

// NewTaskRequested returns true if the user pressed 'n' to start a new task.
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
