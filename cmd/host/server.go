package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	executor "piggy-bank/cmd/execute_commands"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
)

var timeRegex = regexp.MustCompile(`(?i)time[=<]\s*([\d.]+)\s*ms`)

type wslNodeStatus struct {
	Name   string
	Status string
}

type tailscaleStatus struct {
	BackendState string `json:"BackendState"`
}

// TailscaleHealth separates whether the CLI is available from whether it is
// connected to a tailnet. Connected is true only for BackendState "Running".
type TailscaleHealth struct {
	CLIAvailable bool
	Connected    bool
	BackendState string
	Error        string
}

// CheckTailscale verifies that the Tailscale CLI is available and reads its
// JSON status. A running backend is required before the SSH host can be
// reached through its Tailscale DNS name.
func CheckTailscale(ctx context.Context) TailscaleHealth {
	binary, err := exec.LookPath("tailscale")
	if err != nil {
		// GUI applications do not necessarily inherit the shell PATH. The
		// macOS Tailscale app ships its CLI at this well-known bundle path.
		binary, err = exec.LookPath("/Applications/Tailscale.app/Contents/MacOS/Tailscale")
		if err != nil {
			return TailscaleHealth{Error: "Tailscale CLI is not installed or unavailable"}
		}
	}

	out, err := exec.CommandContext(ctx, binary, "status", "--json").CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message != "" {
			return TailscaleHealth{CLIAvailable: true, Error: "Tailscale is unavailable: " + message}
		}
		return TailscaleHealth{CLIAvailable: true, Error: fmt.Sprintf("run Tailscale status: %v", err)}
	}

	status, err := parseTailscaleStatus(out)
	if err != nil {
		return TailscaleHealth{CLIAvailable: true, Error: err.Error()}
	}

	health := TailscaleHealth{CLIAvailable: true, BackendState: status.BackendState}
	if status.BackendState == "Running" {
		health.Connected = true
		return health
	}
	if status.BackendState == "" {
		health.Error = "Tailscale did not report a connection state"
	} else {
		health.Error = fmt.Sprintf("Tailscale is %s; connect it to access the tailnet", strings.ToLower(status.BackendState))
	}
	return health
}

func parseTailscaleStatus(out []byte) (tailscaleStatus, error) {
	var status tailscaleStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return tailscaleStatus{}, fmt.Errorf("read Tailscale status: %w", err)
	}
	return status, nil
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

// CheckSSHPort dials the SSH port, returning an error that says whether the
// name doesn't resolve, the host is down/blocked, or the port is closed.
func CheckSSHPort(ctx context.Context, timeout time.Duration) error {
	host := sshTarget[strings.Index(sshTarget, "@")+1:]
	addr := net.JoinHostPort(host, sshPort)

	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		var dnsErr *net.DNSError
		var netErr net.Error
		switch {
		case errors.As(err, &dnsErr):
			return fmt.Errorf("cannot resolve %s (is Tailscale connected?)", host)
		case errors.Is(err, syscall.ECONNREFUSED):
			return fmt.Errorf("%s is up but nothing listens on port %s (is sshd running?)", host, sshPort)
		case errors.As(err, &netErr) && netErr.Timeout():
			return fmt.Errorf("timed out connecting to %s: host is off/asleep or a firewall blocks the port", addr)
		}
		return fmt.Errorf("cannot connect to %s: %w", addr, err)
	}
	return conn.Close()
}

// runSSH runs a command on the Windows host. Failures carry ssh's stderr so a
// connection error is never mistaken for empty output.
func runSSH(ctx context.Context, remote string) ([]byte, error) {
	if err := CheckSSHPort(ctx, 3*time.Second); err != nil {
		return nil, fmt.Errorf("ssh %s: %w", sshTarget, err)
	}

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
	// dbus-launch leaves a dbus-daemon running so WSL doesn't idle the distro
	// out. It must be started through WMI: Windows OpenSSH kills every process
	// the session spawned when it closes, including that daemon.
	wslCmd := fmt.Sprintf("wsl.exe -d %s --exec dbus-launch true", wslName)
	script := fmt.Sprintf(`$ProgressPreference = 'SilentlyContinue'
$r = Invoke-CimMethod -ClassName Win32_Process -MethodName Create -Arguments @{ CommandLine = '%s' }
if ($r.ReturnValue -ne 0) { [Console]::Error.WriteLine("Win32_Process.Create returned $($r.ReturnValue)"); exit 1 }`,
		strings.ReplaceAll(wslCmd, "'", "''"))

	_, err := runSSH(childCtx, "powershell -NoProfile -NonInteractive -EncodedCommand "+encodePowerShell(script))
	return err
}

// encodePowerShell encodes a script for powershell -EncodedCommand, which
// avoids quoting issues between ssh, the Windows shell and PowerShell.
func encodePowerShell(script string) string {
	u := utf16.Encode([]rune(script))
	b := make([]byte, 2*len(u))
	for i, c := range u {
		b[2*i] = byte(c)
		b[2*i+1] = byte(c >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
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
