package command

import (
	"echodb/internal/config"
	cmdCfg "echodb/internal/domain/command-config"
	"fmt"
)

type Settings struct {
	Config *cmdCfg.ConfigData
	AppCfg *config.Settings
}

func NewApp(appCfg *config.Settings, config *cmdCfg.ConfigData) *Settings {
	return &Settings{
		Config: config,
		AppCfg: appCfg,
	}
}

func (s *Settings) GetCommand() (string, string, error) {
	gen, ok := GetGenerator(s.AppCfg.Driver)
	if !ok {
		return "", "", fmt.Errorf("unsupported driver: %s", s.AppCfg.Driver)
	}

	cmd, remotePath := gen.Generate(s.Config, s.AppCfg)
	return cmd, remotePath, nil
}
