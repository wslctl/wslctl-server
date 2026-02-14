package wslctl_server

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	defaultTimeout  = 30 * time.Second
	defaultWaitTime = 500 * time.Millisecond
	pipeName        = `\\.\pipe\wslctl`
)

type Service struct {
	name        string
	description string
}

func NewService(name, description string) *Service {
	return &Service{name: name, description: description}
}

func (service *Service) GetName() string {
	return service.name
}

func (service *Service) GetDescription() string {
	return service.description
}

// ---------------- Windows Service Execute ----------------

func (s *Service) Execute(
	args []string,
	requests <-chan svc.ChangeRequest,
	status chan<- svc.Status,
) (bool, uint32) {

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	elog, err := eventlog.Open(s.name)
	if err != nil {
		return false, 1
	}
	defer elog.Close()
	elog.Info(1, "Service started")

	// Start background tasks
	go s.runTicker(ctx, elog)
	go s.runPipeServer(ctx, elog)

	// SCM control loop
	for {
		select {
		case req := <-requests:
			switch req.Cmd {
			case svc.Stop, svc.Shutdown:
				elog.Info(1, "Shutdown requested")
				cancel()
				status <- svc.Status{State: svc.StopPending}
				return false, 0
			}
		case <-ctx.Done():
			return false, 0
		}
	}
}

// ---------------- Background Tasks ----------------

func (s *Service) runTicker(ctx context.Context, elog *eventlog.Log) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			elog.Info(1, "Ticker stopped")
			return
		case <-ticker.C:
			elog.Info(1, "hello")
		}
	}
}

// Named pipe server
func (s *Service) runPipeServer(ctx context.Context, elog *eventlog.Log) {
	config := &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;BA)", // Administrators only
		MessageMode:        true,
	}

	listener, err := winio.ListenPipe(pipeName, config)
	if err != nil {
		elog.Error(1, fmt.Sprintf("Named pipe listen failed: %v", err))
		return
	}
	defer listener.Close()
	elog.Info(1, "Named pipe server started")

	for {
		select {
		case <-ctx.Done():
			elog.Info(1, "Pipe server shutting down")
			return
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			elog.Error(1, fmt.Sprintf("Pipe accept error: %v", err))
			continue
		}

		go s.handlePipeConnection(conn, elog)
	}
}

func (s *Service) handlePipeConnection(conn net.Conn, elog *eventlog.Log) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	msg, _ := reader.ReadString('\n')

	switch msg {
	case "hello\n":
		conn.Write([]byte("hello from wslctl-server\n"))
		elog.Info(1, "Responded to hello request")
	default:
		conn.Write([]byte("unknown command\n"))
	}
}

// ---------------- SCM / Service Management ----------------

func (s *Service) Register() error {
	return s.withMgr(func(m *mgr.Mgr) error {
		exepath, _ := os.Executable()

		svcHandle, err := m.OpenService(s.name)
		if err == nil {
			defer svcHandle.Close()
			return svcHandle.UpdateConfig(mgr.Config{Description: s.description})
		}

		svcHandle, err = m.CreateService(s.name, exepath, mgr.Config{
			DisplayName: s.name,
			Description: s.description,
			StartType:   mgr.StartAutomatic,
		})
		if err != nil {
			return err
		}
		defer svcHandle.Close()

		return eventlog.InstallAsEventCreate(s.name, eventlog.Info|eventlog.Warning|eventlog.Error)
	})
}

func (s *Service) Unregister() error {
	return s.withMgrAndService(func(_ *mgr.Mgr, svcHandle *mgr.Service) error {
		status, _ := svcHandle.Query()
		if status.State == svc.Running {
			_, _ = svcHandle.Control(svc.Stop)
			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			defer cancel()
			_ = waitForState(ctx, svcHandle, svc.Stopped, defaultWaitTime)
		}

		err := svcHandle.Delete()
		_ = eventlog.Remove(s.name)
		return err
	})
}

func (s *Service) Start() error {
	return s.withMgrAndService(func(_ *mgr.Mgr, svcHandle *mgr.Service) error {
		if err := svcHandle.Start(); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		return waitForState(ctx, svcHandle, svc.Running, defaultWaitTime)
	})
}

func (s *Service) Stop() error {
	return s.withMgrAndService(func(_ *mgr.Mgr, svcHandle *mgr.Service) error {
		_, err := svcHandle.Control(svc.Stop)
		return err
	})
}

// ---------------- Helpers ----------------

func (s *Service) withMgr(fn func(m *mgr.Mgr) error) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	return fn(m)
}

func (s *Service) withMgrAndService(fn func(m *mgr.Mgr, s *mgr.Service) error) error {
	return s.withMgr(func(m *mgr.Mgr) error {
		svcHandle, err := m.OpenService(s.name)
		if err != nil {
			return fmt.Errorf("service not installed")
		}
		defer svcHandle.Close()
		return fn(m, svcHandle)
	})
}

func waitForState(ctx context.Context, s *mgr.Service, desired svc.State, waitTime time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := s.Query()
		if err != nil {
			return err
		}
		if status.State == desired {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}
}
