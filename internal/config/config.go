package config

import (
	"errors"
	"strconv"
	"strings"
)

type NetAddress struct {
	Host string
	Port int
}

type appConfig struct {
	ServerAddress string `env:"SERVER_ADDRESS"` // struct is not supported by env
	BaseURL       string `env:"BASE_URL"`
}

var defaults = appConfig{
	ServerAddress: "localhost:8080", //NetAddress{Host: "localhost", Port: 8080},
	BaseURL:       "http://localhost:8080",
}

var Current = appConfig{}

func SetDefaults() {
	if Current.ServerAddress == "" {
		Current.ServerAddress = defaults.ServerAddress
	}
	if Current.BaseURL == "" {
		Current.BaseURL = defaults.BaseURL
	}
}

func (addr *NetAddress) String() string {
	return strings.Join([]string{addr.Host, strconv.Itoa(addr.Port)}, ":")
}

func (addr *NetAddress) Set(s string) error {
	addrParts := strings.SplitN(s, ":", 2)
	if len(addrParts) < 2 {
		return errors.New("provide an address in format \"host:port\"")
	}
	port, err := strconv.Atoi(addrParts[1])
	if err != nil {
		return err
	}
	addr.Host = addrParts[0]
	addr.Port = port
	return nil
}
