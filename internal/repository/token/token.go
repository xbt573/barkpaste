package token

import (
	"log/slog"

	"github.com/xbt573/barkpaste/internal/models"
	"gorm.io/gorm"
)

type Repository interface {
	Create(token models.Token) (models.Token, error)
	List() ([]models.Token, error)
	Delete(token string) error
	Exists(token string) (bool, error)
}

type Options struct {
	Token string
}

type concreteRepository struct {
	db *gorm.DB
}

func New(db *gorm.DB, opts Options) (Repository, error) {
	if err := db.AutoMigrate(&models.Token{}); err != nil {
		return nil, err
	}

	var count int64
	if err := db.Model(&models.Token{}).Count(&count).Error; err != nil {
		return nil, err
	}

	if count == 0 {
		if err := db.Create(&models.Token{Token: opts.Token}).Error; err != nil {
			return nil, err
		}

		slog.Info("created default token from config, please replace it")
	}

	return &concreteRepository{db}, nil
}

func (r *concreteRepository) Create(token models.Token) (models.Token, error) {
	result := r.db.Create(&token)

	return token, result.Error
}

func (r *concreteRepository) List() ([]models.Token, error) {
	var tokens []models.Token

	result := r.db.Find(&tokens)

	return tokens, result.Error
}

func (r *concreteRepository) Delete(token string) error {
	result := r.db.Delete(&models.Token{}, "token = ?", token)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return result.Error
}

func (r *concreteRepository) Exists(token string) (bool, error) {
	var count int64

	result := r.db.Model(&models.Token{}).Where("token = ?", token).Count(&count)

	return count > 0, result.Error
}
