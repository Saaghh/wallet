package config

type Config struct {
	Port     string
	LogLevel string
}

func New() *Config {
	return &Config{
		Port:     ":8080",
		LogLevel: "debug",
	}
}
