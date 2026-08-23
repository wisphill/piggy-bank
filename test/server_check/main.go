package main

import (
	"fmt"
	server "piggy-bank/cmd/host"
	"time"
)

func main() {
	hosts := []string{"Yuu.local", "phil.local"}

	for _, host := range hosts {
		serverOnline, ping := server.PingOS(host, 5*time.Second)
		if serverOnline {
			fmt.Printf("✅ %s is ONLINE, ping: %s \n", host, ping)
		} else {
			fmt.Printf("❌ %s is OFFLINE\n", host)
		}
	}

	out, err := server.GetWSLNodes()
	if err != nil {
		return
	}

	fmt.Println("Running WSL Nodes ", out)
}
