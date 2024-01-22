package config

type Config struct {
	BindAddress string
	LogLevel    string
}

func New() *Config {
	return &Config{
		BindAddress: ":8080",
		LogLevel:    "debug",
	}
}
