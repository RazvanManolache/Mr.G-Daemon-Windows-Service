package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LogEvent struct {
	Message string `json:"message"`
	AppId   string `json:"appId"`
}

func logMessage(subAppDir string, name string, prefix string, message string) string {

	// Create a log file for the current day if it doesn't exist
	logFileName := prefix + "." + getTodayAsString() + ".log"
	logFilePath := filepath.Join(subAppDir, logFileName)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		return ""
	}
	defer logFile.Close()

	// Write the log message to the log file
	if prefix == "log" {
		message = fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), message)
	}
	_, err = logFile.WriteString(message + "\n")
	if err != nil {
		fmt.Printf("Failed to write to log file: %v\n", err)
		return message
	}

	// Also log to the console
	fmt.Printf("%s: %s\n", name, message)
	return message
}

func logToFile(logType string, message string, subApp *SubApplication, logFlags ...bool) {
	func() {
		var location string
		var app string = "daemon"
		if subApp != nil {
			app = subApp.Id
		}

		location, err := getLogLocation(app, subApp)
		if err == nil {
			if subApp != nil {
				if subApp.LogLocation != location {
					subApp.LogLocation = location

				}
				if subApp.LogLocation == "" {
					return
				}
			}
			msg := logMessage(location, app, logType, message)
			if logType != "log" {
				msg = message
			}
			broadcastToSocket(logType, LogEvent{Message: msg, AppId: app})

		}
		if logFlags != nil && logFlags[0] {

			if subApp != nil {
				message = fmt.Sprintf("[%s] %s", subApp.Id, message)
			}
			logToMainFile(message)
		}
	}()

}

func logToMainFile(message string) {
	logToFile("log", message, nil, false)
}
