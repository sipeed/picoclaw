package internal

type Options struct {
	DryRun        bool
	ConfigOnly    bool
	WorkspaceOnly bool
	Force         bool
	Refresh       bool
	Source        string
	SourceHome    string
	TargetHome    string
}

type Operation interface {
	GetSourceName() string
	GetSourceHome() (string, error)
	GetSourceWorkspace() (string, error)
	GetSourceConfigFile() (string, error)
	ExecuteConfigMigration(srcConfigPath, dstConfigPath string) error
	GetMigrateableFiles() []string
	GetMigrateableDirs() []string
}

type HandlerFactory func(opts Options) Operation

type ActionType int

const (
	ActionCopy ActionType = iota
	ActionSkip
	ActionBackup
	ActionConvertConfig
	ActionCreateDir
	ActionMergeConfig
)

type Action struct {
	Type        ActionType
	Source      string
	Target      string
	Description string
}

type Result struct {
	FilesCopied    int
	FilesSkipped   int
	BackupsCreated int
	ConfigMigrated bool
	DirsCreated    int
	Warnings       []string
	Errors         []error
}
