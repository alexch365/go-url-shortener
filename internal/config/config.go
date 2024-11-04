package config

type appConfig struct {
	ServerAddress   string `env:"SERVER_ADDRESS"`
	BaseURL         string `env:"BASE_URL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

var defaults = appConfig{
	ServerAddress:   "localhost:8080",
	BaseURL:         "http://localhost:8080",
	FileStoragePath: "shorten_urls.json",
	DatabaseDSN:     "",
}

var Current = appConfig{}

func SetDefaults() {
	if Current.ServerAddress == "" {
		Current.ServerAddress = defaults.ServerAddress
	}
	if Current.BaseURL == "" {
		Current.BaseURL = defaults.BaseURL
	}
	if Current.FileStoragePath == "" {
		Current.FileStoragePath = defaults.FileStoragePath
	}
}
