package evolutioncmd

import (
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/evolution"
)

func loadEvolutionDeps() (*config.Config, *evolution.Store, string, error) {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return nil, nil, "", err
	}

	workspace := cfg.WorkspacePath()
	store := evolution.NewStore(evolution.NewPaths(workspace, cfg.Evolution.StateDir))
	return cfg, store, workspace, nil
}
