package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var dayNames = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func (m model) initCronEditor() model {
	m.state = stateCronEditor

	// Parse existing cron expression
	minute, hour, dom, month, dow := parseCronFields(m.editTask.At)

	labels := [4]string{"Minute", "Hour", "Day of Month", "Month"}
	values := [4]string{minute, hour, dom, month}

	for i := range 4 {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.CharLimit = 20
		ti.Width = 20
		ti.SetValue(values[i])
		m.cronInputs[i] = ti
	}

	m.cronDays = parseDOW(dow)
	m.cronCursor = 0
	m.cronDaysCursor = 0

	// Focus the first input
	m.cronInputs[0].Focus()
	for i := 1; i < 4; i++ {
		m.cronInputs[i].Blur()
	}

	return m
}

func (m model) cronUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel cron editing
			m.state = stateEditor
			return m, nil

		case "enter":
			// Confirm and apply the cron expression
			expr := buildCronExpr(m.cronInputs, m.cronDays)
			m.editTask.At = expr
			m.editTask.Scheduled = expr != ""
			m.state = stateEditor
			return m, nil

		case "ctrl+x":
			// Clear the schedule
			m.editTask.At = ""
			m.editTask.Scheduled = false
			m.state = stateEditor
			return m, nil

		case "tab", "down":
			return m.cronNextField()

		case "shift+tab", "up":
			return m.cronPrevField()

		// Day of week shortcuts (only when in the days area)
		case "w":
			if m.cronCursor == 4 {
				// Weekdays: Mon-Fri
				m.cronDays = [7]bool{false, true, true, true, true, true, false}
				return m, nil
			}
		case "W":
			if m.cronCursor == 4 {
				// Weekends: Sat, Sun
				m.cronDays = [7]bool{true, false, false, false, false, false, true}
				return m, nil
			}
		case "*":
			if m.cronCursor == 4 {
				// Every day
				m.cronDays = [7]bool{true, true, true, true, true, true, true}
				return m, nil
			}
		case "n":
			if m.cronCursor == 4 {
				// No specific days (use *)
				m.cronDays = [7]bool{false, false, false, false, false, false, false}
				return m, nil
			}

		case "left", "h":
			if m.cronCursor == 4 && m.cronDaysCursor > 0 {
				m.cronDaysCursor--
			}
			return m, nil

		case "right", "l":
			if m.cronCursor == 4 && m.cronDaysCursor < 6 {
				m.cronDaysCursor++
			}
			return m, nil

		case " ":
			if m.cronCursor == 4 {
				m.cronDays[m.cronDaysCursor] = !m.cronDays[m.cronDaysCursor]
				return m, nil
			}
		}
	}

	// Update the focused text input
	if m.cronCursor >= 0 && m.cronCursor < 4 {
		var cmd tea.Cmd
		m.cronInputs[m.cronCursor], cmd = m.cronInputs[m.cronCursor].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) cronNextField() (tea.Model, tea.Cmd) {
	if m.cronCursor < 4 {
		if m.cronCursor >= 0 && m.cronCursor < 4 {
			m.cronInputs[m.cronCursor].Blur()
		}
		m.cronCursor++
		if m.cronCursor < 4 {
			cmd := m.cronInputs[m.cronCursor].Focus()
			return m, cmd
		}
	}
	return m, nil
}

func (m model) cronPrevField() (tea.Model, tea.Cmd) {
	if m.cronCursor > 0 {
		if m.cronCursor >= 0 && m.cronCursor < 4 {
			m.cronInputs[m.cronCursor].Blur()
		}
		m.cronCursor--
		if m.cronCursor < 4 {
			cmd := m.cronInputs[m.cronCursor].Focus()
			return m, cmd
		}
	}
	return m, nil
}

