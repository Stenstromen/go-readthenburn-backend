package services

import (
	"database/sql"
	"go-readthenburn-backend/internal/models"
	"go-readthenburn-backend/internal/repository"
	"go-readthenburn-backend/pkg/encryption"

	"github.com/google/uuid"
)

type MessageService struct {
	repo      *repository.MessageRepository
	encryptor *encryption.Encryptor
	db        *sql.DB
}

func NewMessageService(repo *repository.MessageRepository, encryptor *encryption.Encryptor, db *sql.DB) *MessageService {
	return &MessageService{
		repo:      repo,
		encryptor: encryptor,
		db:        db,
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

	if err := s.repo.CreateMessage(msg); err != nil {
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

func (s *MessageService) GetDB() *sql.DB {
	return s.db
}
