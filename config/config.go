package config

import (
	"errors"
	"fmt"
	"os"
)

// Config is a struct representing the application's configuration.
type Config struct {
	infuraKey  string
	network    string
	url        string
	addr       string
	logLevel   string
}

var	cfg Config

// LoadConfig loads configuration settings from environment variables.
func LoadConfig() error {
	network := os.Getenv("NETWORK")
	infuraKey := os.Getenv("INFURA_PROJECT_ID")

	if network == "" || infuraKey == "" {
		return errors.New("NETWORK and INFURA_PROJECT_ID must be set")
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"  
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost" 
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" 
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	baseURL := fmt.Sprintf("https://%s.infura.io/v3/%s", network, infuraKey)

	cfg = Config{
		network:   network,
		infuraKey: infuraKey,
		url:       baseURL,
		addr: 	   addr,
		logLevel:  logLevel,
	}

	return nil
}

// GetConfig returns the loaded Config instance.
func GetConfig() Config {
	return cfg
}

// Network returns the Ethereum network for the configuration.
func (c Config) Network() string {
	return c.network
}

// InfuraKey returns the Infura project ID for the configuration.
func (c Config) InfuraKey() string {
	return c.infuraKey
}

// URL returns the Infura Ethereum node URL for the configuration.
func (c Config) URL() string {
	return c.url
}

// Addr returns the application's server address for the configuration.
func (c Config) Addr() string {
	return c.addr
}

// LogLevel returns the logging level for the configuration.
func (c Config) LogLevel() string {
	return c.logLevel
}

