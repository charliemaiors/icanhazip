package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Server  Server `mapstructure:"server"`
	Results Result `mapstructure:"results"`
}

type Server struct {
	Port int    `mapstructure:"port",omitempty`
	Host string `mapstructure:"host",omitempty`
}

type Result struct {
	IncludePrivate bool `mapstructure:"include_private"`
}

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("~/.config/icanhazip/")
	viper.AddConfigPath("/etc/icanhazip/")
}

func loadConfig() (Config, error) {
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}
	config := Config{}
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Error unmarshaling config: %s", err)
		config = defaultConfig()
	}
	return config, nil
}

func defaultConfig() Config {
	return Config{
		Server: Server{
			Port: 8091,
			Host: "",
		},
		Results: Result{
			IncludePrivate: true,
		},
	}
}
