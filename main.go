package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"ws-server/data"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	mu     sync.Mutex
	closed bool
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	vaccineHub = &Hub{make(map[*Client]bool), make(chan []byte), make(chan *Client), make(chan *Client), sync.RWMutex{}}
	alcoholHub = &Hub{make(map[*Client]bool), make(chan []byte), make(chan *Client), make(chan *Client), sync.RWMutex{}}
	humidityHub = &Hub{make(map[*Client]bool), make(chan []byte), make(chan *Client), make(chan *Client), sync.RWMutex{}}
	temperatureHub = &Hub{make(map[*Client]bool), make(chan []byte), make(chan *Client), make(chan *Client), sync.RWMutex{}}
)

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error escribiendo mensaje: %v", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump(messageHandler func([]byte, *Client)) {
	defer func() {
		c.hub.unregister <- c
		c.close()
	}()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error WebSocket: %v", err)
			}
			break
		}

		if messageType != websocket.TextMessage {
			continue
		}

		messageHandler(message, c)
	}
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
		c.conn.Close()
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Cliente conectado. Total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				h.mu.Unlock()
				client.close()
				log.Printf("Cliente desconectado. Total: %d", len(h.clients))
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					client.close()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func handleVaccineMessage(message []byte, sender *Client) {
	var vr data.VaccinationResponse
	if err := json.Unmarshal(message, &vr); err != nil {
		log.Printf("Mensaje inválido (vacuna): %v", err)
		return
	}

	log.Printf("VaccinationResponse recibido: %+v", vr)

	vaccineHub.mu.Lock()
	for client := range vaccineHub.clients {
		select {
		case client.send <- message:
		default:
			delete(vaccineHub.clients, client)
			client.close()
		}
	}
	vaccineHub.mu.Unlock()
}


func handleAlcoholMessage(message []byte, sender *Client) {
	var alcoholData []data.AlcoholData
	if err := json.Unmarshal(message, &alcoholData); err != nil {
		log.Printf("Mensaje inválido (alcohol): %v", err)
		return
	}

	log.Printf("AlcoholData recibido: %+v", alcoholData)

	alcoholHub.mu.RLock()
	for client := range alcoholHub.clients {
		select {
		case client.send <- message:
		default:
			delete(alcoholHub.clients, client)
			client.close()
		}
	}
	alcoholHub.mu.RUnlock()
}

func handleHumidityMessage(message []byte, sender *Client) {
	log.Printf("HumidityData recibido: %s", string(message))
	humidityHub.mu.RLock()
	for client := range humidityHub.clients {
		select {
		case client.send <- message:
		default:
			delete(humidityHub.clients, client)
			client.close()
		}
	}
	humidityHub.mu.RUnlock()
}

func handleTemperatureMessage(message []byte, sender *Client) {
	log.Printf("TemperatureData recibido: %s", string(message))
	temperatureHub.mu.RLock()
	for client := range temperatureHub.clients {
		select {
		case client.send <- message:
		default:
			delete(temperatureHub.clients, client)
			client.close()
		}
	}
	temperatureHub.mu.RUnlock()
}

func handleVaccineStatsWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Error al hacer upgrade: %v", err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte, 256), hub: vaccineHub}
	client.hub.register <- client
	go client.writePump()
	go client.readPump(handleVaccineMessage)
}

func handleAlcoholStatsWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Error al hacer upgrade: %v", err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte, 256), hub: alcoholHub}
	client.hub.register <- client
	go client.writePump()
	go client.readPump(handleAlcoholMessage)
}

func handleHumidityStatsWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Error al hacer upgrade: %v", err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte, 256), hub: humidityHub}
	client.hub.register <- client
	go client.writePump()
	go client.readPump(handleHumidityMessage)
}

func handleTemperatureStatsWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Error al hacer upgrade: %v", err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte, 256), hub: temperatureHub}
	client.hub.register <- client
	go client.writePump()
	go client.readPump(handleTemperatureMessage)
}

func main() {
	go vaccineHub.run()
	go alcoholHub.run()
	go humidityHub.run()
	go temperatureHub.run()

	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	router.Use(cors.New(config))

	router.GET("/ws/vaccine-stats", handleVaccineStatsWS)
	router.GET("/ws/alcohol-stats", handleAlcoholStatsWS)
	router.GET("/ws/humidity-stats", handleHumidityStatsWS)
	router.GET("/ws/temperature-stats", handleTemperatureStatsWS)

	fmt.Println("Servidor WebSocket escuchando en puerto 8080")
	log.Fatal(router.Run(":8080"))
}
