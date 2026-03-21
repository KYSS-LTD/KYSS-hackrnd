package kitchen

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

type playerFrame struct {
	Nick       string
	X          int
	Y          int
	Carrying   itemType
	Score      int
	Deliveries int
	Mistakes   int
	Prompt     *promptState
}

type requirementFrame struct {
	Item itemType
	Need int
	Have int
}

type orderFrame struct {
	Title      string
	Result     string
	Requires   []requirementFrame
	Code       []string
	BugLine    int
	CodeFixed  bool
	WrongDrops int
	Age        time.Duration
}

type anomalyFrame struct {
	Kind    stationKind
	Target  stationKind
	Command string
}

type frameState struct {
	Players     []playerFrame
	CPU         int
	Memory      int
	Processes   int
	TotalScore  int
	Streak      int
	Message     string
	MixBuffer   []itemType
	MixOutput   itemType
	ActiveOrder orderFrame
	NextOrders  []string
	Anomalies   []anomalyFrame
	GameOver    bool
}

func renderFrame(state frameState, viewer string) []byte {
	var buf bytes.Buffer
	buf.WriteString("\x1b[H")
	buf.WriteString("  ASCII KITCHEN RUSH\r\n")
	buf.WriteString(fmt.Sprintf("  cpu %s  mem %s  proc %s  score %04d  streak x%d\r\n", bar(state.CPU, maxCPU, 14, '!'), bar(state.Memory, maxMemory, 10, '#'), bar(state.Processes, maxProcessSlots, 8, '+'), state.TotalScore, state.Streak))
	buf.WriteString("  ┌────────────────────────────────────┬──────────────────────────────────────────────┐\r\n")
	for row := 1; row <= boardHeight; row++ {
		buf.WriteString("  │")
		buf.WriteString(renderBoardRow(state, row, viewer))
		buf.WriteString("│")
		buf.WriteString(renderSidePanel(state, row, viewer))
		buf.WriteString("│\r\n")
	}
	buf.WriteString("  └────────────────────────────────────┴──────────────────────────────────────────────┘\r\n")
	buf.WriteString("  WASD/стрелки — ходить, E/Enter — станция, X — выбросить предмет, Q — выход.\r\n")
	buf.WriteString(fmt.Sprintf("  log: %s\r\n", fit(state.Message, 90)))
	if state.GameOver {
		buf.WriteString("  KERNEL PANIC: CPU достиг 100%. Нажмите R для новой смены.\r\n")
	}
	return buf.Bytes()
}

func renderBoardRow(state frameState, row int, viewer string) string {
	cells := make([]string, boardWidth)
	for i := range cells {
		cells[i] = "."
	}
	for station, pos := range stationPositions {
		if pos.Y == row {
			cells[pos.X-1] = stationGlyph(station)
		}
	}
	for _, p := range state.Players {
		if p.Y == row {
			glyph := "@"
			if p.Nick != viewer {
				glyph = strings.ToUpper(string([]rune(p.Nick)[0]))
			}
			cells[p.X-1] = glyph
		}
	}
	return strings.Join(cells, "")
}

func renderSidePanel(state frameState, row int, viewer string) string {
	lines := make([]string, boardHeight)
	for i := range lines {
		lines[i] = strings.Repeat(" ", 46)
	}
	set := func(i int, s string) {
		if i >= 0 && i < len(lines) {
			lines[i] = fit(s, 46)
		}
	}
	set(0, fmt.Sprintf(" ORDER: %s", strings.ToUpper(state.ActiveOrder.Title)))
	set(1, fmt.Sprintf(" TARGET: %s", state.ActiveOrder.Result))
	status := "PENDING"
	if state.ActiveOrder.CodeFixed {
		status = "FIXED"
	}
	set(2, fmt.Sprintf(" CODE: %s   age:%2.0fs", status, state.ActiveOrder.Age.Seconds()))
	set(3, " RECIPE:")
	for i, req := range state.ActiveOrder.Requires {
		set(4+i, fmt.Sprintf("  - %-14s %d/%d", itemLabel(req.Item), req.Have, req.Need))
	}
	base := 4 + len(state.ActiveOrder.Requires)
	set(base, fmt.Sprintf(" WRONG DROPS: %d", state.ActiveOrder.WrongDrops))
	set(base+1, " CODE BUG:")
	for i, line := range state.ActiveOrder.Code {
		marker := " "
		if i == state.ActiveOrder.BugLine {
			marker = "!"
		}
		if state.ActiveOrder.CodeFixed && i == state.ActiveOrder.BugLine {
			marker = "✓"
		}
		set(base+2+i, fmt.Sprintf("  %s %s", marker, line))
	}
	mixRow := boardHeight - 5
	set(mixRow, fmt.Sprintf(" MIX: [%s] -> %s", joinItems(state.MixBuffer), itemLabel(state.MixOutput)))
	if len(state.Anomalies) > 0 {
		set(mixRow+1, " ALERTS:")
		for i, a := range state.Anomalies {
			set(mixRow+2+i, fmt.Sprintf("  %s on %s", strings.ToUpper(string(a.Kind)), stationLabel(a.Target)))
		}
	} else {
		set(mixRow+1, " ALERTS: none")
	}

	for _, p := range state.Players {
		if p.Nick == viewer {
			set(boardHeight-2, fmt.Sprintf(" YOU: %s carry=%s", p.Nick, itemLabel(p.Carrying)))
			if p.Prompt != nil {
				set(boardHeight-1, fmt.Sprintf(" CMD[%s]: %s_", p.Prompt.hint, p.Prompt.input))
			} else {
				set(boardHeight-1, fmt.Sprintf(" TASKS: next -> %s / %s", strings.Join(state.NextOrders, ", "), strings.Repeat(" ", 8)))
			}
			break
		}
	}
	return lines[row-1]
}

func stationGlyph(station stationKind) string {
	switch station {
	case stationSecurity:
		return "S"
	case stationCore:
		return "C"
	case stationParser:
		return "P"
	case stationStdout:
		return "O"
	case stationMixer:
		return "M"
	case stationBuild:
		return "B"
	case stationDebug:
		return "D"
	case stationDispatch:
		return "X"
	case stationVirus:
		return "V"
	case stationFreeze:
		return "F"
	default:
		return "?"
	}
}

func joinItems(items []itemType) string {
	if len(items) == 0 {
		return "empty"
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = itemLabel(item)
	}
	return strings.Join(parts, "+")
}

func fit(s string, width int) string {
	if len(s) > width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func bar(value, maxValue, width int, fill rune) string {
	if maxValue <= 0 {
		return "[invalid]"
	}
	filled := value * width / maxValue
	if filled > width {
		filled = width
	}
	return fmt.Sprintf("[%s%s]", strings.Repeat(string(fill), filled), strings.Repeat("-", width-filled))
}
