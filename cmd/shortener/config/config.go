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

type Config struct {
	ServerAddress NetAddress
	BaseURL string
}

func (addr *NetAddress) String() string {
    return addr.Host + ":" + strconv.Itoa(addr.Port)
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