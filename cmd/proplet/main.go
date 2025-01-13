package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/absmach/magistrala/pkg/server"
	"github.com/absmach/propeller/pkg/config"
	"github.com/absmach/propeller/pkg/mqtt"
	"github.com/absmach/propeller/proplet"
	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const svcName = "proplet"

type envConfig struct {
	LogLevel           string        `env:"PROPLET_LOG_LEVEL"           envDefault:"info"`
	InstanceID         string        `env:"PROPLET_INSTANCE_ID"`
	MQTTAddress        string        `env:"PROPLET_MQTT_ADDRESS"        envDefault:"tcp://localhost:1883"`
	MQTTTimeout        time.Duration `env:"PROPLET_MQTT_TIMEOUT"        envDefault:"30s"`
	MQTTQoS            byte          `env:"PROPLET_MQTT_QOS"            envDefault:"2"`
	LivelinessInterval time.Duration `env:"PROPLET_LIVELINESS_INTERVAL" envDefault:"10s"`
	ConfigPath         string        `env:"CONFIG_PATH"                 envDefault:"config.toml"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := envConfig{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load configuration : %s", err.Error())
	}

	if cfg.InstanceID == "" {
		cfg.InstanceID = uuid.NewString()
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		log.Fatalf("failed to parse log level: %s", err.Error())
	}
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	conf, err := config.LoadConfig(cfg.ConfigPath)
	if err != nil {
		logger.Error("failed to load TOML configuration", slog.String("error", err.Error()))

		return
	}

	mqttPubSub, err := mqtt.NewPubSub(cfg.MQTTAddress, cfg.MQTTQoS, cfg.InstanceID, conf.Proplet.ThingID, conf.Proplet.ThingKey, conf.Proplet.ChannelID, cfg.MQTTTimeout, logger)
	if err != nil {
		logger.Error("failed to initialize mqtt client", slog.Any("error", err))

		return
	}
	wazero := proplet.NewWazeroRuntime(logger, mqttPubSub, conf.Proplet.ChannelID)

	service, err := proplet.NewService(ctx, conf.Proplet.ChannelID, conf.Proplet.ThingID, conf.Proplet.ThingKey, cfg.LivelinessInterval, mqttPubSub, logger, wazero)
	if err != nil {
		logger.Error("failed to initialize service", slog.Any("error", err))

		return
	}

	if err := service.Run(ctx, logger); err != nil {
		logger.Error("failed to run service", slog.Any("error", err))

		return
	}

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service exited with error: %s", svcName, err))
	}
}
