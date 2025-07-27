package main

import (
	"encoding/json"
	"log"
	"net/http"
	"ws-server/data"
	"ws-server/utils"

	"github.com/gorilla/websocket"
)

var (
	humidityHub    = utils.NewHub()
	temperatureHub = utils.NewHub()

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func main() {
	go humidityHub.Run()
	go temperatureHub.Run()

	http.HandleFunc("/ws/humidity-stats", func(w http.ResponseWriter, r *http.Request) {
		handleConnections(humidityHub, w, r)
	})

	http.HandleFunc("/ws/temperature-stats", func(w http.ResponseWriter, r *http.Request) {
		handleConnections(temperatureHub, w, r)
	})

	log.Println("Servidor WebSocket iniciado en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleConnections(hub *utils.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}

	client := &utils.Client{
		Conn: conn,
		Send: make(chan []byte, 256),
		Hub:  hub,
	}
	client.Hub.Register <- client

	go client.WritePump()
	client.ReadPump(func(message []byte, sender *utils.Client) {
		var data data.FrecuenciaData
		if err := json.Unmarshal(message, &data); err != nil {
			log.Printf("Mensaje inválido: %v", err)
			return
		}

		log.Printf("FrecuenciaData recibido: %+v", data)
		hub.Broadcast <- message 
	})
}
