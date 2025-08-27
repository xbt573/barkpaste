package paste

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	pasteService "github.com/xbt573/barkpaste/internal/service/paste"
	"golang.org/x/net/idna"
)

type Controller interface {
	CreateRegular(ctx *fiber.Ctx) error
	CreatePersistent(ctx *fiber.Ctx) error

	Get(ctx *fiber.Ctx) error

	Update(ctx *fiber.Ctx) error

	Delete(ctx *fiber.Ctx) error

	CreateToken(ctx *fiber.Ctx) error
	RevokeToken(ctx *fiber.Ctx) error
}

type concreteController struct {
	pasteService pasteService.Service
}

func New(pasteService pasteService.Service) Controller {
	return &concreteController{pasteService}
}

func (c *concreteController) CreateRegular(ctx *fiber.Ctx) error {
	now := time.Now()
	body := ctx.Body()

	ttl := c.pasteService.TTL()

	if header := ctx.Get("X-Expires-After"); header != "" {
		num, err := strconv.Atoi(header)
		if err != nil {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		ttl = time.Second * time.Duration(num)
	}

	if header := ctx.Get("X-Expires-At"); header != "" {
		t, err := time.Parse(time.RFC3339, header)
		if err != nil {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		ttl = t.Sub(now)
	}

	paste, err := c.pasteService.CreateRegular(body, ttl)
	if err != nil {
		if errors.Is(err, pasteService.ErrExists) {
			return ctx.Status(fiber.StatusConflict).SendString(
				"this paste already exists, but should not. consider this your lucky day! :D",
			)
		}

		if errors.Is(err, pasteService.ErrInvalidRequest) {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		if errors.Is(err, pasteService.ErrTooBig) {
			return ctx.SendStatus(fiber.StatusRequestEntityTooLarge)
		}

		slog.Error("internal error", "err", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	scheme := "http"
	if ctx.Protocol() == "https" {
		scheme = "https"
	}

	host := ctx.Hostname()

	url := fmt.Sprintf("%v://%v/%v", scheme, host, paste.ID)

	ctx.Set("X-Expires-At", paste.ExpiredAt.Format(time.RFC3339))
	ctx.Set("Content-Location", "/"+paste.ID)

	return ctx.Status(fiber.StatusCreated).SendString(url)
}

func (c *concreteController) CreatePersistent(ctx *fiber.Ctx) error {
	token := ""

	rawToken := ctx.Get("Authorization")
	if rawToken != "" {
		fmt.Sscanf(rawToken, "Bearer %v", &token)
	}

	id := ctx.Params("id")

	now := time.Now()
	body := ctx.Body()

	ttl := time.Duration(0)

	if header := ctx.Get("X-Expires-After"); header != "" {
		num, err := strconv.Atoi(header)
		if err != nil {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		ttl = time.Second * time.Duration(num)
	}

	if header := ctx.Get("X-Expires-At"); header != "" {
		t, err := time.Parse(time.RFC3339, header)
		if err != nil {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		ttl = t.Sub(now)
	}

	paste, err := c.pasteService.CreatePersistent(token, id, body, ttl)
	if err != nil {
		if errors.Is(err, pasteService.ErrExists) {
			return ctx.Status(fiber.StatusConflict).SendString(
				"this paste already exists, but should not. consider this your lucky day! :D",
			)
		}

		if errors.Is(err, pasteService.ErrInvalidRequest) {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		if errors.Is(err, pasteService.ErrTooBig) {
			return ctx.SendStatus(fiber.StatusRequestEntityTooLarge)
		}

		if errors.Is(err, pasteService.ErrUnauthorized) {
			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		slog.Error("internal error", "err", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	scheme := "http"
	if ctx.Protocol() == "https" {
		scheme = "https"
	}

	host := ctx.Hostname()

	url := fmt.Sprintf("%v://%v/%v", scheme, host, paste.ID)

	url, err = idna.ToUnicode(url)
	if err != nil {
		slog.Error("shouldn't happen", "err", err)
	}

	ctx.Set("X-Expires-At", paste.ExpiredAt.Format(time.RFC3339))
	ctx.Set("Content-Location", "/"+paste.ID)

	return ctx.Status(fiber.StatusCreated).SendString(url)
}

// TODO: сделать лимит выше для токенизированных блядей
func (c *concreteController) Get(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	paste, err := c.pasteService.Get(id)
	if err != nil {
		if errors.Is(err, pasteService.ErrNotFound) {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		slog.Error("internal error", "err", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	if time.Now().After(paste.ExpiredAt) {
		// FIXME: крон
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	ctx.Set("X-Expires-At", paste.ExpiredAt.Format(time.RFC3339))

	_, err = ctx.Write(paste.Content)
	if err != nil {
		return err
	}

	return nil
}

func (c *concreteController) Update(ctx *fiber.Ctx) error {
	token := ""

	rawToken := ctx.Get("Authorization")
	if rawToken != "" {
		fmt.Sscanf(rawToken, "Bearer %v", &token)
	}

	id := ctx.Params("id")

	now := time.Now()
	body := ctx.Body()

	ttl := time.Duration(0)

	if header := ctx.Get("X-Expires-After"); header != "" {
		num, err := strconv.Atoi(header)
		if err != nil {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		ttl = time.Second * time.Duration(num)
	}

	if header := ctx.Get("X-Expires-At"); header != "" {
		t, err := time.Parse(time.RFC3339, header)
		if err != nil {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		ttl = t.Sub(now)
	}

	paste, err := c.pasteService.Get(id)
	if err != nil {
		if errors.Is(err, pasteService.ErrInvalidRequest) {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		if errors.Is(err, pasteService.ErrTooBig) {
			return ctx.SendStatus(fiber.StatusRequestEntityTooLarge)
		}

		slog.Error("internal error", "err", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	if time.Now().After(paste.ExpiredAt) {
		// FIXME: крон
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	if len(body) > 0 {
		paste.Content = body
	}

	if ttl > 0 {
		paste.ExpiredAt = time.Now().Add(ttl)
	}

	paste, err = c.pasteService.Update(token, paste)
	if err != nil {
		if errors.Is(err, pasteService.ErrInvalidRequest) {
			return ctx.SendStatus(fiber.StatusBadRequest)
		}

		if errors.Is(err, pasteService.ErrTooBig) {
			return ctx.SendStatus(fiber.StatusRequestEntityTooLarge)
		}

		if errors.Is(err, pasteService.ErrUnauthorized) {
			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		slog.Error("internal error", "err", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	ctx.Set("X-Expires-At", paste.ExpiredAt.Format(time.RFC3339))
	return nil
}

func (c *concreteController) Delete(ctx *fiber.Ctx) error {
	token := ""

	rawToken := ctx.Get("Authorization")
	if rawToken != "" {
		fmt.Sscanf(rawToken, "Bearer %v", &token)
	}

	id := ctx.Params("id")

	_, err := c.pasteService.Delete(token, id)
	if err != nil {
		if errors.Is(err, pasteService.ErrNotFound) {
			return ctx.SendStatus(fiber.StatusNotFound)
		}

		if errors.Is(err, pasteService.ErrUnauthorized) {
			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		slog.Error("internal error", "err", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return nil
}

func (c *concreteController) CreateToken(ctx *fiber.Ctx) error {
	accessToken := ""

	rawToken := ctx.Get("Authorization")
	if rawToken != "" {
		fmt.Sscanf(rawToken, "Bearer %v", &accessToken)
	}

	token, err := c.pasteService.CreateToken(accessToken)
	if err != nil {
		if errors.Is(err, pasteService.ErrUnauthorized) {
			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		slog.Error("internal error", "err", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	_, err = ctx.WriteString(token)
	return err
}

func (c *concreteController) RevokeToken(ctx *fiber.Ctx) error {
	accessToken := ""

	rawToken := ctx.Get("Authorization")
	if rawToken != "" {
		fmt.Sscanf(rawToken, "Bearer %v", &accessToken)
	}

	token := ctx.Params("token")

	err := c.pasteService.RevokeToken(accessToken, token)
	if err != nil {
		if errors.Is(err, pasteService.ErrUnauthorized) {
			return ctx.SendStatus(fiber.StatusUnauthorized)
		}

		slog.Error("internal error", "err", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return nil
}
