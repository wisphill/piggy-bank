package state

import (
	"context"
	"fmt"
	server "piggy-bank/cmd/host"
	"sync"
	"time"

	"gioui.org/widget"
)

type HostAction int

const (
	HostActionTurnOn HostAction = iota
	HostActionShutdown
)

type TerminalScript struct {
	Action   HostAction
	Commands []string // commands for running the script
}

type HostState struct {
	// protected by mutex
	Mu          sync.Mutex
	Name        string
	Description string
	Address     string
	IsOnline    bool
	PingRTT     time.Duration
	Wsls        []*WSLState
	Busy        bool // a power action is in progress
	BusyAction  HostAction
	BusyLabel   string // e.g. "Shutting down…"
	SSHChecked  bool   // at least one SSH attempt has finished
	SSHReady    bool   // the last SSH attempt succeeded
	SSHError    string // why the last SSH attempt failed

	Activity     *ActivityLog
	ServerSignal chan bool
}

type WSLState struct {
	Name     string
	Status   string // Running or Stopped
	Pending  bool   // a start/stop command is in flight
	BtnPower widget.Clickable
}

// WSLSnapshot is a copy of a WSL instance's fields, safe to read without Mu.
type WSLSnapshot struct {
	Node    *WSLState
	Name    string
	Status  string
	Pending bool
}

func (host *HostState) WSLSnapshot() []WSLSnapshot {
	host.Mu.Lock()
	defer host.Mu.Unlock()
	out := make([]WSLSnapshot, len(host.Wsls))
	for i, n := range host.Wsls {
		out[i] = WSLSnapshot{Node: n, Name: n.Name, Status: n.Status, Pending: n.Pending}
	}
	return out
}

// refreshWSL lists WSL instances over SSH and records whether SSH is usable.
func (host *HostState) refreshWSL(ctx context.Context) {
	wslNodes, err := server.GetWSLNodes(ctx)
	if ctx.Err() != nil {
		return
	}

	host.Mu.Lock()
	wasReady, wasChecked, online := host.SSHReady, host.SSHChecked, host.IsOnline
	host.SSHChecked = true
	host.SSHReady = err == nil

	if err != nil {
		host.SSHError = err.Error()
		host.Wsls = nil
		host.Mu.Unlock()
		fmt.Println("Error while getting the WSL nodes:", err)
		if online && (wasReady || !wasChecked) {
			host.Activity.Log("SSH is not reachable: " + err.Error())
		}
		return
	}

	host.SSHError = ""
	// reuse existing entries so their buttons keep click state
	existing := make(map[string]*WSLState, len(host.Wsls))
	for _, n := range host.Wsls {
		existing[n.Name] = n
	}
	next := make([]*WSLState, 0, len(wslNodes))
	for _, wslNode := range wslNodes {
		n, ok := existing[wslNode.Name]
		if !ok {
			n = &WSLState{Name: wslNode.Name}
		}
		n.Status = wslNode.Status
		next = append(next, n)
	}
	host.Wsls = next
	host.Mu.Unlock()

	if wasChecked && !wasReady {
		host.Activity.Log("SSH connected")
		host.Activity.HideToastAfter(2 * time.Second)
	}
}

func (host *HostState) PingToServerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var wg sync.WaitGroup
			host.Mu.Lock()
			addr := host.Address
			host.Mu.Unlock()

			wg.Add(1)
			go func(h *HostState, address string) {
				defer wg.Done()
				online, rtt := server.PingOS(address, 1500*time.Millisecond)
				h.Mu.Lock()
				h.IsOnline = online
				h.PingRTT = rtt
				h.Mu.Unlock()
			}(host, addr)
			wg.Wait()

			time.Sleep(3 * time.Second)
		}
	}
}

func (host *HostState) FetchWSLNodesLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("yeah, return, no more loops!")
			return
		default:
			host.refreshWSL(ctx)
			time.Sleep(3 * time.Second)
		}
	}
}

func (host *HostState) HandleServerSignal(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("yeah, return, no more loops!")
			return
		case signal, ok := <-host.ServerSignal:
			if !ok {
				fmt.Println("ServerSignal closed")
				return
			}

			if signal == false {
				fmt.Println("Turning off the server!")
				server.TurnOffServer(ctx)
			} else {
				fmt.Println("Turning on the server!")
				server.TurnOnServer(ctx)
			}
		}
	}
}
