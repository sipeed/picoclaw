package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/migrate/internal"
	"github.com/sipeed/picoclaw/pkg/migrate/sources/openclaw"
)

type (
	Options        = internal.Options
	Operation      = internal.Operation
	ActionType     = internal.ActionType
	Action         = internal.Action
	Result         = internal.Result
	HandlerFactory = internal.HandlerFactory
)

const (
	ActionCopy          = internal.ActionCopy
	ActionSkip          = internal.ActionSkip
	ActionBackup        = internal.ActionBackup
	ActionConvertConfig = internal.ActionConvertConfig
	ActionCreateDir     = internal.ActionCreateDir
	ActionMergeConfig   = internal.ActionMergeConfig
)

type MigrateInstance struct {
	options  Options
	handlers map[string]Operation
}

func NewMigrateInstance(opts Options) *MigrateInstance {
	instance := &MigrateInstance{
		options:  opts,
		handlers: make(map[string]Operation),
	}

	openclaw_handler, err := openclaw.NewOpenclawHandler(opts)
	if err == nil {
		instance.Register(openclaw_handler.GetSourceName(), openclaw_handler)
	}

	return instance
}

func (m *MigrateInstance) Register(moduleName string, module Operation) {
	m.handlers[moduleName] = module
}

func (m *MigrateInstance) getCurrentHandler() (Operation, error) {
	source := m.options.Source
	if source == "" {
		source = "openclaw"
	}
	handler, ok := m.handlers[source]
	if !ok {
		return nil, fmt.Errorf("Source '%s' not found", source)
	}
	return handler, nil
}

func (m *MigrateInstance) Run(opts Options) (*Result, error) {
	handler, err := m.getCurrentHandler()
	if err != nil {
		return nil, err
	}

	if opts.ConfigOnly && opts.WorkspaceOnly {
		return nil, fmt.Errorf("--config-only and --workspace-only are mutually exclusive")
	}

	if opts.Refresh {
		opts.WorkspaceOnly = true
	}

	sourceHome, err := handler.GetSourceHome()
	if err != nil {
		return nil, err
	}

	targetHome, err := internal.ResolveTargetHome(opts.TargetHome)
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(sourceHome); os.IsNotExist(err) {
		return nil, fmt.Errorf("Source installation not found at %s", sourceHome)
	}

	actions, warnings, err := m.Plan(opts, sourceHome, targetHome)
	if err != nil {
		return nil, err
	}

	fmt.Println("Migrating from Source to PicoClaw")
	fmt.Printf("  Source:      %s\n", sourceHome)
	fmt.Printf("  Target: %s\n", targetHome)
	fmt.Println()

	if opts.DryRun {
		PrintPlan(actions, warnings)
		return &Result{Warnings: warnings}, nil
	}

	if !opts.Force {
		PrintPlan(actions, warnings)
		if !Confirm() {
			fmt.Println("Aborted.")
			return &Result{Warnings: warnings}, nil
		}
		fmt.Println()
	}

	result := m.Execute(actions, sourceHome, targetHome)
	result.Warnings = warnings
	return result, nil
}

func (m *MigrateInstance) Plan(opts Options, sourceHome, targetHome string) ([]Action, []string, error) {
	var actions []Action
	var warnings []string
	handler, err := m.getCurrentHandler()
	if err != nil {
		return nil, nil, err
	}

	force := opts.Force || opts.Refresh

	if !opts.WorkspaceOnly {
		configPath, err := handler.GetSourceConfigFile()
		if err != nil {
			if opts.ConfigOnly {
				return nil, nil, err
			}
			warnings = append(warnings, fmt.Sprintf("Config migration skipped: %v", err))
		} else {
			actions = append(actions, Action{
				Type:        ActionConvertConfig,
				Source:      configPath,
				Target:      filepath.Join(targetHome, "config.json"),
				Description: "convert Source config to PicoClaw format",
			})
		}
	}

	if !opts.ConfigOnly {
		srcWorkspace, err := handler.GetSourceWorkspace()
		if err != nil {
			return nil, nil, fmt.Errorf("getting source workspace: %w", err)
		}
		dstWorkspace := internal.ResolveWorkspace(targetHome)

		if _, err := os.Stat(srcWorkspace); err == nil {
			wsActions, err := internal.PlanWorkspaceMigration(srcWorkspace, dstWorkspace,
				handler.GetMigrateableFiles(),
				handler.GetMigrateableDirs(),
				force)
			if err != nil {
				return nil, nil, fmt.Errorf("planning workspace migration: %w", err)
			}
			actions = append(actions, wsActions...)
		} else {
			warnings = append(warnings, "Source workspace directory not found, skipping workspace migration")
		}
	}

	return actions, warnings, nil
}

