// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sipeed/picoclaw/pkg/cron"
	"github.com/spf13/cobra"
)

func newCronCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Manage scheduled tasks",
	}
	cmd.AddCommand(
		newCronListCmd(),
		newCronAddCmd(),
		newCronRemoveCmd(),
		newCronEnableCmd(),
		newCronDisableCmd(),
	)
	return cmd
}

func newCronListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all scheduled jobs",
		RunE:  runCronList,
	}
}

func newCronAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new scheduled job",
		RunE:  runCronAdd,
	}
	cmd.Flags().StringP("name", "n", "", "Job name")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringP("message", "m", "", "Message for agent")
	cmd.MarkFlagRequired("message")
	cmd.Flags().Int64P("every", "e", 0, "Run every N seconds")
	cmd.Flags().StringP("cron", "c", "", "Cron expression (e.g. '0 9 * * *')")
	cmd.Flags().BoolP("deliver", "d", false, "Deliver response to channel")
	cmd.Flags().String("to", "", "Recipient for delivery")
	cmd.Flags().String("channel", "", "Channel for delivery")
	return cmd
}

func newCronRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <job_id>",
		Short: "Remove a job by ID",
		Args:  cobra.ExactArgs(1),
		RunE:  runCronRemove,
	}
}

func newCronEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <job_id>",
		Short: "Enable a job",
		Args:  cobra.ExactArgs(1),
		RunE:  runCronEnable,
	}
}

func newCronDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <job_id>",
		Short: "Disable a job",
		Args:  cobra.ExactArgs(1),
		RunE:  runCronDisable,
	}
}

func getCronStorePath() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", fmt.Errorf("Error loading config: %w", err)
	}
	return filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json"), nil
}

func runCronList(cmd *cobra.Command, args []string) error {
	storePath, err := getCronStorePath()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	cs := cron.NewCronService(storePath, nil)
	jobs := cs.ListJobs(true)

	if len(jobs) == 0 {
		fmt.Println("No scheduled jobs.")
		return nil
	}

	fmt.Println("\nScheduled Jobs:")
	fmt.Println("----------------")
	for _, job := range jobs {
		var schedule string
		if job.Schedule.Kind == "every" && job.Schedule.EveryMS != nil {
			schedule = fmt.Sprintf("every %ds", *job.Schedule.EveryMS/1000)
		} else if job.Schedule.Kind == "cron" {
			schedule = job.Schedule.Expr
		} else {
			schedule = "one-time"
		}

		nextRun := "scheduled"
		if job.State.NextRunAtMS != nil {
			nextTime := time.UnixMilli(*job.State.NextRunAtMS)
			nextRun = nextTime.Format("2006-01-02 15:04")
		}

		status := "enabled"
		if !job.Enabled {
			status = "disabled"
		}

		fmt.Printf("  %s (%s)\n", job.Name, job.ID)
		fmt.Printf("    Schedule: %s\n", schedule)
		fmt.Printf("    Status: %s\n", status)
		fmt.Printf("    Next run: %s\n", nextRun)
	}
	return nil
}

func runCronAdd(cmd *cobra.Command, args []string) error {
	storePath, err := getCronStorePath()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	name, _ := cmd.Flags().GetString("name")
	message, _ := cmd.Flags().GetString("message")
	everySec, _ := cmd.Flags().GetInt64("every")
	cronExpr, _ := cmd.Flags().GetString("cron")
	deliver, _ := cmd.Flags().GetBool("deliver")
	to, _ := cmd.Flags().GetString("to")
	channel, _ := cmd.Flags().GetString("channel")

	if everySec == 0 && cronExpr == "" {
		fmt.Println("Error: Either --every or --cron must be specified")
		return nil
	}

	var schedule cron.CronSchedule
	if everySec != 0 {
		everyMS := everySec * 1000
		schedule = cron.CronSchedule{
			Kind:    "every",
			EveryMS: &everyMS,
		}
	} else {
		schedule = cron.CronSchedule{
			Kind: "cron",
			Expr: cronExpr,
		}
	}

	cs := cron.NewCronService(storePath, nil)
	job, err := cs.AddJob(name, schedule, message, deliver, channel, to)
	if err != nil {
		fmt.Printf("Error adding job: %v\n", err)
		return nil
	}

	fmt.Printf("\u2713 Added job '%s' (%s)\n", job.Name, job.ID)
	return nil
}

func runCronRemove(cmd *cobra.Command, args []string) error {
	storePath, err := getCronStorePath()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	jobID := args[0]
	cs := cron.NewCronService(storePath, nil)
	if cs.RemoveJob(jobID) {
		fmt.Printf("\u2713 Removed job %s\n", jobID)
	} else {
		fmt.Printf("\u2717 Job %s not found\n", jobID)
	}
	return nil
}

func runCronEnable(cmd *cobra.Command, args []string) error {
	storePath, err := getCronStorePath()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	jobID := args[0]
	cs := cron.NewCronService(storePath, nil)
	job := cs.EnableJob(jobID, true)
	if job != nil {
		fmt.Printf("\u2713 Job '%s' enabled\n", job.Name)
	} else {
		fmt.Printf("\u2717 Job %s not found\n", jobID)
	}
	return nil
}

func runCronDisable(cmd *cobra.Command, args []string) error {
	storePath, err := getCronStorePath()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	jobID := args[0]
	cs := cron.NewCronService(storePath, nil)
	job := cs.EnableJob(jobID, false)
	if job != nil {
		fmt.Printf("\u2713 Job '%s' disabled\n", job.Name)
	} else {
		fmt.Printf("\u2717 Job %s not found\n", jobID)
	}
	return nil
}
