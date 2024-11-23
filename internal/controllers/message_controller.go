package controllers

import (
	"encoding/json"
	"go-readthenburn-backend/internal/config"
	"go-readthenburn-backend/internal/models"
	"go-readthenburn-backend/internal/services"
	"net"
	"net/http"
	"time"
)

type MessageController struct {
	service *services.MessageService
	config  *config.Config
}

func NewMessageController(service *services.MessageService, config *config.Config) *MessageController {
	return &MessageController{
		service: service,
		config:  config,
	}
}

func (c *MessageController) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Message) >= 121 {
		resp := models.MessageResponse{Error: "Message length exceeded"}
		writeJSON(w, resp, http.StatusBadRequest)
		return
	}

	msgID, err := c.service.CreateMessage(req.Message)
	if err != nil {
		resp := models.MessageResponse{Error: "Failed to create message"}
		writeJSON(w, resp, http.StatusInternalServerError)
		return
	}

	resp := models.MessageResponse{MsgID: msgID}
	writeJSON(w, resp, http.StatusCreated)
}

func (c *MessageController) HandleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[1:] // Remove leading slash
	if len(id) >= 38 {
		resp := models.MessageResponse{Error: "msgId length exceeded"}
		writeJSON(w, resp, http.StatusBadRequest)
		return
	}

	msg, err := c.service.ReadAndBurnMessage(id)
	if err != nil {
		resp := models.MessageResponse{BurnMsg: "Message does not exist or has been burned already"}
		writeJSON(w, resp, http.StatusOK)
		return
	}

	resp := models.MessageResponse{BurnMsg: msg}
	writeJSON(w, resp, http.StatusOK)
}

func (c *MessageController) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle health check endpoints
		if r.URL.Path == "/ready" {
			c.handleReadiness(w, r)
			return
		}
		if r.URL.Path == "/status" {
			c.handleLiveness(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Handle OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (c *MessageController) handleReadiness(w http.ResponseWriter, _ *http.Request) {
	conn := c.service.GetDB()
	err := conn.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *MessageController) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", c.config.DBHost+":3306", timeout)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer conn.Close()
	w.WriteHeader(http.StatusOK)
}
