package hub

import (
	"fmt"
	"io"
	"strings"

	"ssh-games/internal/lobby"
	"ssh-games/internal/tui"
)

func runLobbyList(rw io.ReadWriter, mgr *lobby.Manager, game lobby.GameType) (string, error) {
	for {
		lobbies := mgr.ListLobbies(game)

		fmt.Fprint(rw, tui.ClearScreen+tui.Home+tui.CursorHide)

		innerW := 46
		titleRow := 2
		tableStartRow := titleRow + 2

		tui.DrawBoxTitle(rw, titleRow, 2, len(lobbies)+8, innerW, strings.ToUpper(string(game))+" LOBBIES")

		tui.MoveTo(rw, tableStartRow, 4)
		fmt.Fprintf(rw, "%-12s  %-8s", "ID", "PLAYERS")
		tui.MoveTo(rw, tableStartRow+1, 4)
		fmt.Fprint(rw, strings.Repeat("─", innerW-4))

		for i, l := range lobbies {
			tui.MoveTo(rw, tableStartRow+2+i, 4)
			fmt.Fprintf(rw, "%-12s  %d/%d", l.ID, l.Players, l.MaxPlayers)
		}

		offset := len(lobbies) + 2
		tui.MoveTo(rw, tableStartRow+offset, 4)
		fmt.Fprint(rw, strings.Repeat("─", innerW-4))
		tui.MoveTo(rw, tableStartRow+offset+1, 4)
		fmt.Fprint(rw, "[N] Create new lobby")
		tui.MoveTo(rw, tableStartRow+offset+2, 4)
		fmt.Fprint(rw, "[B] Back to game menu")

		promptRow := tableStartRow + offset + 4
		tui.MoveTo(rw, promptRow, 4)
		fmt.Fprint(rw, "Enter lobby ID, N or B: ")
		fmt.Fprint(rw, tui.CursorShow)

		var input []rune
		var errMsg string
		buf := make([]byte, 4)

		for {
			if errMsg != "" {
				tui.MoveTo(rw, promptRow+1, 4)
				fmt.Fprintf(rw, "%s%*s", errMsg, innerW-len(errMsg)-4, "")
			}

			n, err := rw.Read(buf)
			if err != nil {
				return "", err
			}
			if n == 0 {
				continue
			}
			b := buf[0]

			switch {
			case b == 'n' || b == 'N':
				fmt.Fprint(rw, tui.CursorHide)
				return "", nil
			case b == 'b' || b == 'B':
				fmt.Fprint(rw, tui.CursorHide)
				return "-1", nil
			case b == '\r' || b == '\n':
				s := strings.TrimSpace(string(input))
				if s == "" {
					continue
				}
				if _, found := mgr.GetLobby(s); !found {
					errMsg = fmt.Sprintf("Lobby %s not found.", s)
					input = nil
					tui.MoveTo(rw, promptRow, 4)
					fmt.Fprintf(rw, "%-40s", "Enter lobby ID, N or B: ")
					tui.MoveTo(rw, promptRow, 28)
					continue
				}
				fmt.Fprint(rw, tui.CursorHide)
				return s, nil
			case b == 127 || b == 8:
				if len(input) > 0 {
					input = input[:len(input)-1]
					tui.MoveTo(rw, promptRow, 4)
					fmt.Fprintf(rw, "%-40s", "Enter lobby ID, N or B: "+string(input))
					tui.MoveTo(rw, promptRow, 28+len(input))
				}
			case (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || b == '-':
				input = append(input, rune(b))
				fmt.Fprintf(rw, "%c", b)
			}
		}
	}
}
