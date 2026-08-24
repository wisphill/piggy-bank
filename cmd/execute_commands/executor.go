package executor

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// use the OS shell
func ExecuteCommands(ctx context.Context, commands ...string) (string, error) {
	if len(commands) == 0 {
		return "", nil
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		joinedCmds := strings.Join(commands, " & ")
		cmd = exec.CommandContext(ctx, "cmd", "/C", joinedCmds)
	default:
		joinedCmds := strings.Join(commands, " && ")
		cmd = exec.CommandContext(ctx, "sh", "-c", joinedCmds)
	}

	// Execute and get the stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("Error while running commands [%v]: %w", commands, err)
	}

	return string(output), nil
}
