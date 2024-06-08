package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type ApplicationStatus struct {
	Status          string            `json:"status"`
	SubApplications []*SubApplication `json:"subApplications"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type MessageRequest struct {
	Request   string            `json:"request"`
	RequestId string            `json:"requestId"`
	App       SubApplication    `json:"app"`
	Config    map[string]string `json:"config"`
}

type DeamonStatus struct {
	Name            string            `json:"name"`
	Config          Config            `json:"config"`
	SubApplications []*SubApplication `json:"subApplications"`
}

type ResponseSocket struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast_response = make(chan interface{})

//var broadcast_socket = make(chan []byte) // broadcast channel

func startServer() error {

	http.HandleFunc("/config", changeConfig)
	http.HandleFunc("/flags", listFlags)
	http.HandleFunc("/diskinfo", listDiskSpace)

	http.HandleFunc("/status", apiStatus)
	http.HandleFunc("/app", applicationOperation)
	http.HandleFunc("/applications", listApplications)
	http.HandleFunc("/kits", listKits)
	http.HandleFunc("/ws", wsHandler)

	err := http.ListenAndServe(":8187", nil)
	if err != nil {
		return err
	}
	go broadcastMessages()
	return nil
}

func listKits(w http.ResponseWriter, r *http.Request) {
	obj := getAllKits()
	handleJsonAndError(w, obj, nil)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	clients[conn] = true

	for {
		var msg MessageRequest
		messageType, bytes, err := conn.ReadMessage()
		if err != nil {
			delete(clients, conn)
			break
		}

		if messageType != websocket.TextMessage {
			continue
		}

		err = json.Unmarshal(bytes, &msg)

		if err != nil {
			delete(clients, conn)
			break
		}

		request := strings.ToLower(msg.Request)
		switch request {
		case "stopservice":
			softStopService()
		case "restartservice":
			restartService()
		case "config":
			updateConfigFile(msg.Config)
		case "diskinfo":
			listDiskSpaceInternal()
		case "flags":
			listFlagsInternal(msg.App.Id)
		case "appadd":
			addSubApplication(&msg.App)
		case "appinstall":
			installSubApplication(&msg.App)
		case "appupdate":
			updateSubApplication(&msg.App)
		case "appstart":
			startSubApplication(&msg.App)
		case "appstop":
			stopSubApplication(&msg.App)
		case "apprestart":
			restartSubApplication(&msg.App)
		case "appuninstall":
			uninstallSubApplication(&msg.App)
		case "appconfig":
			modifySubApplication(&msg.App)
		case "appremove":
			removeSubApplication(msg.App.Id)
		case "applist":
			listApplicationsInternal()
		case "status":
			apiStatusInternal()
		case "kits":
			getAllKits()

		}

	}
}

func broadcastToSocket(request string, data interface{}) {
	if data == nil {
		return
	}
	resp := ResponseSocket{Type: request, Data: data}
	// json, err := json.Marshal(resp)
	// if err != nil {
	// 	return
	// }
	broadcast_response <- resp
	//broadcast_socket <- json
}

func broadcastMessages() {
	for {
		msg := <-broadcast_response
		for client := range clients {

			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func changeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var data map[string]string
	err := decoder.Decode(&data)
	if err != nil {
		handleJsonAndError(w, CurrentConfig, nil)
		return
	}
	config := updateConfigFile(data)
	handleJsonAndError(w, config, nil)

}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	status, err := apiStatusInternal()
	handleJsonAndError(w, status, err)
}

func handleJsonAndError(w http.ResponseWriter, obj interface{}, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json, err := json.Marshal(obj)
	if err != nil {
		http.Error(w, "Error encoding json", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func listDiskSpace(w http.ResponseWriter, r *http.Request) {
	obj, err := listDiskSpaceInternal()
	handleJsonAndError(w, obj, err)
}

func listApplications(w http.ResponseWriter, r *http.Request) {
	obj := listApplicationsInternal()
	handleJsonAndError(w, obj, nil)
}

func listFlags(w http.ResponseWriter, r *http.Request) {
	application := r.URL.Query().Get("application")
	obj, err := listFlagsInternal(application)
	handleJsonAndError(w, obj, err)
}

func applicationOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "PUT" && r.Method != "DELETE" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var data SubApplication
	err := decoder.Decode(&data)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var operation string = r.Method
	appStatus, err := appRequestHandlerInternal(data, operation)

	handleJsonAndError(w, appStatus, err)
}
