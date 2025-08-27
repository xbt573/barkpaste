package paste

import (
	"time"

	"github.com/xbt573/barkpaste/internal/models"
	"gorm.io/gorm"
)

type Repository interface {
	Create(paste models.Paste) (models.Paste, error)

	List() ([]models.Paste, error)
	GetByID(id string) (models.Paste, error)

	Update(paste models.Paste) (models.Paste, error)

	Delete(id string) (models.Paste, error)
	CleanExpired() error
}

type concreteRepository struct {
	db *gorm.DB
}

func New(db *gorm.DB) (Repository, error) {
	if err := db.AutoMigrate(&models.Paste{}); err != nil {
		return nil, err
	}

	return &concreteRepository{db}, nil
}

func (c *concreteRepository) Create(paste models.Paste) (models.Paste, error) {
	result := c.db.Create(paste)

	return paste, result.Error
}

func (c *concreteRepository) Delete(id string) (models.Paste, error) {
	var paste models.Paste

	result := c.db.Where("id = ?", id).Delete(&paste)
	if result.RowsAffected == 0 {
		return paste, gorm.ErrRecordNotFound
	}

	return paste, result.Error
}

func (c *concreteRepository) GetByID(id string) (models.Paste, error) {
	var paste models.Paste

	result := c.db.Where("id = ?", id).First(&paste)

	return paste, result.Error
}

func (c *concreteRepository) List() ([]models.Paste, error) {
	var pastes []models.Paste

	result := c.db.Find(&pastes)

	return pastes, result.Error
}

func (c *concreteRepository) Update(paste models.Paste) (models.Paste, error) {
	result := c.db.Model(&paste).Updates(paste)

	return paste, result.Error
}

func (c *concreteRepository) CleanExpired() error {
	result := c.db.Where("expired_at < ? AND is_persistent = ?", time.Now(), false).Delete(&models.Paste{})
	return result.Error
}
