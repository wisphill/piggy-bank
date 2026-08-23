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
	Mu       sync.Mutex
	Name     string
	Address  string
	IsOnline bool
	PingRTT  time.Duration
	Wsls     []*WSLState

	ServerSignal chan bool
	BtnPower     widget.Clickable
}

type WSLState struct {
	Name     string
	Status   string // Running or Stopped
	BtnPower widget.Clickable
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
			var wg sync.WaitGroup
			host.Mu.Lock()
			addr := host.Address
			host.Mu.Unlock()

			wg.Add(1)
			go func(h *HostState, address string) {
				defer wg.Done()
				wslNodes, err := server.GetWSLNodes()
				h.Mu.Lock()
				h.Wsls = make([]*WSLState, 0)
				if err != nil {
					fmt.Println("Error while getting the WSL nodes")
					h.Mu.Unlock()
					return
				}

				for _, wslNode := range wslNodes {
					h.Wsls = append(h.Wsls, &WSLState{
						Name:   wslNode.Name,
						Status: wslNode.Status,
					})
				}

				h.Mu.Unlock()
			}(host, addr)
			wg.Wait()
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
				server.TurnOffServer()
			} else {
				fmt.Println("Turning on the server!")
				server.TurnOnServer()
			}
		}
	}
}
