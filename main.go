package main

import (
	"fmt"
	"log"
	"os"
	"time"

	wslctl_server "github.com/wslctl/wslctl-server/wslctl-server"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

func main() {
	service := wslctl_server.NewService("wslctl-server", "WSL Control Server")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "register":
			if err := service.Register(); err != nil {
				log.Fatalf("Register failed: %v", err)
			}
			return

		case "unregister":
			if err := service.Unregister(); err != nil {
				log.Fatalf("Unregister failed: %v", err)
			}
			return

		case "start":
			if err := service.Start(); err != nil {
				log.Fatalf("Start failed: %v", err)
			}
			return
		case "stop":
			if err := service.Stop(); err != nil {
				log.Fatalf("Stop failed: %v", err)
			}
			return
		}
	}

	isService, err := svc.IsWindowsService()
	if err != nil {
		return
	}

	if isService {
		svc.Run(service.GetName(), service)
	} else {
		fmt.Println("Running in debug mode (Ctrl+C to exit)")
		elog, err := eventlog.Open(service.GetName())
		if err == nil {
			defer elog.Close()
			for {
				elog.Info(1, "hello (debug)")
				time.Sleep(1 * time.Minute)
			}
		}
	}
}
