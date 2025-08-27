package app

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/xbt573/barkpaste/internal/controller/paste"
)

type App struct {
	pasteController paste.Controller
	options         Options
}

type Options struct {
	BodyLimit uint
}

func New(pasteController paste.Controller, opts Options) *App {
	return &App{pasteController, opts}
}

func (a *App) Listen(addr string, ctx context.Context) error {
	f := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             int(a.options.BodyLimit),
	})

	f.Post("/token", a.pasteController.CreateToken)
	f.Delete("/token/:token", a.pasteController.RevokeToken)

	f.Post("/", a.pasteController.CreateRegular)
	f.Post("/:id", a.pasteController.CreatePersistent)
	f.Get("/:id", a.pasteController.Get)
	f.Patch("/:id", a.pasteController.Update)
	f.Delete("/:id", a.pasteController.Delete)

	errch := make(chan error)

	go func() {
		errch <- f.Listen(addr)
	}()

	select {
	case <-ctx.Done():
		return f.Shutdown()
	case err := <-errch:
		if err != nil {
			return err
		}
	}

	return nil
}
