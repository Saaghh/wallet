package apiserver

type Config struct {
	Port     string
	LogLevel string
}

func NewConfig() *Config {
	return &Config{
		Port:     ":8080",
		LogLevel: "debug",
	}
}