func (m model) cronView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Schedule Editor"))
	b.WriteString("\n")

	labels := [4]string{"Minute:", "Hour:", "Day of Month:", "Month:"}

	for i := range 4 {
		lbl := labels[i]
		if m.cronCursor == i {
			lbl = focusedLabelStyle.Render(lbl)
		} else {
			lbl = labelStyle.Render(lbl)
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", lbl, m.cronInputs[i].View()))
	}

	b.WriteString("\n")

	// Day of week header
	dowLabel := "Day of Week:"
	if m.cronCursor == 4 {
		dowLabel = focusedLabelStyle.Render(dowLabel)
	} else {
		dowLabel = labelStyle.Render(dowLabel)
	}
	b.WriteString(fmt.Sprintf("  %s\n", dowLabel))

	// Day checkboxes
	b.WriteString("                    ")
	for i, name := range dayNames {
		checked := m.cronDays[i]
		var indicator string
		if checked {
			indicator = "●"
		} else {
			indicator = "○"
		}

		label := fmt.Sprintf("%s %s", indicator, name)

		if m.cronCursor == 4 && m.cronDaysCursor == i {
			label = daySelectedStyle.Render(label)
		} else if checked {
			label = dayActiveStyle.Render(label)
		} else {
			label = dayInactiveStyle.Render(label)
		}

		b.WriteString(label)
		if i < 6 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")

	b.WriteString("\n")

	// Preview
	expr := buildCronExpr(m.cronInputs, m.cronDays)
	if expr != "" {
		b.WriteString(fmt.Sprintf("  Cron expression: %s\n", activeStyle.Render(expr)))
		desc := describeCron(expr)
		if desc != "" {
			b.WriteString(fmt.Sprintf("  %s\n", helpStyle.Render(desc)))
		}
	} else {
		b.WriteString(helpStyle.Render("  (no schedule set)"))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Shortcuts
	shortcuts := "Shortcuts (in day-of-week): w weekdays  W weekends  * every day  n none"
	b.WriteString(helpStyle.Render("  " + shortcuts))
	b.WriteString("\n")

	help := "Tab/↑/↓: navigate fields  Space: toggle day  Enter: confirm  Esc: cancel  Ctrl+X: clear schedule"
	b.WriteString(helpStyle.Render("  " + help))

	return b.String()
}

// parseCronFields splits a cron expression into its 5 components.
func parseCronFields(expr string) (minute, hour, dom, month, dow string) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "*", "*", "*", "*", "*"
	}

	parts := strings.Fields(expr)
	if len(parts) >= 1 {
		minute = parts[0]
	}
	if len(parts) >= 2 {
		hour = parts[1]
	}
	if len(parts) >= 3 {
		dom = parts[2]
	}
	if len(parts) >= 4 {
		month = parts[3]
	}
	if len(parts) >= 5 {
		dow = parts[4]
	}

	return
}

// parseDOW parses a day-of-week cron field into boolean flags.
func parseDOW(dow string) [7]bool {
	var days [7]bool

	if dow == "" || dow == "*" {
		// All days or unspecified
		return days // all false means "*" (any day)
	}

	// Handle comma-separated values and ranges
	for part := range strings.SplitSeq(dow, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			// Range like 1-5
			rangeParts := strings.SplitN(part, "-", 2)
			start := parseDayNum(rangeParts[0])
			end := parseDayNum(rangeParts[1])
			if start >= 0 && end >= 0 {
				for i := start; i <= end; i++ {
					if i >= 0 && i < 7 {
						days[i] = true
					}
				}
			}
		} else {
			// Single day
			d := parseDayNum(part)
			if d >= 0 && d < 7 {
				days[d] = true
			}
		}
	}

	return days
}

func parseDayNum(s string) int {
	s = strings.TrimSpace(strings.ToUpper(s))
	switch s {
	case "SUN", "0":
		return 0
	case "MON", "1":
		return 1
	case "TUE", "2":
		return 2
	case "WED", "3":
		return 3
	case "THU", "4":
		return 4
	case "FRI", "5":
		return 5
	case "SAT", "6":
		return 6
	case "7": // Some systems use 7 for Sunday
		return 0
	}
	n, err := strconv.Atoi(s)
	if err == nil && n >= 0 && n <= 7 {
		if n == 7 {
			return 0
		}
		return n
	}
	return -1
}

