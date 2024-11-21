package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"go-readthenburn-backend/internal/config"
	"go-readthenburn-backend/internal/controllers"
	"go-readthenburn-backend/internal/repository"
	"go-readthenburn-backend/internal/services"
	"go-readthenburn-backend/pkg/encryption"
	"log"
	"net/http"
)

func initDB(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open(cfg.DBDriver,
		cfg.DBUser+":"+cfg.DBPass+"@tcp("+cfg.DBHost+")/"+cfg.DBName)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS burntable (
            id INT AUTO_INCREMENT PRIMARY KEY,
            messageId VARCHAR(255),
            messageEnc VARCHAR(255),
            messageIv VARCHAR(255)
        )
    `)
	return db, err
}

func main() {
	cfg := config.LoadConfig()

	// Initialize database
	db, err := initDB(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize components
	encryptor := encryption.NewEncryptor([]byte(cfg.SecretKey))
	repo := repository.NewMessageRepository(db)
	service := services.NewMessageService(repo, encryptor, db)
	controller := controllers.NewMessageController(service, cfg)

	// Setup routes
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/ready", controller.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	mux.HandleFunc("/status", controller.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))

	// API endpoints
	mux.HandleFunc("/", controller.Middleware(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			controller.HandleCreate(w, r)
		} else {
			controller.HandleGet(w, r)
		}
	}))

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
