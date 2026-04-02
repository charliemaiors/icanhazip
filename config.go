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
	TLS  *TLS   `mapstructure:"tls",omitempty`
}

type TLS struct {
	CertFile string `mapstructure:"cert_file",omitempty`
	KeyFile  string `mapstructure:"key_file",omitempty`
	Acme     *Acme  `mapstructure:"acme",omitempty`
}

type Acme struct {
	Email            string   `mapstructure:"email",omitempty`
	Domains          []string `mapstructure:"domains",omitempty`
	AcmeDirectoryURL string   `mapstructure:"acme_directory_url",omitempty`
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

func LoadConfig(flagConfig string) (Config, error) {

	if flagConfig != "" {
		viper.SetConfigFile(flagConfig)
	}

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
