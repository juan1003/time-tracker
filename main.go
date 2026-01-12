package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const historyFile = "history.txt"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	timerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	historyItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type view int

const (
	menuView view = iota
	trackingView
	historyView
	settingsView
)

type tickMsg time.Time

type session struct {
	start    time.Time
	end      time.Time
	duration time.Duration
}

type model struct {
	currentView    view
	cursor         int
	menuItems      []string
	tracking       bool
	trackingStart  time.Time
	elapsed        time.Duration
	history        []session
	settingsCursor int
	settings       map[string]bool
}

func initialModel() model {
	return model{
		currentView: menuView,
		menuItems: []string{
			"Start tracking",
			"Stop tracking",
			"View history",
			"Settings",
			"Quit",
		},
		history: loadHistory(),
		settings: map[string]bool{
			"Show seconds":    true,
			"Auto-save":       true,
			"Notifications":   false,
			"Dark mode":       true,
		},
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.tracking {
			m.elapsed = time.Since(m.trackingStart)
			return m, tickCmd()
		}

	case tea.KeyMsg:
		switch m.currentView {
		case menuView:
			return m.updateMenu(msg)
		case trackingView:
			return m.updateTracking(msg)
		case historyView:
			return m.updateHistory(msg)
		case settingsView:
			return m.updateSettings(msg)
		}
	}

	return m, nil
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.menuItems)-1 {
			m.cursor++
		}
	case "enter":
		switch m.cursor {
		case 0: // Start tracking
			if !m.tracking {
				m.tracking = true
				m.trackingStart = time.Now()
				m.elapsed = 0
				m.currentView = trackingView
				return m, tickCmd()
			}
		case 1: // Stop tracking
			if m.tracking {
				m.tracking = false
				m.history = append(m.history, session{
					start:    m.trackingStart,
					end:      time.Now(),
					duration: m.elapsed,
				})
				m.elapsed = 0
				saveHistory(m.history)
			}
		case 2: // View history
			m.currentView = historyView
			m.cursor = 0
		case 3: // Settings
			m.currentView = settingsView
			m.settingsCursor = 0
		case 4: // Quit
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) updateTracking(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "b":
		m.currentView = menuView
		return m, nil
	case "enter", "s":
		if m.tracking {
			m.tracking = false
			m.history = append(m.history, session{
				start:    m.trackingStart,
				end:      time.Now(),
				duration: m.elapsed,
			})
			m.elapsed = 0
			m.currentView = menuView
			saveHistory(m.history)
		}
		return m, nil
	}
	return m, tickCmd()
}

func (m model) updateHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "b":
		m.currentView = menuView
		m.cursor = 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.history)-1 {
			m.cursor++
		}
	case "d", "backspace":
		if len(m.history) > 0 && m.cursor < len(m.history) {
			m.history = append(m.history[:m.cursor], m.history[m.cursor+1:]...)
			if m.cursor >= len(m.history) && m.cursor > 0 {
				m.cursor--
			}
			saveHistory(m.history)
		}
	}
	return m, nil
}

func (m model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	settingsKeys := m.getSettingsKeys()

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "b":
		m.currentView = menuView
		m.cursor = 0
	case "up", "k":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
	case "down", "j":
		if m.settingsCursor < len(settingsKeys)-1 {
			m.settingsCursor++
		}
	case "enter", " ":
		key := settingsKeys[m.settingsCursor]
		m.settings[key] = !m.settings[key]
	}
	return m, nil
}

func (m model) getSettingsKeys() []string {
	return []string{"Show seconds", "Auto-save", "Notifications", "Dark mode"}
}

func (m model) View() string {
	switch m.currentView {
	case trackingView:
		return m.viewTracking()
	case historyView:
		return m.viewHistory()
	case settingsView:
		return m.viewSettings()
	default:
		return m.viewMenu()
	}
}

