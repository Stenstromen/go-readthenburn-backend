package repository

import (
	"database/sql"
	"go-readthenburn-backend/internal/models"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
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

func (r *MessageRepository) DeleteMessage(id string) error {
	_, err := r.db.Exec("DELETE FROM burntable WHERE messageId=?", id)
	return err
}
