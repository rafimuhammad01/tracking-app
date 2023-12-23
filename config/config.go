package config

import (
	"github.com/rs/zerolog/log"

	"github.com/spf13/viper"
)

var (
	conf *Config
)

type (
	Config struct {
		Kafka Kafka `mapstructure:"kafka"`
		HTTP  HTTP  `mapstructure:"http"`
		Debug bool  `mapstructure:"debug"`
	}

	HTTP struct {
		DriverPort  string `mapstructure:"driver_port"`
		TrackerPort string `mapstructure:"tracker_port"`
	}

	Kafka struct {
		Connection KafkaConnection `mapstructure:"connection"`
		Consumer   KafkaConsumer   `mapstructure:"consumer"`
	}

	KafkaConnection struct {
		Brokers  []string `mapstructure:"brokers"`
		Username string   `mapstructure:"username"`
		Password string   `mapstructure:"password"`
	}

	KafkaConsumer struct {
		MinBytes int    `mapstructure:"min_bytes"`
		MaxBytes int    `mapstructure:"max_bytes"`
		Topic    string `mapstructure:"topic"`
	}
)

func Get() *Config {
	return conf
}

func SetFromFile(path string) {
	viper.SetConfigType("yaml")
	viper.SetConfigFile(path)
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {
		log.Fatal().Err(err).Msg("fatal when reading config file")
	}

	var configFromViper Config
	err = viper.Unmarshal(&configFromViper)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to decode into config struct")
	}

	conf = &configFromViper
}
