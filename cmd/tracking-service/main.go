package main

import (
	"context"
	"crypto/tls"
	"flag"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"

	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rafimuhammad01/tracking-app/config"
	ihttp "github.com/rafimuhammad01/tracking-app/http"
	ikafka "github.com/rafimuhammad01/tracking-app/kafka"

	"github.com/rafimuhammad01/tracking-app/track"
)

func main() {
	// config
	configPath := flag.String("env_file", "./config/development.yaml", "define the environment file path")
	flag.Parse()
	config.SetFromFile(*configPath)

	// log setup
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if config.Get().Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// app related
	done := make(chan os.Signal, 1)
	ctx := context.Background()
	dep := InitDependency()
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// server definition
	srv := NewHTTPServer(dep)
	consumer := NewKafkaConsumer()

	// start server
	go func() {
		log.Info().Any("port", srv.Addr).Msg("starting http server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start http server")
		}
	}()

	go func() {
		log.Info().Any("topic", consumer.Config().Topic).Msg("starting kafka consumer")
		dep.KafkaHandler.Listen(ctx, consumer)
	}()

	// graceful shutdown
	<-done
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Info().Msg("stopping http server")
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to shutdown http server")
		} else {
			log.Info().Msg("http server stopped")
		}
	}()

	go func() {
		defer wg.Done()
		log.Info().Msg("stopping kafka consumer")
		kafkaClosed := make(chan struct{})
		kafkaClosedErr := make(chan error)
		go func() {
			if err := consumer.Close(); err != nil {
				kafkaClosedErr <- err
			} else {
				kafkaClosed <- struct{}{}
			}

		}()

		select {
		case <-ctx.Done():
			log.Fatal().Err(ctx.Err()).Msg("failed to shutdown kafka consumer")
		case err := <-kafkaClosedErr:
			log.Fatal().Err(err).Msg("failed to shutdown kafka consumer")
		case <-kafkaClosed:
			log.Info().Msg("kafka consumer stopped")

		}
	}()
	wg.Wait()
}

type Dependency struct {
	Hub          *track.Hub
	HTTPHandler  *ihttp.Handler
	KafkaHandler *ikafka.Handler
}

func InitDependency() *Dependency {
	hub := track.NewHub()
	httpHandler := ihttp.NewHandler(hub)
	kafkaHandler := ikafka.NewHandler(hub)

	return &Dependency{
		Hub:          hub,
		HTTPHandler:  httpHandler,
		KafkaHandler: kafkaHandler,
	}
}

func NewHTTPServer(d *Dependency) *http.Server {
	port := ":" + config.Get().HTTP.Port
	srv := &http.Server{
		Addr: port,
	}

	http.HandleFunc("/location", d.HTTPHandler.GetLatestLocation)
	return srv
}

func NewKafkaConsumer() *kafka.Reader {
	mechanism, err := scram.Mechanism(scram.SHA256, config.Get().Kafka.Connection.Username, config.Get().Kafka.Connection.Password)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start kafka consumer")
	}

	dialer := &kafka.Dialer{
		SASLMechanism: mechanism,
		TLS:           &tls.Config{},
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     config.Get().Kafka.Connection.Brokers,
		Topic:       config.Get().Kafka.Consumer.Topic,
		MinBytes:    config.Get().Kafka.Consumer.MinBytes,
		MaxBytes:    config.Get().Kafka.Consumer.MaxBytes,
		Dialer:      dialer,
		StartOffset: kafka.LastOffset,
		GroupID:     uuid.NewString(),
	})

	return r
}
