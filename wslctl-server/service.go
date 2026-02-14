package wslctl_server

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

type Service struct {
	name        string
	description string
}

func NewService(name string, description string) *Service {
	return &Service{
		name:        name,
		description: description,
	}
}

func (service *Service) GetName() string {
	return service.name
}

func (service *Service) GetDescription() string {
	return service.description
}

func (service *Service) Execute(
	args []string,
	requests <-chan svc.ChangeRequest,
	status chan<- svc.Status,
) (bool, uint32) {

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	status <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	elog, err := eventlog.Open(service.name)
	if err != nil {
		return false, 1
	}
	defer elog.Close()

	elog.Info(1, "Service started")

	// Start your daemon logic here
	go service.run(ctx, elog)

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

func (service *Service) register(m *mgr.Mgr) error {
	exepath, err := os.Executable()
	if err != nil {
		return err
	}

	s, err := m.OpenService(service.name)
	if err == nil {
		defer s.Close()
		return s.UpdateConfig(mgr.Config{
			Description: service.description,
		})
	}

	s, err = m.CreateService(service.name, exepath, mgr.Config{
		DisplayName: service.name,
		Description: service.description,
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		return err
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(
		service.name,
		eventlog.Info|eventlog.Warning|eventlog.Error,
	)
	if err != nil {
		_ = s.Delete()
		return err
	}

	log.Println("Service registered successfully")
	return nil
}

func (service *Service) Register() error {
	return service.withMgr(service.register)
}

func (service *Service) start(_ *mgr.Mgr, s *mgr.Service) error {
	if err := s.Start(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	_ = waitForState(ctx, s, svc.Running, defaultWaitTime)

	log.Println("Service started successfully")
	return nil
}

func (service *Service) Start() error {
	return service.withMgrAndService(service.start)
}

func (service *Service) stop(_ *mgr.Mgr, s *mgr.Service) error {
	_, err := s.Control(svc.Stop)
	return err
}

func (service *Service) Stop() error {
	return service.withMgrAndService(service.stop)
}

func (service *Service) run(ctx context.Context, elog *eventlog.Log) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			elog.Info(1, "Service stopped")
			return
		case <-ticker.C:
			elog.Info(1, "hello")
		}
	}
}

func (service *Service) unregister(_ *mgr.Mgr, s *mgr.Service) error {
	status, err := s.Query()
	if err == nil && status.State == svc.Running {
		_, _ = s.Control(svc.Stop)

		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		if err := waitForState(ctx, s, svc.Stopped, defaultWaitTime); err != nil {
			return err
		}
	}

	if err := s.Delete(); err != nil {
		return err
	}

	_ = eventlog.Remove(service.name)

	log.Println("Service unregistered successfully")
	return nil
}

func (service *Service) Unregister() error {
	return service.withMgrAndService(service.unregister)
}

func (service *Service) withMgr(fn func(m *mgr.Mgr) error) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	return fn(m)
}

func (service *Service) withMgrAndService(fn func(m *mgr.Mgr, s *mgr.Service) error) error {
	return service.withMgr(func(m *mgr.Mgr) error {
		s, err := m.OpenService(service.name)
		if err != nil {
			return fmt.Errorf("service not installed")
		}
		defer s.Close()

		return fn(m, s)
	})
}

const defaultTimeout = 30 * time.Second
const defaultWaitTime = 500 * time.Millisecond

func waitForState(
	ctx context.Context,
	s *mgr.Service,
	desired svc.State,
	waitTime time.Duration,
) error {
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