func (m *MigrateInstance) Execute(actions []Action, sourceHome, targetHome string) *Result {
	result := &Result{}
	handler, err := m.getCurrentHandler()
	if err != nil {
		return result
	}

	for _, action := range actions {
		switch action.Type {
		case ActionConvertConfig:
			if err := handler.ExecuteConfigMigration(action.Source, action.Target); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("config migration: %w", err))
				fmt.Printf("  ✗ Config migration failed: %v\n", err)
			} else {
				result.ConfigMigrated = true
				fmt.Printf("  ✓ Converted config: %s\n", action.Target)
			}
		case ActionCreateDir:
			if err := os.MkdirAll(action.Target, 0o755); err != nil {
				result.Errors = append(result.Errors, err)
			} else {
				result.DirsCreated++
			}
		case ActionBackup:
			bakPath := action.Target + ".bak"
			if err := internal.CopyFile(action.Target, bakPath); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("backup %s: %w", action.Target, err))
				fmt.Printf("  ✗ Backup failed: %s\n", action.Target)
				continue
			}
			result.BackupsCreated++
			fmt.Printf(
				"  ✓ Backed up %s -> %s.bak\n",
				filepath.Base(action.Target),
				filepath.Base(action.Target),
			)

			if err := os.MkdirAll(filepath.Dir(action.Target), 0o755); err != nil {
				result.Errors = append(result.Errors, err)
				continue
			}
			if err := internal.CopyFile(action.Source, action.Target); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("copy %s: %w", action.Source, err))
				fmt.Printf("  ✗ Copy failed: %s\n", action.Source)
			} else {
				result.FilesCopied++
				fmt.Printf("  ✓ Copied %s\n", internal.RelPath(action.Source, sourceHome))
			}
		case ActionCopy:
			if err := os.MkdirAll(filepath.Dir(action.Target), 0o755); err != nil {
				result.Errors = append(result.Errors, err)
				continue
			}
			if err := internal.CopyFile(action.Source, action.Target); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("copy %s: %w", action.Source, err))
				fmt.Printf("  ✗ Copy failed: %s\n", action.Source)
			} else {
				result.FilesCopied++
				fmt.Printf("  ✓ Copied %s\n", internal.RelPath(action.Source, sourceHome))
			}
		case ActionSkip:
			result.FilesSkipped++
		}
	}

	return result
}

func Confirm() bool {
	fmt.Print("Proceed with migration? (y/n): ")
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}

func (m *MigrateInstance) PrintSummary(result *Result) {
	fmt.Println()
	parts := []string{}
	if result.FilesCopied > 0 {
		parts = append(parts, fmt.Sprintf("%d files copied", result.FilesCopied))
	}
	if result.ConfigMigrated {
		parts = append(parts, "1 config converted")
	}
	if result.BackupsCreated > 0 {
		parts = append(parts, fmt.Sprintf("%d backups created", result.BackupsCreated))
	}
	if result.FilesSkipped > 0 {
		parts = append(parts, fmt.Sprintf("%d files skipped", result.FilesSkipped))
	}

	if len(parts) > 0 {
		fmt.Printf("Migration complete! %s.\n", strings.Join(parts, ", "))
	} else {
		fmt.Println("Migration complete! No actions taken.")
	}

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Printf("%d errors occurred:\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}
}

func PrintPlan(actions []Action, warnings []string) {
	fmt.Println("Planned actions:")
	copies := 0
	skips := 0
	backups := 0
	configCount := 0

	for _, action := range actions {
		switch action.Type {
		case ActionConvertConfig:
			fmt.Printf("  [config]  %s -> %s\n", action.Source, action.Target)
			configCount++
		case ActionCopy:
			fmt.Printf("  [copy]    %s\n", filepath.Base(action.Source))
			copies++
		case ActionBackup:
			fmt.Printf("  [backup]  %s (exists, will backup and overwrite)\n", filepath.Base(action.Target))
			backups++
			copies++
		case ActionSkip:
			if action.Description != "" {
				fmt.Printf("  [skip]    %s (%s)\n", filepath.Base(action.Source), action.Description)
			}
			skips++
		case ActionCreateDir:
			fmt.Printf("  [mkdir]   %s\n", action.Target)
		}
	}

	if len(warnings) > 0 {
		fmt.Println()
		fmt.Println("Warnings:")
		for _, w := range warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	fmt.Println()
	fmt.Printf("%d files to copy, %d configs to convert, %d backups needed, %d skipped\n",
		copies, configCount, backups, skips)
}