// buildCronExpr constructs a cron expression from the editor state.
func buildCronExpr(inputs [4]textinput.Model, days [7]bool) string {
	minute := strings.TrimSpace(inputs[0].Value())
	hour := strings.TrimSpace(inputs[1].Value())
	dom := strings.TrimSpace(inputs[2].Value())
	month := strings.TrimSpace(inputs[3].Value())

	if minute == "" {
		minute = "*"
	}
	if hour == "" {
		hour = "*"
	}
	if dom == "" {
		dom = "*"
	}
	if month == "" {
		month = "*"
	}

	dow := formatDOW(days)

	expr := fmt.Sprintf("%s %s %s %s %s", minute, hour, dom, month, dow)

	// If everything is default, return empty (no schedule)
	if expr == "* * * * *" {
		// Check if ALL fields were left at defaults
		allDefault := true
		for i := range 4 {
			v := strings.TrimSpace(inputs[i].Value())
			if v != "" && v != "*" {
				allDefault = false
				break
			}
		}
		anyDay := false
		for _, d := range days {
			if d {
				anyDay = true
				break
			}
		}
		if allDefault && !anyDay {
			return ""
		}
	}

	return expr
}

// formatDOW converts day booleans to a cron day-of-week field.
func formatDOW(days [7]bool) string {
	anySet := false
	for _, d := range days {
		if d {
			anySet = true
			break
		}
	}
	if !anySet {
		return "*"
	}

	allSet := true
	for _, d := range days {
		if !d {
			allSet = false
			break
		}
	}
	if allSet {
		return "*"
	}

	// Check for contiguous ranges
	var parts []string
	i := 0
	for i < 7 {
		if !days[i] {
			i++
			continue
		}
		start := i
		for i < 7 && days[i] {
			i++
		}
		end := i - 1

		if start == end {
			parts = append(parts, strconv.Itoa(start))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", start, end))
		}
	}

	return strings.Join(parts, ",")
}

// describeCron returns a human-readable description of a cron expression.
func describeCron(expr string) string {
	minute, hour, dom, month, dow := parseCronFields(expr)
	var parts []string

	// Time description
	if minute != "*" && hour != "*" {
		// Handle comma-separated hours
		if strings.Contains(hour, ",") {
			hours := strings.Split(hour, ",")
			var times []string
			for _, h := range hours {
				times = append(times, fmt.Sprintf("%s:%s", zeroPad(h), zeroPad(minute)))
			}
			parts = append(parts, "At "+strings.Join(times, " and "))
		} else {
			parts = append(parts, fmt.Sprintf("At %s:%s", zeroPad(hour), zeroPad(minute)))
		}
	} else if minute != "*" {
		parts = append(parts, fmt.Sprintf("At minute %s", minute))
	} else if hour != "*" {
		parts = append(parts, fmt.Sprintf("At hour %s", hour))
	}

	// DOM description
	if dom != "*" {
		parts = append(parts, fmt.Sprintf("on day %s of the month", dom))
	}

	// Month description
	if month != "*" {
		parts = append(parts, fmt.Sprintf("in month %s", month))
	}

	// DOW description
	if dow != "*" {
		days := parseDOW(dow)
		dayList := describeDays(days)
		if dayList != "" {
			parts = append(parts, dayList)
		}
	}

	if len(parts) == 0 {
		return "Every minute"
	}

	return strings.Join(parts, ", ")
}

func describeDays(days [7]bool) string {
	// Check common patterns
	weekdays := days == [7]bool{false, true, true, true, true, true, false}
	weekends := days == [7]bool{true, false, false, false, false, false, true}

	if weekdays {
		return "on weekdays"
	}
	if weekends {
		return "on weekends"
	}

	var names []string
	for i, d := range days {
		if d {
			names = append(names, dayNames[i])
		}
	}

	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return "on " + names[0]
	}
	return "on " + strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

func zeroPad(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}
