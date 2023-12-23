package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rafimuhammad01/tracking-app/config"
	ihttp "github.com/rafimuhammad01/tracking-app/http"
	ikafka "github.com/rafimuhammad01/tracking-app/kafka"
	"github.com/rafimuhammad01/tracking-app/track"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
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

	// start server
	go func() {
		log.Info().Any("port", srv.Addr).Msg("starting http server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start http server")
		}
	}()

	// graceful shutdown
	<-done
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("stopping http server")
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to shutdown http server")
		} else {
			log.Info().Msg("http server stopped")
		}
	}()
	wg.Wait()
}

type Dependency struct {
	Tracker      *track.Tracker
	HTTPHandler  *ihttp.TrackingHandler
	KafkaTracker *ikafka.Tracker
}

func InitDependency() *Dependency {
	writer := NewKafkaWriter()
	kafkaTracker := ikafka.NewTracker(ikafka.WithWriter(writer))

	tracker := track.NewTracker(track.WithSender(kafkaTracker))
	httpHandler := ihttp.NewHandler(tracker)

	return &Dependency{
		Tracker:      tracker,
		HTTPHandler:  httpHandler,
		KafkaTracker: kafkaTracker,
	}
}

func NewHTTPServer(d *Dependency) *http.Server {
	port := ":" + config.Get().HTTP.DriverPort
	srv := &http.Server{
		Addr: port,
	}

	http.HandleFunc("/location", d.HTTPHandler.SendLocation)
	return srv
}

func NewKafkaWriter() *kafka.Writer {
	mechanism, err := scram.Mechanism(scram.SHA256, config.Get().Kafka.Connection.Username, config.Get().Kafka.Connection.Password)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start kafka consumer")
	}

	dialer := &kafka.Dialer{
		SASLMechanism: mechanism,
		TLS:           &tls.Config{},
	}

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  config.Get().Kafka.Connection.Brokers,
		Topic:    config.Get().Kafka.Consumer.Topic,
		Balancer: &kafka.Hash{},
		Dialer:   dialer,
	})

	return w
}
