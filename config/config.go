package config

type Config struct {
    workspacePath string `json:"workspace_path"`
    Heartbeat     struct {
        Interval int `json:"interval"`
        Enabled  bool `json:"enabled"`
    } `json:"heartbeat"`
}

func (c *Config) WorkspacePath() string {
    return c.workspacePath
}