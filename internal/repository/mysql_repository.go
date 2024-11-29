package repository

import (
	"database/sql"
	"fmt"
	"go-readthenburn-backend/internal/models"
	"time"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) IncrementTotalMessages() error {
	date := time.Now().Format("2006-01-02") // YYYY-MM-DD format
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
	fmt.Println("Getting total messages for database:", dbName)
	err := r.db.QueryRow(fmt.Sprintf("SELECT AUTO_INCREMENT FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = 'burntable'", dbName)).Scan(&count)
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
		var date time.Time
		if err := rows.Scan(&date, &stat.TotalMessages); err != nil {
			return nil, err
		}
		stat.Date = date.Format("2006-01-02")
		stats = append(stats, stat)
	}
	fmt.Println("Stats:", stats)
	return stats, rows.Err()
}
