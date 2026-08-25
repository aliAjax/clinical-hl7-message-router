package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPAddr, MLLPAddr string
	DatabaseURL        string
	AuthToken          string
}

type fileConfig struct {
	HTTP struct {
		Address string `yaml:"address"`
	} `yaml:"http"`
	MLLP struct {
		Address string `yaml:"address"`
	} `yaml:"mllp"`
	Storage struct {
		DatabaseURL string `yaml:"database_url"`
	} `yaml:"storage"`
}

func Load() (Config, error) {
	c := Config{HTTPAddr: ":8084", MLLPAddr: ":2575", DatabaseURL: "memory://"}
	if path := os.Getenv("CONFIG_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config %s: %w", path, err)
		}
		var file fileConfig
		if err := yaml.Unmarshal(data, &file); err != nil {
			return Config{}, fmt.Errorf("decode config %s: %w", path, err)
		}
		if file.HTTP.Address != "" {
			c.HTTPAddr = file.HTTP.Address
		}
		if file.MLLP.Address != "" {
			c.MLLPAddr = file.MLLP.Address
		}
		if file.Storage.DatabaseURL != "" {
			c.DatabaseURL = file.Storage.DatabaseURL
		}
	}
	c.HTTPAddr = env("HTTP_ADDR", c.HTTPAddr)
	c.MLLPAddr = env("MLLP_ADDR", c.MLLPAddr)
	c.DatabaseURL = env("DATABASE_URL", c.DatabaseURL)
	c.AuthToken = os.Getenv("AUTH_TOKEN")
	return c, nil
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func IntEnv(k string, d int) int {
	v, _ := strconv.Atoi(os.Getenv(k))
	if v == 0 {
		return d
	}
	return v
}

func RequestTimeout() time.Duration {
	return 15 * time.Second
}
