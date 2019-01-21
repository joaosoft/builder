package builder

import (
	"fmt"

	"time"

	"github.com/joaosoft/manager"
)

// AppConfig ...
type AppConfig struct {
	Builder *BuilderConfig `json:"builder"`
}

// BuilderConfig ...
type BuilderConfig struct {
	Source      string        `json:"source"`
	Destination string        `json:"destination"`
	ReloadTime  time.Duration `json:"reload_time"`
	Log         struct {
		Level string `json:"level"`
	} `json:"log"`
}

// NewConfig ...
func NewConfig() (*AppConfig, manager.IConfig, error) {
	appConfig := &AppConfig{}
	simpleConfig, err := manager.NewSimpleConfig(fmt.Sprintf("/config/app.%s.json", GetEnv()), appConfig)

	return appConfig, simpleConfig, err
}
