package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xbt573/barkpaste/internal/app"
	pasteController "github.com/xbt573/barkpaste/internal/controller/paste"
	pasteRepository "github.com/xbt573/barkpaste/internal/repository/paste"
	tokenRepository "github.com/xbt573/barkpaste/internal/repository/token"
	pasteService "github.com/xbt573/barkpaste/internal/service/paste"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	config     Config
	configFile string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file (.yaml)")

	rootCmd.PersistentFlags().StringVarP(&config.Listen, "listen", "l", "127.0.0.1:8888", "Host and port to listen on")

	rootCmd.PersistentFlags().StringVar((*string)(&config.Database.Type), "type", string(SQLite), "Database type (one of postgresql sqlite)")
	rootCmd.PersistentFlags().StringVar(&config.Database.URI, "uri", "barkpaste.db", "Database URI (or file for SQLite)")

	rootCmd.PersistentFlags().DurationVar(&config.Settings.TTL, "ttl", time.Hour*24, "TTL of pastes (default to 1d)")
	rootCmd.PersistentFlags().UintVar(&config.Settings.Limit, "limit", 1*1024*1024, "Maximum size of paste (default to 1 MB, uint)")
	rootCmd.PersistentFlags().UintVar(&config.Settings.BodyLimit, "bodylimit", 200*1024*1024, "Maximum size of body (default to 200 MB, uint)")
	// FIXME: поменяй на норм перед релизом, а то засмеют
	rootCmd.PersistentFlags().StringVar(&config.Settings.Token, "token", "verycooltokensir", "Default token (CHANGE TO SECURE)")

	cobra.OnInitialize(func() {
		if configFile != "" {
			viper.SetConfigFile(configFile)
		} else {
			viper.SetConfigName("config")
			viper.SetConfigType("yaml")

			viper.AddConfigPath(".")
			viper.AddConfigPath("~/.config/barkpaste")
			viper.AddConfigPath("/etc/barkpaste")
		}

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				panic(err)
			}
		}

		if err := viper.Unmarshal(&config); err != nil {
			panic(err)
		}
	})
}

var rootCmd = &cobra.Command{
	Use: "barkpaste",
	RunE: func(cmd *cobra.Command, args []string) error {
		var dialector gorm.Dialector

		switch config.Database.Type {
		case SQLite:
			dialector = sqlite.Open(config.Database.URI)
		case PostgreSQL:
			dialector = postgres.Open(config.Database.URI)
		default:
			return fmt.Errorf("unknown database type: %v", config.Database.Type)
		}

		db, err := gorm.Open(dialector, &gorm.Config{
			TranslateError: true,
			Logger:         logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return err
		}

		pr, err := pasteRepository.New(db)
		if err != nil {
			return err
		}

		tr, err := tokenRepository.New(db, tokenRepository.Options{Token: config.Settings.Token})
		if err != nil {
			return err
		}

		ps := pasteService.New(pr, tr, pasteService.Options{
			TTL:   config.Settings.TTL,
			Limit: config.Settings.Limit,
		})

		pc := pasteController.New(ps)

		a := app.New(pc, app.Options{
			BodyLimit: config.Settings.BodyLimit,
		})

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		slog.Info("running on", "addr", config.Listen)
		if err := a.Listen(config.Listen, ctx); err != nil {
			return err
		}

		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
