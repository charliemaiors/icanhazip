package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port int    `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server"`
	Results struct {
		IncludePrivate bool `mapstructure:"include_private"`
	} `mapstructure:"results"`
}

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
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
	}
	return config, nil
}
