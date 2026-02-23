package setup

import (
	"github.com/sipeed/picoclaw/pkg/config"
)

// QuestionType defines the type of response expected for a question.
type QuestionType string

const (
	QuestionTypeText   QuestionType = "text"
	QuestionTypeSelect QuestionType = "select"
	QuestionTypeYesNo  QuestionType = "yesno"
)

// Question represents a single question in the setup wizard.
type Question struct {
	ID           string       // unique identifier for mapping answer to config
	Type         QuestionType // response type: text, select, yesno
	Prompt       string       // question text shown to user
	Info         string       // optional helper text
	Options      []string     // options for select/yesno types
	DefaultValue string       // default value to prefill
	DependsOn    string       // question ID this depends on (optional)
	DependsValue string       // value that must be matched (optional)
	ConfigPath   string       // dot-notation path to config field (e.g. "Agents.Defaults.Workspace")
	Transformer  string       // optional: "lowercase", "bool_yesno", "channel_enable"
}

// QuestionGroup represents a group of related questions.
type QuestionGroup struct {
	Name      string
	Questions []Question
}

// AllQuestions returns the complete list of all possible questions in the setup wizard.
// This is the declarative data structure - questions are defined here, not built procedurally.
var AllQuestions = []QuestionGroup{
	{
		Name: "Workspace",
		Questions: []Question{
			{
				ID:           "workspace",
				Type:         QuestionTypeText,
				Prompt:       "1. Workspace path",
				Info:         "Directory for agent workspace files",
				ConfigPath:   "Agents.Defaults.Workspace",
				DefaultValue: "~/.picoclaw/workspace",
			},
			{
				ID:          "restrict_workspace",
				Type:        QuestionTypeYesNo,
				Prompt:      "1b. Restrict to workspace?",
				ConfigPath:  "Agents.Defaults.RestrictToWorkspace",
				Transformer: "bool_yesno",
			},
		},
	},
	{
		Name: "Provider",
		Questions: []Question{
			{
				ID:         "provider",
				Type:       QuestionTypeSelect,
				Prompt:     "2. Choose provider",
				ConfigPath: "Agents.Defaults.Provider",
			},
			{
				ID:         "provider_api_key",
				Type:       QuestionTypeText,
				Prompt:     "2b. API key for provider",
				ConfigPath: "Providers.{provider}.APIKey",
				DependsOn:  "provider",
			},
			{
				ID:         "provider_api_base",
				Type:       QuestionTypeText,
				Prompt:     "2c. API base URL (optional)",
				ConfigPath: "Providers.{provider}.APIBase",
				DependsOn:  "provider",
			},
		},
	},
	{
		Name: "Model",
		Questions: []Question{
			{
				ID:         "model_select",
				Type:       QuestionTypeSelect,
				Prompt:     "3. Choose model",
				ConfigPath: "Agents.Defaults.Model",
			},
			{
				ID:           "custom_model",
				Type:         QuestionTypeText,
				Prompt:       "3b. Enter custom model name",
				ConfigPath:   "Agents.Defaults.Model",
				DependsOn:    "model_select",
				DependsValue: "custom",
			},
		},
	},
	{
		Name: "Channel",
		Questions: []Question{
			{
				ID:         "channel_select",
				Type:       QuestionTypeSelect,
				Prompt:     "4. Choose channel",
				ConfigPath: "channel_enabled",
			},
			{
				ID:         "channel_token",
				Type:       QuestionTypeText,
				Prompt:     "4b. Channel token/credential",
				ConfigPath: "channel.{channel}.token",
				DependsOn:  "channel_select",
			},
		},
	},
	{
		Name: "Confirmation",
		Questions: []Question{
			{
				ID:     "confirm",
				Type:   QuestionTypeYesNo,
				Prompt: "5. Confirm and save configuration?",
			},
		},
	},
}

// QuestionRegistry holds resolved questions for a specific config state.
type QuestionRegistry struct {
	Questions []Question
	Defaults  map[string]string // questionID -> default value
}

// BuildQuestionRegistry creates the question registry based on current config.
// It resolves dynamic options (provider list, channel list, model suggestions) from config.
func BuildQuestionRegistry(cfg *config.Config) QuestionRegistry {
	registry := QuestionRegistry{
		Questions: []Question{},
		Defaults:  map[string]string{},
	}

	// Collect provider options
	provInfo := config.GetProvidersInfo(cfg)
	ordered := config.GetOrderedProviderNames()
	provOptions := []string{}
	added := map[string]struct{}{}
	for _, name := range ordered {
		if _, ok := provInfoLookup(provInfo, name); ok {
			provOptions = append(provOptions, name)
			added[name] = struct{}{}
		}
	}
	for _, p := range provInfo {
		if _, ok := added[p.Name]; !ok {
			provOptions = append(provOptions, p.Name)
		}
	}

	// Collect channel options
	channelOptions := config.GetAllChannelNames()

	// Collect model options based on selected provider
	selProvider := cfg.Agents.Defaults.Provider
	modelSuggestions := config.GetPopularModels(selProvider)
	modelOptions := make([]string, len(modelSuggestions)+1)
	copy(modelOptions, modelSuggestions)
	modelOptions[len(modelSuggestions)] = "custom"

	// Build flat list of questions with resolved options
	for _, group := range AllQuestions {
		for _, q := range group.Questions {
			question := q

			// Set options based on question type
			if question.Type == QuestionTypeSelect {
				switch question.ID {
				case "provider":
					question.Options = provOptions
				case "channel_select":
					question.Options = channelOptions
				case "model_select":
					question.Options = modelOptions
				}
			}

			// Set defaults from config
			if question.DefaultValue != "" {
				registry.Defaults[question.ID] = question.DefaultValue
			}

			registry.Questions = append(registry.Questions, question)
		}
	}

	return registry
}

// GetQuestionByID returns a question by its ID.
func (r *QuestionRegistry) GetQuestionByID(id string) *Question {
	for i := range r.Questions {
		if r.Questions[i].ID == id {
			return &r.Questions[i]
		}
	}
	return nil
}

// GetDependentQuestions returns questions that depend on a specific question.
func (r *QuestionRegistry) GetDependentQuestions(dependsOn string) []Question {
	var deps []Question
	for _, q := range r.Questions {
		if q.DependsOn == dependsOn {
			deps = append(deps, q)
		}
	}
	return deps
}

// ShouldShowQuestion checks if a question should be shown based on its dependencies.
func (r *QuestionRegistry) ShouldShowQuestion(questionID string, answers map[string]string) bool {
	q := r.GetQuestionByID(questionID)
	if q == nil || q.DependsOn == "" {
		return true
	}

	depValue, exists := answers[q.DependsOn]
	if !exists {
		return false
	}

	// If DependsValue is set, check for exact match
	if q.DependsValue != "" {
		return depValue == q.DependsValue
	}

	// Otherwise show if dependency has any value
	return depValue != ""
}
