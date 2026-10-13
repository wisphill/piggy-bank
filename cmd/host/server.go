package server

import (
	"bytes"
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
	"unicode/utf16"
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

// SSH is only reachable through Tailscale (yuu), not the LAN name (Yuu.local).
const (
	sshTarget = "Windows@yuu"
	sshPort   = "2223"
)

// runSSH runs a command on the Windows host. Failures carry ssh's stderr so a
// connection error is never mistaken for empty output.
func runSSH(ctx context.Context, remote string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-p", sshPort, sshTarget, remote,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("ssh %s: %w", sshTarget, ctx.Err())
		}
		msg := strings.TrimSpace(decodeWindowsOutput(stderr.Bytes()))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("ssh %s: %s", sshTarget, msg)
	}
	return out, nil
}

// decodeWindowsOutput handles wsl.exe, which writes UTF-16LE instead of UTF-8.
func decodeWindowsOutput(b []byte) string {
	if len(b) >= 2 && b[1] == 0 {
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
		}
		b = []byte(string(utf16.Decode(u)))
	}
	return strings.TrimPrefix(string(b), "\ufeff")
}

func TurnOffServer(ctx context.Context) error {
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := runSSH(childCtx, "shutdown /s /t 0")
	return err
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

	output, err := runSSH(childCtx, "wsl -l -v")
	if err != nil {
		return nil, err
	}

	// NAME  STATE  VERSION header, then one row per distro; "*" marks the default
	var wslNodes []*wslNodeStatus
	for _, line := range strings.Split(decodeWindowsOutput(output), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if line == "" || strings.HasPrefix(line, "NAME") {
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

func TurnOffWSLNode(ctx context.Context, wslName string) error {
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := runSSH(childCtx, fmt.Sprintf("wsl -t %s", wslName))
	return err
}

func TurnOnWSLNode(ctx context.Context, wslName string) error {
	childCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := runSSH(childCtx, fmt.Sprintf("wsl -d %s", wslName))
	return err
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
