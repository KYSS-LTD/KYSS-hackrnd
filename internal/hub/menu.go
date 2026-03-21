package hub

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"ssh-games/internal/lobby"
	"ssh-games/internal/tui"

	gossh "github.com/gliderlabs/ssh"
)

type menu struct {
	sess    gossh.Session
	manager *lobby.Manager
	nick    string
	pty     gossh.Pty
	winCh   <-chan gossh.Window
	w       int
	h       int
}

func newMenu(sess gossh.Session, m *lobby.Manager, nick string, pty gossh.Pty, winCh <-chan gossh.Window) *menu {
	return &menu{
		sess:    sess,
		manager: m,
		nick:    nick,
		pty:     pty,
		winCh:   winCh,
		w:       pty.Window.Width,
		h:       pty.Window.Height,
	}
}

func (m *menu) write(s string) {
	io.WriteString(m.sess, s)
}

func (m *menu) run() {
	m.write(tui.AltScreenOn)
	m.write(tui.CursorHide)
	defer func() {
		m.write(tui.CursorShow)
		m.write(tui.AltScreenOff)
	}()

	games := availableGames()

	for {
		game := lobby.GameSnake
		if len(games) > 1 {
			selected := m.selectGame(games)
			if selected == "" {
				return
			}
			game = lobby.GameType(selected)
		}

		action := m.selectLobby(game, games)
		if action == "back" {
			continue
		}
		if action == "quit" {
			return
		}
	}
}

func (m *menu) updateSize() {
	select {
	case w := <-m.winCh:
		m.w = w.Width
		m.h = w.Height
	default:
	}
}

func (m *menu) clearAndDraw() {
	m.updateSize()
	m.write(tui.ClearScreen)
	m.write(tui.Home)
}

func renderHeader(width int, nick string) string {
	innerWidth := max(0, width-2)
	var b strings.Builder
	b.WriteString(tui.FgGray)
	b.WriteString(fmt.Sprintf("╔%s╗\r\n", strings.Repeat("═", innerWidth)))
	title := tui.CenterPad("SSH GAMES", innerWidth)
	b.WriteString(fmt.Sprintf("║%s%s%s%s║\r\n", tui.Bold+tui.FgWhite, title, tui.Reset+tui.FgGray, ""))
	sub := tui.CenterPad(fmt.Sprintf("player: %s", nick), innerWidth)
	b.WriteString(fmt.Sprintf("║%s%s%s║\r\n", tui.Dim, sub, tui.Reset+tui.FgGray))
	b.WriteString(fmt.Sprintf("╚%s╝\r\n", strings.Repeat("═", innerWidth)))
	b.WriteString(tui.Reset)
	return b.String()
}

func (m *menu) drawHeader() {
	m.write(renderHeader(m.w, m.nick))
}

func (m *menu) selectGame(games []gameOption) string {
	cursor := 0

	readBuf := make([]byte, 3)
	for {
		m.clearAndDraw()
		m.drawHeader()
		m.write("\r\n")
		m.write(tui.FgGray + "  выберите игру:\r\n\r\n" + tui.Reset)

		for i, g := range games {
			if i == cursor {
				m.write(fmt.Sprintf("  %s▶ %s%-12s%s  %s%s%s\r\n",
					tui.Bold+tui.FgWhite, tui.Reset+tui.Bold+tui.FgWhite,
					g.name, tui.Reset,
					tui.Dim+tui.FgGray, g.desc, tui.Reset))
			} else {
				m.write(fmt.Sprintf("    %s%-12s%s  %s%s%s\r\n",
					tui.FgGray, g.name, tui.Reset,
					tui.Dim+tui.FgGray, g.desc, tui.Reset))
			}
		}

		m.write(fmt.Sprintf("\r\n  %s↑↓ навигация   Enter выбрать   Q выход%s\r\n",
			tui.Dim+tui.FgGray, tui.Reset))

		n, err := m.sess.Read(readBuf)
		if err != nil {
			return ""
		}
		key := parseKey(readBuf[:n])
		switch key {
		case "up":
			if cursor > 0 {
				cursor--
			}
		case "down":
			if cursor < len(games)-1 {
				cursor++
			}
		case "enter":
			return games[cursor].id
		case "q", "Q":
			return ""
		}
	}
}

type gameOption struct {
	id   string
	name string
	desc string
}

