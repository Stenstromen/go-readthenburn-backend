package services

import (
	"database/sql"
	"fmt"
	"go-readthenburn-backend/internal/config"
	"go-readthenburn-backend/internal/models"
	"go-readthenburn-backend/internal/repository"
	"go-readthenburn-backend/pkg/encryption"

	"github.com/google/uuid"
)

type MessageService struct {
	repo      *repository.MessageRepository
	encryptor *encryption.Encryptor
	db        *sql.DB
	config    *config.Config
}

func NewMessageService(repo *repository.MessageRepository, encryptor *encryption.Encryptor, db *sql.DB, config *config.Config) *MessageService {
	return &MessageService{
		repo:      repo,
		encryptor: encryptor,
		db:        db,
		config:    config,
	}
}

func (s *MessageService) CreateMessage(content string) (string, error) {
	encrypted, iv, err := s.encryptor.Encrypt(content)
	if err != nil {
		return "", err
	}

	msg := &models.Message{
		ID:        uuid.New().String(),
		Content:   content,
		Encrypted: encrypted,
		IV:        iv,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if err := s.repo.CreateMessage(msg); err != nil {
		return "", err
	}

	if err := s.repo.IncrementTotalMessages(); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return msg.ID, nil
}

func (s *MessageService) ReadAndBurnMessage(id string) (string, error) {
	msg, err := s.repo.GetMessage(id)
	if err != nil {
		return "", err
	}

	content, err := s.encryptor.Decrypt(msg.Encrypted, msg.IV)
	if err != nil {
		return "", err
	}

	if err := s.repo.DeleteMessage(id); err != nil {
		return "", err
	}

	return content, nil
}

func (s *MessageService) GetTotalMessages() (int, error) {
	return s.repo.GetTotalMessages(s.config.DBName)
}

func (s *MessageService) GetStats() (*models.StatsResponse, error) {
	total, err := s.repo.GetTotalMessages(s.config.DBName)
	if err != nil {
		fmt.Printf("Error getting total messages: %v\n", err)
		return nil, fmt.Errorf("failed to get total messages: %w", err)
	}

	history, err := s.repo.GetAllStats()
	if err != nil {
		fmt.Printf("Error getting message history: %v\n", err)
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	return &models.StatsResponse{
		TotalMessages: total,
		History:       history,
	}, nil
}

func (s *MessageService) GetDB() *sql.DB {
	return s.db
}
