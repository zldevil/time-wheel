package conf

import (
	"flag"
	"github.com/BurntSushi/toml"
	"time"
)

var (
	confPath string
	// Conf config
	Conf *Config
)

func init() {
	flag.StringVar(&confPath, "conf", "time-wheel.toml", "default config path.")
}

// Init init config.
func Init() (err error) {
	Conf = Default()
	_, err = toml.DecodeFile(confPath, &Conf)
	return
}

// Default new a config with specified defualt value.
func Default() *Config {
	return &Config{
		WheelConfig: &WheelConfig{},
	}
}

// Config is comet config.
type Config struct {
	WheelConfig *WheelConfig
}

// Env is env config.
type WheelConfig struct {
	Size   int64
	TickMs int
	T      time.Duration
}
