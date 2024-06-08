package main

import (
	"log"
	"os"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

type myService struct {
	quit chan struct{}
	done chan struct{}
}

func runApplication() {
	IsWindowsService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if !IsWindowsService {
		runInteractive()
	} else {
		runService(serviceName, false)
	}
}

func runInteractive() {
	log.Print("Running in interactive mode")
	service := newMyService()
	go baseLoop(service.quit, service.done)
}

var status_app string = "starting"

var service *myService

func runService(name string, isDebug bool) {
	service = newMyService()
	var err error
	if isDebug {
		err = debug.Run(name, service)
	} else {
		err = svc.Run(name, service)
	}
	status_app = "Running"
	if err != nil {
		log.Fatalf("%s service failed: %v", name, err)
	}
}

func newMyService() *myService {
	return &myService{

		quit: make(chan struct{}),
		done: make(chan struct{}),
	}
}

func (m *myService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	s <- svc.Status{State: svc.StartPending}
	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	log.Print("Service is running")

	eventLog, err := eventlog.Open(serviceName)
	if err != nil {
		log.Fatalf("cannot open event log: %v", err)
	}
	defer eventLog.Close()
	eventLog.Info(1, "Service started")

	// Start the main service logic
	go m.runMainService(eventLog)

loop:
	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			s <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			close(m.quit)
			break loop
		default:
			eventLog.Warning(1, "unexpected control request")
		}
	}

	s <- svc.Status{State: svc.StopPending}
	<-m.done
	eventLog.Info(1, "Service stopped")
	s <- svc.Status{State: svc.Stopped}
	return false, 0
}

func (m *myService) runMainService(eventLog *eventlog.Log) {
	go baseLoop(m.quit, m.done)
	eventLog.Info(1, "Main service started")

	<-m.quit
	close(m.done)
}

func installService(name, desc string) {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		log.Fatalf("service %s already exists", name)
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable path: %v", err)
	}

	s, err = m.CreateService(name, exePath, mgr.Config{
		DisplayName: desc,
		StartType:   mgr.StartAutomatic,
	}, "is", "auto-started")
	if err != nil {
		log.Fatalf("failed to create service: %v", err)
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		log.Fatalf("failed to setup event log source: %v", err)
	}

	log.Printf("service %s installed", name)
}

func removeService(name string) {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		log.Fatalf("service %s not found", name)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		log.Fatalf("failed to delete service: %v", err)
	}

	err = eventlog.Remove(name)
	if err != nil {
		log.Fatalf("failed to remove event log source: %v", err)
	}

	log.Printf("service %s removed", name)
}

func startService(name string) {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		log.Fatalf("service %s not found", name)
	}
	defer s.Close()

	err = s.Start("is", "manual-started")
	if err != nil {
		log.Fatalf("failed to start service: %v", err)
	}

	log.Printf("service %s started", name)
}

func restartService() {
	softStopService()
	status_app = "Restarting"
	runService(serviceName, false)
}

func softStopService() {
	status_app = "Stopping"
	stopAllSubApplications()
	service.quit <- struct{}{}
	os.Exit(0)
}

func stopService(name string) {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		log.Fatalf("service %s not found", name)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		log.Fatalf("failed to stop service: %v", err)
	}

	log.Printf("service %s stopped with status %v", name, status)
}
