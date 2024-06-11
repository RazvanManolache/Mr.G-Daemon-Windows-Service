package main

import (
	"log"
	"os"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

// methods to install, remove, start, and stop a service

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
		Description: "Mr.G Daemon service, a service that allows you to run and control applications installed within it",
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
