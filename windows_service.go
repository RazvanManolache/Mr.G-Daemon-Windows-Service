package main

import (
	"log"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var service *myService

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

func (m *myService) runMainService(eventLog *eventlog.Log) {
	go run()
	eventLog.Info(1, "Main service started")

	<-m.quit
	close(m.done)
}
