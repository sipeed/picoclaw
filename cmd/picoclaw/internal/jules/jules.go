package jules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"jane/cmd/picoclaw/internal"
)

var julesBaseURL = "https://jules.googleapis.com/v1alpha"

func getAPIKey() (string, error) {
	if key := os.Getenv("JULES_API_KEY"); key != "" {
		return key, nil
	}
	cfg, err := internal.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.Tools.Jules.APIKey != "" {
		return cfg.Tools.Jules.APIKey, nil
	}
	return "", fmt.Errorf("JULES_API_KEY environment variable or config value not set")
}

func doRequest(method, url string, body []byte) error {
	apiKey, err := getAPIKey()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if len(respBody) > 0 {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, respBody, "", "  "); err == nil {
			fmt.Println(prettyJSON.String())
		} else {
			fmt.Println(string(respBody))
		}
	} else {
		fmt.Println("Success")
	}

	return nil
}

func NewJulesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jules",
		Short: "Manage Jules sessions and activities",
	}

	cmd.AddCommand(newSessionCmd())
	cmd.AddCommand(newActivityCmd())

	return cmd
}

func newSessionCmd() *cobra.Command {
	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Manage Jules sessions",
	}

	var prompt string
	var title string
	var source string
	var branch string

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if prompt == "" || source == "" {
				return fmt.Errorf("--prompt and --source are required")
			}

			payload := map[string]interface{}{
				"prompt": prompt,
				"sourceContext": map[string]interface{}{
					"source": source,
				},
			}
			if title != "" {
				payload["title"] = title
			}
			if branch != "" {
				payload["sourceContext"].(map[string]interface{})["githubRepoContext"] = map[string]interface{}{
					"startingBranch": branch,
				}
			}

			body, _ := json.Marshal(payload)
			return doRequest("POST", julesBaseURL+"/sessions", body)
		},
	}
	createCmd.Flags().StringVar(&prompt, "prompt", "", "The prompt for the session")
	createCmd.Flags().StringVar(&title, "title", "", "The title of the session")
	createCmd.Flags().StringVar(&source, "source", "", "The source repository (e.g., sources/github-owner-repo)")
	createCmd.Flags().StringVar(&branch, "branch", "", "The starting branch")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doRequest("GET", julesBaseURL+"/sessions", nil)
		},
	}

	getCmd := &cobra.Command{
		Use:   "get [sessionId]",
		Short: "Get a session by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doRequest("GET", julesBaseURL+"/sessions/"+args[0], nil)
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete [sessionId]",
		Short: "Delete a session by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doRequest("DELETE", julesBaseURL+"/sessions/"+args[0], nil)
		},
	}

	var message string
	messageCmd := &cobra.Command{
		Use:   "message [sessionId]",
		Short: "Send a message to a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if message == "" {
				return fmt.Errorf("--message is required")
			}
			payload := map[string]interface{}{
				"prompt": message,
			}
			body, _ := json.Marshal(payload)
			return doRequest("POST", julesBaseURL+"/sessions/"+args[0]+":sendMessage", body)
		},
	}
	messageCmd.Flags().StringVar(&message, "message", "", "The message to send")

	approveCmd := &cobra.Command{
		Use:   "approve [sessionId]",
		Short: "Approve a session plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doRequest("POST", julesBaseURL+"/sessions/"+args[0]+":approvePlan", []byte("{}"))
		},
	}

	sessionCmd.AddCommand(createCmd, listCmd, getCmd, deleteCmd, messageCmd, approveCmd)
	return sessionCmd
}

func newActivityCmd() *cobra.Command {
	activityCmd := &cobra.Command{
		Use:   "activity",
		Short: "Manage Jules activities",
	}

	listCmd := &cobra.Command{
		Use:   "list [sessionId]",
		Short: "List activities for a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doRequest("GET", julesBaseURL+"/sessions/"+args[0]+"/activities", nil)
		},
	}

	getCmd := &cobra.Command{
		Use:   "get [sessionId] [activityId]",
		Short: "Get an activity by ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return doRequest("GET", julesBaseURL+"/sessions/"+args[0]+"/activities/"+args[1], nil)
		},
	}

	activityCmd.AddCommand(listCmd, getCmd)
	return activityCmd
}
