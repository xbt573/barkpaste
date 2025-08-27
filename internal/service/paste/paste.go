package paste

import (
	"errors"
	"time"

	"github.com/xbt573/barkpaste/internal/models"
	"github.com/xbt573/barkpaste/internal/repository/paste"
	"github.com/xbt573/barkpaste/internal/repository/token"
	"gorm.io/gorm"

	nanoid "github.com/matoous/go-nanoid/v2"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrExists         = errors.New("already exists")
	ErrTooBig         = errors.New("too big")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInvalidRequest = errors.New("invalid request")
)

type Service interface {
	// token == "" is fine
	CreateRegular(token string, content []byte, userTTL time.Duration) (models.Paste, error)
	CreatePersistent(token, name string, content []byte, userTTL time.Duration) (models.Paste, error)

	Get(id string) (models.Paste, error)

	Update(token string, paste models.Paste) (models.Paste, error)

	Delete(token, id string) (models.Paste, error)
	CleanExpired() error

	CreateToken(token string) (string, error)
	RevokeToken(accessToken, toRevokeToken string) error

	TTL() time.Duration
	Limit() uint
}

type Options struct {
	TTL   time.Duration
	Limit uint
}

type concreteService struct {
	pasteRepository paste.Repository
	tokenRepository token.Repository

	options Options
}

func New(pasteRepository paste.Repository, tokenRepository token.Repository, options Options) Service {
	return &concreteService{pasteRepository, tokenRepository, options}
}

func (c *concreteService) TTL() time.Duration {
	return c.options.TTL
}

func (c *concreteService) Limit() uint {
	return c.options.Limit
}

func (c *concreteService) CleanExpired() error {
	return c.pasteRepository.CleanExpired()
}

// TODO: (regular) content limit does not apply to named
// NOTE: CreatePersistent allows TTL == 0
func (c *concreteService) CreatePersistent(token string, name string, content []byte, userTTL time.Duration) (models.Paste, error) {
	exists, err := c.tokenRepository.Exists(token)
	if !exists {
		return models.Paste{}, ErrUnauthorized
	}

	if err != nil {
		return models.Paste{}, err
	}

	// TODO: да, почини эту хуйню, ещё вспомни завтра что надо было чинить
	// if len(content) > int(c.options.Limit) {
	// 	return models.Paste{}, ErrTooBig
	// }

	if len(content) < 1 {
		return models.Paste{}, ErrInvalidRequest
	}

	expires := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	if userTTL > 0 {
		expires = time.Now().Add(userTTL)
	}

	paste := models.Paste{
		ID:           name,
		Content:      content,
		IsPersistent: true,
		ExpiredAt:    expires,
	}

	paste, err = c.pasteRepository.Create(paste)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			err = ErrExists
		}

		return models.Paste{}, err
	}

	return paste, nil
}

func (c *concreteService) CreateRegular(token string, content []byte, userTTL time.Duration) (models.Paste, error) {
	authorized, err := c.tokenRepository.Exists(token)
	if err != nil {
		return models.Paste{}, err
	}

	if token != "" && !authorized {
		return models.Paste{}, ErrUnauthorized
	}

	if len(content) > int(c.options.Limit) && !authorized {
		return models.Paste{}, ErrTooBig
	}

	if len(content) < 1 {
		return models.Paste{}, ErrInvalidRequest
	}

	if c.options.TTL == 0 {
		c.options.TTL = time.Hour * 24
	}

	var ttl time.Duration

	if authorized {
		ttl = userTTL
	} else {
		ttl = min(userTTL, c.options.TTL)
	}

	paste := models.Paste{
		ID:        nanoid.Must(8),
		Content:   content,
		ExpiredAt: time.Now().Add(ttl),
	}

	paste, err = c.pasteRepository.Create(paste)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			err = ErrExists
		}

		return models.Paste{}, err
	}

	return paste, nil
}

func (c *concreteService) CreateToken(token string) (string, error) {
	exists, err := c.tokenRepository.Exists(token)
	if !exists {
		return "", ErrUnauthorized
	}

	if err != nil {
		return "", err
	}

	nt, err := c.tokenRepository.Create(models.Token{Token: nanoid.Must(32)})
	if err != nil {
		return "", err
	}

	return nt.Token, nil
}

func (c *concreteService) Delete(token string, id string) (models.Paste, error) {
	exists, err := c.tokenRepository.Exists(token)
	if !exists {
		return models.Paste{}, ErrUnauthorized
	}

	if err != nil {
		return models.Paste{}, err
	}

	paste, err := c.pasteRepository.Delete(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrNotFound
		}

		return models.Paste{}, err
	}

	return paste, nil
}

func (c *concreteService) Get(id string) (models.Paste, error) {
	paste, err := c.pasteRepository.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Paste{}, ErrNotFound
		}

		return models.Paste{}, err
	}

	return paste, nil
}

func (c *concreteService) RevokeToken(accessToken string, toRevokeToken string) error {
	exists, err := c.tokenRepository.Exists(accessToken)
	if !exists {
		return ErrUnauthorized
	}

	if err != nil {
		return err
	}

	err = c.tokenRepository.Delete(toRevokeToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}

		return err
	}

	return nil
}

func (c *concreteService) Update(token string, paste models.Paste) (models.Paste, error) {
	exists, err := c.tokenRepository.Exists(token)
	if !exists {
		return models.Paste{}, ErrUnauthorized
	}

	if err != nil {
		return models.Paste{}, err
	}

	paste, err = c.pasteRepository.Update(paste)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrNotFound
		}

		return models.Paste{}, err
	}

	return paste, nil
}
