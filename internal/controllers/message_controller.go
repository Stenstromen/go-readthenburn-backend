package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-readthenburn-backend/internal/config"
	"go-readthenburn-backend/internal/models"
	"go-readthenburn-backend/internal/services"
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

func (c *MessageController) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := c.service.GetStats()
	if err != nil {
		fmt.Println("Error getting stats:", err)
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}
	writeJSON(w, stats, http.StatusOK)
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
		if r.URL.Path == "/stats" {
			c.HandleStats(w, r)
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
	if c.config.DBUser == "" || c.config.DBPass == "" || c.config.DBName == "" || c.config.DBHost == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("MySQL configuration not available"))
		return
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", c.config.DBUser, c.config.DBPass, c.config.DBHost, c.config.DBName)

	db, err := sql.Open(c.config.DBDriver, dsn)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("MySQL server connection failed"))
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("MySQL server is not responding"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("MySQL server is healthy"))
}
