package hub

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"ssh-games/internal/lobby"
	"ssh-games/internal/tui"
)

func runLobbyList(rw io.ReadWriter, reg *lobby.Registry, mgr *lobby.Manager, game lobby.GameType) (int, error) {
	for {
		lobbies := reg.List(game)

		fmt.Fprint(rw, tui.ClearScreen+tui.HideCursor)

		innerW := 46
		titleRow := 2
		tableStartRow := titleRow + 2

		tui.DrawBoxWithTitle(rw, titleRow, 2, len(lobbies)+5, innerW, strings.ToUpper(string(game))+" LOBBIES")

		header := fmt.Sprintf("  %-4s  %-8s  %-9s", "ID", "PLAYERS", "STATUS")
		tui.WriteBoxLine(rw, tableStartRow, 2, innerW, header)
		tui.WriteBoxLine(rw, tableStartRow+1, 2, innerW, "  "+strings.Repeat("─", innerW-4))

		for i, l := range lobbies {
			players := fmt.Sprintf("%d/%d", l.PlayerCount, l.MaxPlayers)
			line := fmt.Sprintf("  %-4d  %-8s  %-9s", l.ID, players, l.State.String())
			tui.WriteBoxLine(rw, tableStartRow+2+i, 2, innerW, line)
		}

		offset := len(lobbies) + 2
		tui.WriteBoxLine(rw, tableStartRow+offset, 2, innerW, "  "+strings.Repeat("─", innerW-4))
		tui.WriteBoxLine(rw, tableStartRow+offset+1, 2, innerW, "  [N]  Create new lobby")
		tui.WriteBoxLine(rw, tableStartRow+offset+2, 2, innerW, "  [B]  Back to game menu")

		promptRow := tableStartRow + offset + 4
		tui.WriteAt(rw, promptRow, 2, "  Enter lobby ID, N or B: ")
		fmt.Fprint(rw, tui.ShowCursor)

		var input []rune
		var errMsg string
		buf := make([]byte, 4)

		for {
			if errMsg != "" {
				tui.WriteAt(rw, promptRow+1, 2, "  "+errMsg+"                    ")
			}

			n, err := rw.Read(buf)
			if err != nil {
				return 0, err
			}
			if n == 0 {
				continue
			}
			b := buf[0]

			if b == 'n' || b == 'N' {
				fmt.Fprint(rw, tui.HideCursor)
				return 0, nil
			}

			if b == 'b' || b == 'B' {
				fmt.Fprint(rw, tui.HideCursor)
				return -1, nil
			}

			if b == '\r' || b == '\n' {
				s := strings.TrimSpace(string(input))
				if s == "" {
					continue
				}
				id, convErr := strconv.Atoi(s)
				if convErr != nil {
					errMsg = "Invalid number. Try again."
					input = nil
					tui.WriteAt(rw, promptRow, 2, "  Enter lobby ID, N or B:                 ")
					tui.WriteAt(rw, promptRow, 2, "  Enter lobby ID, N or B: ")
					continue
				}
				_, found := reg.Get(id)
				if !found {
					errMsg = fmt.Sprintf("Lobby %d not found.", id)
					input = nil
					tui.WriteAt(rw, promptRow, 2, "  Enter lobby ID, N or B:                 ")
					tui.WriteAt(rw, promptRow, 2, "  Enter lobby ID, N or B: ")
					continue
				}
				fmt.Fprint(rw, tui.HideCursor)
				return id, nil
			}

			if b == 127 || b == 8 {
				if len(input) > 0 {
					input = input[:len(input)-1]
					col := 28 + len(input)
					tui.WriteAt(rw, promptRow, 2, "  Enter lobby ID, N or B: "+string(input)+" ")
					tui.MoveTo(rw, promptRow, col)
				}
				continue
			}

			if b >= '0' && b <= '9' {
				input = append(input, rune(b))
				fmt.Fprintf(rw, "%c", b)
			}
		}
	}
}
