package repository

import (
	"database/sql"
	"fmt"
	"go-readthenburn-backend/internal/models"
	"log"
	"os"
	"time"
)

type MessageRepository struct {
	db *sql.DB
}

func (r *MessageRepository) GetCurrentTime() time.Time {
	if dateStr := os.Getenv("CURRENT_DATE"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			log.Printf("Using date from environment: %s", t.Format("2006-01-02"))
			return t
		}
		log.Printf("Failed to parse CURRENT_DATE: %v", dateStr)
	}
	return time.Now()
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) IncrementTotalMessages() error {
	date := r.GetCurrentTime().Format("2006-01-02")
	_, err := r.db.Exec(`
        INSERT INTO stats (date, totalMessages) 
        VALUES(?, 1) 
        ON DUPLICATE KEY UPDATE totalMessages = totalMessages + 1`,
		date)
	return err
}

func (r *MessageRepository) CreateMessage(msg *models.Message) error {
	stmt, err := r.db.Prepare("INSERT INTO burntable (messageId, messageEnc, messageIv) VALUES(?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(msg.ID, msg.Encrypted, msg.IV)
	return err
}

func (r *MessageRepository) GetMessage(id string) (*models.Message, error) {
	msg := &models.Message{}
	err := r.db.QueryRow("SELECT messageId, messageEnc, messageIv FROM burntable WHERE messageId=?", id).
		Scan(&msg.ID, &msg.Encrypted, &msg.IV)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (r *MessageRepository) GetTotalMessages(dbName string) (int, error) {
	var count int
	err := r.db.QueryRow(fmt.Sprintf("SELECT AUTO_INCREMENT-1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = 'burntable'", dbName)).Scan(&count)
	if err != nil {
		fmt.Println("Error getting total messages:", err)
		return 0, err
	}
	return count, nil
}

func (r *MessageRepository) DeleteMessage(id string) error {
	_, err := r.db.Exec("DELETE FROM burntable WHERE messageId=?", id)
	return err
}

func (r *MessageRepository) GetAllStats() ([]models.DailyStats, error) {
	rows, err := r.db.Query("SELECT date, totalMessages FROM stats ORDER BY date DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.DailyStats
	for rows.Next() {
		var stat models.DailyStats
		var dateStr string
		if err := rows.Scan(&dateStr, &stat.TotalMessages); err != nil {
			return nil, err
		}
		stat.Date = dateStr
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}
