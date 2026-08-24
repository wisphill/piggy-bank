package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	executor "piggy-bank/cmd/execute_commands"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var timeRegex = regexp.MustCompile(`(?i)time[=<]\s*([\d.]+)\s*ms`)

type wslNodeStatus struct {
	Name   string
	Status string
}

// resolve host to the ipv4
func resolveHost(host string) string {
	host = strings.TrimSpace(host)

	if ip := net.ParseIP(host); ip != nil {
		return host
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return host
	}

	// get the IP v4
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}

	return host
}

func TurnOffServer(ctx context.Context) {
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := executor.ExecuteCommands(
		childCtx,
		`ssh Windows@yuu -p 2223 "shutdown /s /t 0"`,
	)
	if err != nil {
		fmt.Printf("Error while processing shutting down %v", err)
	}
}

func TurnOnServer(ctx context.Context) error {
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramBotToken == "" {
		return fmt.Errorf("Cannot find the telegram bot token. Configure at .piggy_bank/config")
	}
	command := fmt.Sprintf(
		`curl -s -X POST "https://api.telegram.org/bot%s/sendMessage" -d chat_id="-5115557042" -d text="/wake"`,
		telegramBotToken,
	)
	_, err := executor.ExecuteCommands(childCtx, command)
	if err != nil {
		fmt.Printf("Error while processing turning the server on %v", err)
		return err
	}

	return nil
}

func GetWSLNodes(ctx context.Context) ([]*wslNodeStatus, error) {
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	output, err := executor.ExecuteCommands(
		childCtx,
		`ssh Windows@yuu -p 2223 "wsl -l -v" | iconv -f UTF-16LE -t UTF-8 | sed '1d; s/^\* //'`)
	if err != nil {
		fmt.Printf("Error while getting the WSL nodes %v", err)
		return nil, err
	}

	var wslNodes []*wslNodeStatus
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		wslNodes = append(wslNodes, &wslNodeStatus{
			Name:   fields[0],
			Status: fields[1],
		})
	}

	return wslNodes, nil
}

func TurnOffWSLNode(ctx context.Context, wslName string) {
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := executor.ExecuteCommands(
		childCtx,
		fmt.Sprintf(`ssh Windows@yuu -p 2223 "wsl -t %s"`, wslName),
	)
	if err != nil {
		fmt.Printf("Error while processing shutting down the WSL node %v", err)
	}
}

func TurnOnWSLNode(ctx context.Context, wslName string) {
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := executor.ExecuteCommands(
		childCtx,
		fmt.Sprintf(`ssh Windows@yuu -p 2223  "wsl -d %s"`, wslName),
	)
	if err != nil {
		fmt.Printf("Error while processing shutting down the WSL node %v", err)
	}
}

func PingOS(host string, timeout time.Duration) (bool, time.Duration) {
	var cmd *exec.Cmd
	timeoutMs := strconv.Itoa(int(timeout.Milliseconds()))

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("ping", "-n", "1", "-w", timeoutMs, host)
	case "darwin":
		cmd = exec.Command("ping", "-c", "1", "-W", timeoutMs, host)
	default:
		timeoutSec := strconv.Itoa(int(timeout.Seconds()))
		if timeoutSec == "0" {
			timeoutSec = "1"
		}
		cmd = exec.Command("ping", "-c", "1", "-W", timeoutSec, host)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, 0
	}

	matches := timeRegex.FindStringSubmatch(string(out))
	if len(matches) > 1 {
		msFloat, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return true, time.Duration(msFloat * float64(time.Millisecond))
		}
	}

	return true, 0
}
