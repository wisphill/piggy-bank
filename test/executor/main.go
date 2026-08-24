package main

import (
	"context"
	"fmt"
	executor "piggy-bank/cmd/execute_commands"
)

func main() {
	out1, err := executor.ExecuteCommands(
		context.Background(),
		"echo === STEP 1 ===",
		"uptime",
		"echo === STEP 2 ===",
	)
	if err != nil {
		fmt.Println("Error:", err)
	}
	fmt.Println(out1)

	cmds := []string{
		"echo 'Shutting down service...'",
		"echo 'Done!'",
	}
	out2, _ := executor.ExecuteCommands(context.Background(), cmds...)
	fmt.Println(out2)
}