func availableGames() []gameOption {
	return []gameOption{
		{"snake", "SNAKE", "классическая змейка, до 8 игроков"},
		{"pingpong", "PING PONG", "дуэль на ракетках, 1v1 или против бота"},
	}
}

func (m *menu) selectLobby(game lobby.GameType, games []gameOption) string {
	readBuf := make([]byte, 3)
	cursor := 0
	message := ""

	for {
		lobbies := m.manager.ListLobbies(game)
		sort.Slice(lobbies, func(i, j int) bool {
			return lobbies[i].ID < lobbies[j].ID
		})

		items := make([]string, 0, len(lobbies)+2)
		for _, l := range lobbies {
			items = append(items, fmt.Sprintf("лобби %s  [%d/%d]", l.ID, l.Players, l.MaxPlayers))
		}
		items = append(items, "создать новое лобби")
		items = append(items, "← назад")

		if cursor >= len(items) {
			cursor = len(items) - 1
		}

		m.clearAndDraw()
		m.drawHeader()
		m.write("\r\n")
		m.write(fmt.Sprintf("  %s%s%s — выберите лобби:\r\n\r\n", tui.Bold+tui.FgWhite, strings.ToUpper(string(game)), tui.Reset))

		for i, item := range items {
			isCreate := i == len(items)-2
			isBack := i == len(items)-1
			sep := ""
			if isCreate {
				sep = "\r\n"
			}

			var line string
			if i == cursor {
				line = fmt.Sprintf("  %s%s▶ %s%s%s", sep, tui.Bold+tui.FgWhite, tui.Reset+tui.Bold+tui.FgWhite, item, tui.Reset)
			} else {
				color := tui.FgGray
				if isBack {
					color = tui.Dim + tui.FgGray
				}
				line = fmt.Sprintf("  %s  %s%s%s", sep, color, item, tui.Reset)
			}
			m.write(line + "\r\n")
		}

		if message != "" {
			m.write(fmt.Sprintf("\r\n  %s%s%s\r\n", tui.FgGray+tui.Dim, message, tui.Reset))
		}

		m.write(fmt.Sprintf("\r\n  %s↑↓ навигация   Enter выбрать   Q выход%s\r\n",
			tui.Dim+tui.FgGray, tui.Reset))

		n, err := m.sess.Read(readBuf)
		if err != nil {
			return "quit"
		}
		key := parseKey(readBuf[:n])
		message = ""

		switch key {
		case "up":
			if cursor > 0 {
				cursor--
			}
		case "down":
			if cursor < len(items)-1 {
				cursor++
			}
		case "enter":
			if cursor == len(items)-1 {
				return "back"
			}
			if cursor == len(items)-2 {
				selectedGame := game
				if len(games) > 1 {
					selected := m.selectGame(games)
					if selected == "" {
						message = "создание лобби отменено"
						continue
					}
					selectedGame = lobby.GameType(selected)
				}
				message = fmt.Sprintf("создаём %s лобби...", strings.ToUpper(string(selectedGame)))
				m.write(fmt.Sprintf("\r\n  %s%s%s\r\n", tui.FgGray+tui.Dim, message, tui.Reset))
				lob, err := m.manager.CreateLobby(selectedGame)
				if err != nil {
					message = fmt.Sprintf("ошибка: %v", err)
					continue
				}
				proxy := newProxy(m.manager, lob.ID)
				proxy.connect(m.sess, m.nick, m.pty, m.winCh)
				return "back"
			}
			lob := lobbies[cursor]
			if lob.Players >= lob.MaxPlayers {
				message = "лобби заполнено"
				continue
			}
			full, ok := m.manager.GetLobby(lob.ID)
			if !ok {
				message = "лобби недоступно"
				continue
			}
			proxy := newProxy(m.manager, full.ID)
			proxy.connect(m.sess, m.nick, m.pty, m.winCh)
			return "back"
		case "q", "Q":
			return "quit"
		}
	}
}

func parseKey(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) == 3 && b[0] == 0x1b && b[1] == '[' {
		switch b[2] {
		case 'A':
			return "up"
		case 'B':
			return "down"
		case 'C':
			return "right"
		case 'D':
			return "left"
		}
	}
	switch b[0] {
	case 13, 10:
		return "enter"
	case 27:
		return "esc"
	case 3:
		return "ctrl-c"
	default:
		return string(b[:1])
	}
}
