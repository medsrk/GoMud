package inputhandlers

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"time"

	"log/slog"

	"github.com/volte6/mud/connection"
	"github.com/volte6/mud/templates"
	"github.com/volte6/mud/users"
)

func AdminCommandInputHandler(clientInput *connection.ClientInput, connectionPool *connection.ConnectionTracker, sharedState map[string]any) (nextHandler bool) {

	// if they didn't hit enter, just keep buffering, go next.
	if !clientInput.EnterPressed {
		return true
	}

	// Handle text input

	// grab whatever was typed in so far and trim out leading/trailing whitespace and null byte
	message := strings.TrimSpace(string(clientInput.Buffer))

	//slog.Info("AdminCommandInputHandler Received", "type", "TXT", "size", len(clientInput.Buffer), "data", string(clientInput.Buffer), "message", message)

	// If all they ever sent was white space, we won't have anything to work with. We can just ignore the input...
	// ALternatively, they may have only hit ENTER, and we may do something with that...
	if len(message) == 0 {
		return true
	}

	// If logged in and of appropraite privs, try to run a admin command
	// If successful, we're done.
	if tryAdminCommand(message, connectionPool, clientInput.ConnectionId) {
		// zero out the current buffer
		clientInput.Buffer = clientInput.Buffer[:0]
		return false
	}

	return true
}

// TODO: Move into own handler?

const AdminCommandPrefix = "/"

type AdminCommandHelp struct {
	Description  string
	Details      string
	ExampleInput string
}

var (
	adminCommandList = map[string]AdminCommandHelp{
		"shutdown": AdminCommandHelp{
			Description:  "Shutdown the server",
			Details:      "An optional argument can be provided to specify the number of seconds to wait before shutting down. The default is 15 seconds.",
			ExampleInput: "shutdown [seconds]",
		},
		"adminhelp": AdminCommandHelp{
			Description:  "Display this help message, or help for a specific command",
			ExampleInput: "adminhelp shutdown",
		},
		"where": AdminCommandHelp{
			Description:  "Display the current location of all online users",
			ExampleInput: "where",
		},
	}
)

func commandParts(cmd string) (adminCmd string, cmdArg string) {

	cmd = strings.TrimSpace(cmd)

	// If there's a space split it into cmd vs arg
	if index := strings.Index(cmd, " "); index != -1 {

		adminCmd, cmdArg = strings.ToLower(cmd[0:index]), cmd[index+1:]

		if cmdArg[0:1] == " " {
			cmdArg = strings.TrimSpace(cmdArg)
		}

		return adminCmd, cmdArg
	}

	adminCmd, cmdArg = strings.ToLower(cmd), ""

	return adminCmd, cmdArg
}

func tryAdminCommand(cmd string, connectionPool *connection.ConnectionTracker, connectionId connection.ConnectionId) bool {

	if len(cmd) < 1 {
		return false
	}

	if cmd[0:1] != AdminCommandPrefix {
		return false
	}

	cmd, arg := commandParts(strings.TrimSpace(cmd[1:]))

	// look for cmd in the command list
	if _, ok := adminCommandList[cmd]; !ok {
		return false
	}

	slog.Info("admin command", "cmd", cmd, "arg", arg)
	//fmt.Printf("cmd:[%s] arg:[%s]\n", cmd, arg)

	if cmd == "where" {
		onlineUsers := users.GetOnlineList()

		headers := []string{"UserId", "Username", "Character", "Zone", "RoomId", "Role"}

		rows := [][]string{}
		for _, user := range onlineUsers {
			rows = append(rows, []string{user.UserId, user.Username, user.CharacterName, user.Zone, fmt.Sprintf(`%d`, user.RoomId), user.Permission})
		}

		onlineTableData := templates.GetTable("Online Users", headers, rows)
		tplTxt, _ := templates.Process("tables/generic", onlineTableData)
		connectionPool.SendTo([]byte(tplTxt), connectionId)
		return true
	}

	if cmd == "shutdown" {
		var timeToShutdown uint64 = 15

		if len(arg) > 0 {
			timeToShutdown, _ = strconv.ParseUint(arg, 10, 64)
		}

		go func() {

			// Not building complex output, so just preparse the ansi in the template and cache that
			tplTxt, _ := templates.Process("admincommands/shutdown-countdown", adminCommandList, templates.AnsiTagsPreParse)

			for i := timeToShutdown; i > 0; i-- {

				writeOut := false
				if i == timeToShutdown {
					writeOut = true
				} else if i > 60 {
					if i%30 == 0 {
						writeOut = true
					}
				} else if i > 15 {
					if i%15 == 0 {
						writeOut = true
					}
				} else if i%5 == 0 {
					writeOut = true
				}

				if writeOut {
					connectionPool.Broadcast([]byte(fmt.Sprintf(tplTxt, i)))
				}

				time.Sleep(time.Second)
			}
			connectionPool.Signal(syscall.SIGTERM)

		}()
		return true
	}

	if cmd == "adminhelp" {

		tplTxt, _ := templates.Process("admincommands/help", adminCommandList)
		connectionPool.SendTo([]byte(tplTxt), connectionId)
		return true
	}

	slog.Error("valid command unhandled", "cmd", cmd, "arg", arg)

	return true
}