func (m model) viewMenu() string {
	s := titleStyle.Render("⏱  Time Tracking") + "\n\n"

	if m.tracking {
		s += timerStyle.Render(fmt.Sprintf("● Recording: %s", formatDuration(m.elapsed))) + "\n\n"
	}

	for i, item := range m.menuItems {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		line := cursor + item
		if m.cursor == i {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += normalStyle.Render(line) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/↓: navigate • enter: select • q: quit")

	return s
}

func (m model) viewTracking() string {
	s := titleStyle.Render("⏱  Tracking Time") + "\n\n"

	s += timerStyle.Render(fmt.Sprintf("  %s  ", formatDuration(m.elapsed))) + "\n\n"

	s += normalStyle.Render(fmt.Sprintf("Started: %s", m.trackingStart.Format("15:04:05"))) + "\n\n"

	s += selectedStyle.Render("> Stop and save") + "\n"
	s += normalStyle.Render("  Press enter/s to stop, esc/b to go back (keeps running)") + "\n"

	s += "\n" + helpStyle.Render("enter/s: stop • esc/b: back • q: quit")

	return s
}

func (m model) viewHistory() string {
	s := titleStyle.Render("📋 History") + "\n\n"

	if len(m.history) == 0 {
		s += normalStyle.Render("No tracking sessions yet.") + "\n"
	} else {
		for i, sess := range m.history {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}

			line := fmt.Sprintf("%s%s - %s (%s)",
				cursor,
				sess.start.Format("Jan 02 15:04"),
				sess.end.Format("15:04"),
				formatDuration(sess.duration),
			)

			if m.cursor == i {
				s += selectedStyle.Render(line) + "\n"
			} else {
				s += historyItemStyle.Render(line) + "\n"
			}
		}
	}

	s += "\n" + helpStyle.Render("↑/↓: navigate • d: delete • esc/b: back • q: quit")

	return s
}

func (m model) viewSettings() string {
	s := titleStyle.Render("⚙  Settings") + "\n\n"

	settingsKeys := m.getSettingsKeys()

	for i, key := range settingsKeys {
		cursor := "  "
		if m.settingsCursor == i {
			cursor = "> "
		}

		checked := "○"
		if m.settings[key] {
			checked = "●"
		}

		line := fmt.Sprintf("%s[%s] %s", cursor, checked, key)
		if m.settingsCursor == i {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += normalStyle.Render(line) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/↓: navigate • enter/space: toggle • esc/b: back • q: quit")

	return s
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func saveHistory(history []session) error {
	var totalDuration time.Duration
	for _, s := range history {
		totalDuration += s.duration
	}

	var sb strings.Builder

	sb.WriteString(`
 ╔════════════════════════════════════════════════════════════════╗
 ║                                                                ║
 ║    ████████╗██╗███╗   ███╗███████╗                             ║
 ║    ╚══██╔══╝██║████╗ ████║██╔════╝                             ║
 ║       ██║   ██║██╔████╔██║█████╗                               ║
 ║       ██║   ██║██║╚██╔╝██║██╔══╝                               ║
 ║       ██║   ██║██║ ╚═╝ ██║███████╗                             ║
 ║       ╚═╝   ╚═╝╚═╝     ╚═╝╚══════╝                             ║
 ║                                                                ║
 ║    ████████╗██████╗  █████╗  ██████╗██╗  ██╗███████╗██████╗    ║
 ║    ╚══██╔══╝██╔══██╗██╔══██╗██╔════╝██║ ██╔╝██╔════╝██╔══██╗   ║
 ║       ██║   ██████╔╝███████║██║     █████╔╝ █████╗  ██████╔╝   ║
 ║       ██║   ██╔══██╗██╔══██║██║     ██╔═██╗ ██╔══╝  ██╔══██╗   ║
 ║       ██║   ██║  ██║██║  ██║╚██████╗██║  ██╗███████╗██║  ██║   ║
 ║       ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝   ║
 ║                                                                ║
 ╚════════════════════════════════════════════════════════════════╝
`)

	sb.WriteString(fmt.Sprintf("\n  Generated: %s\n", time.Now().Format("Mon Jan 02, 2006 at 03:04 PM")))
	sb.WriteString(fmt.Sprintf("  Total Sessions: %d\n", len(history)))
	sb.WriteString(fmt.Sprintf("  Total Time: %s\n", formatDurationLong(totalDuration)))

	sb.WriteString(`
 ┌────────────────────────────────────────────────────────────────┐
 │                      SESSION HISTORY                           │
 └────────────────────────────────────────────────────────────────┘
`)

	if len(history) == 0 {
		sb.WriteString("\n   No sessions recorded yet.\n")
	} else {
		for i, sess := range history {
			sb.WriteString(fmt.Sprintf(`
   ┌──────────────────────────────────────────┐
   │  SESSION #%-3d                            │
   ├──────────────────────────────────────────┤
   │  Date:     %-29s │
   │  Start:    %-29s │
   │  End:      %-29s │
   │  Duration: %-29s │
   └──────────────────────────────────────────┘
`,
				i+1,
				sess.start.Format("Monday, January 02, 2006"),
				sess.start.Format("03:04:05 PM"),
				sess.end.Format("03:04:05 PM"),
				formatDurationLong(sess.duration),
			))
		}
	}

	sb.WriteString(`
 ╔════════════════════════════════════════════════════════════════╗
 ║                        END OF REPORT                           ║
 ╚════════════════════════════════════════════════════════════════╝
`)

	return os.WriteFile(historyFile, []byte(sb.String()), 0644)
}

func loadHistory() []session {
	file, err := os.Open(historyFile)
	if err != nil {
		return []session{}
	}
	defer file.Close()

	var history []session
	scanner := bufio.NewScanner(file)

	var currentSession *session
	var dateStr, startStr, endStr string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "SESSION #") {
			currentSession = &session{}
			dateStr, startStr, endStr = "", "", ""
		}

		if strings.Contains(line, "Date:") {
			parts := strings.SplitN(line, "Date:", 2)
			if len(parts) == 2 {
				dateStr = strings.TrimSpace(strings.Split(parts[1], "│")[0])
			}
		}

		if strings.Contains(line, "Start:") && !strings.Contains(line, "──") {
			parts := strings.SplitN(line, "Start:", 2)
			if len(parts) == 2 {
				startStr = strings.TrimSpace(strings.Split(parts[1], "│")[0])
			}
		}

		if strings.Contains(line, "End:") {
			parts := strings.SplitN(line, "End:", 2)
			if len(parts) == 2 {
				endStr = strings.TrimSpace(strings.Split(parts[1], "│")[0])
			}
		}

		if strings.Contains(line, "└──") && currentSession != nil && dateStr != "" && startStr != "" && endStr != "" {
			startTime, err1 := time.Parse("Monday, January 02, 2006 03:04:05 PM", dateStr+" "+startStr)
			endTime, err2 := time.Parse("Monday, January 02, 2006 03:04:05 PM", dateStr+" "+endStr)

			if err1 == nil && err2 == nil {
				currentSession.start = startTime
				currentSession.end = endTime
				currentSession.duration = endTime.Sub(startTime)
				history = append(history, *currentSession)
			}
			currentSession = nil
		}
	}

	return history
}

func formatDurationLong(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
