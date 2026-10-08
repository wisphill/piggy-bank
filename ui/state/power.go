package state

import (
	"context"
	"fmt"
	server "piggy-bank/cmd/host"
	"time"
)

const (
	powerPollInterval = 5 * time.Second
	powerWaitTimeout  = 5 * time.Minute
)

// Shutdown turns the host off and waits until it stops answering pings.
func (host *HostState) Shutdown(ctx context.Context) {
	if !host.beginBusy(HostActionShutdown, "Shutting down…") {
		return
	}
	defer host.endBusy()

	host.Activity.Log("Server is shutting down!")
	server.TurnOffServer(ctx)

	if host.waitForOnline(ctx, false, "Server is shutting down! Checking the server") {
		host.Activity.Log("Server is shut down!")
	}
	host.Activity.HideToastAfter(2 * time.Second)
}

// PowerOn wakes the host and waits until it answers pings.
func (host *HostState) PowerOn(ctx context.Context) {
	if !host.beginBusy(HostActionTurnOn, "Turning on…") {
		return
	}
	defer host.endBusy()

	host.Activity.Log("Server is starting!")
	if err := server.TurnOnServer(ctx); err != nil {
		host.Activity.Log(fmt.Sprintf("Error while starting the server: %v", err))
		host.Activity.HideToastAfter(5 * time.Second)
		return
	}

	if host.waitForOnline(ctx, true, "Server is turning on! Checking the server") {
		host.Activity.Log("Server is turned on!")
	}
	host.Activity.HideToastAfter(2 * time.Second)
}

func (host *HostState) waitForOnline(ctx context.Context, want bool, status string) bool {
	ticker := time.NewTicker(powerPollInterval)
	defer ticker.Stop()
	deadline := time.After(powerWaitTimeout)

	for {
		select {
		case <-ctx.Done():
			return false
		case <-deadline:
			host.Activity.Log("Timed out waiting for the server")
			return false
		case <-ticker.C:
			host.Activity.Status(status)
			if isOnline, _ := server.PingOS("yuu", time.Second); isOnline == want {
				return true
			}
		}
	}
}

func (host *HostState) beginBusy(action HostAction, label string) bool {
	host.Mu.Lock()
	defer host.Mu.Unlock()
	if host.Busy {
		return false
	}
	host.Busy = true
	host.BusyAction = action
	host.BusyLabel = label
	return true
}

func (host *HostState) endBusy() {
	host.Mu.Lock()
	host.Busy = false
	host.BusyLabel = ""
	host.Mu.Unlock()
}

// ToggleWSL starts a stopped instance or stops a running one.
func (host *HostState) ToggleWSL(ctx context.Context, node *WSLState) {
	host.Mu.Lock()
	if node.Pending {
		host.Mu.Unlock()
		return
	}
	node.Pending = true
	name, status := node.Name, node.Status
	host.Mu.Unlock()

	defer func() {
		host.Mu.Lock()
		node.Pending = false
		host.Mu.Unlock()
	}()

	switch status {
	case "Running":
		host.Activity.Log(fmt.Sprintf("Stopping WSL instance %s…", name))
		server.TurnOffWSLNode(ctx, name)
		host.Activity.Log(fmt.Sprintf("Stop command sent to %s", name))
	case "Stopped":
		host.Activity.Log(fmt.Sprintf("Starting WSL instance %s…", name))
		server.TurnOnWSLNode(ctx, name)
		host.Activity.Log(fmt.Sprintf("Start command sent to %s", name))
	}
	host.Activity.HideToastAfter(2 * time.Second)
}
