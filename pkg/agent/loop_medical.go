package agent

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"jane/pkg/bus"
	"jane/pkg/logger"
)

// The medical CoT state machine phases
const (
	PhaseExtraction          = "extraction"
	PhaseTemporalCorrelation = "temporal_correlation"
	PhaseTheoryGeneration    = "theory_generation"
	PhaseVerification        = "verification"
	PhaseSafetyDisclaimers   = "safety_disclaimers"
)

// processMedicalRequest implements the medical persona CoT loop
func (al *AgentLoop) processMedicalRequest(
	ctx context.Context,
	agent *AgentInstance,
	opts processOptions,
) (string, error) {
	logger.InfoCF("medical", "Starting clinical CoT loop", map[string]any{
		"session_key": opts.SessionKey,
	})

	// Detect target patient from message
	targetPatient := detectTargetPatient(opts.UserMessage)
	if targetPatient == "" {
		// If no specific patient is explicitly asked for, we ask the user for clarification.
		msg := "Please specify the patient you are analyzing (e.g., 'Analyze patient John Doe')."
		if opts.SendResponse {
			al.bus.PublishOutbound(ctx, bus.OutboundMessage{
				Channel: opts.Channel,
				ChatID:  opts.ChatID,
				Content: msg,
			})
		}
		return msg, nil
	}

	// Ensure the session history is isolated to this specific patient and not globally
	// shared across patients if the clinician uses the same chat session.
	patientSessionKey := opts.SessionKey + ":patient:" + targetPatient

	// Lock the workspace path to the patient directory
	patientWorkspace := filepath.Join(agent.Workspace, targetPatient)

	// We must not modify the shared agent instance directly to avoid data races.
	// Instead, we clone the agent instance for this specific request.
	clonedAgent := *agent
	clonedAgent.Workspace = patientWorkspace

	var finalResponse strings.Builder
	finalResponse.WriteString(fmt.Sprintf("Clinician Agent initialized for patient: %s\n\n", targetPatient))

	// CoT State Machine variables
	var currentContext string = opts.UserMessage

	phases := []string{
		PhaseExtraction,
		PhaseTemporalCorrelation,
		PhaseTheoryGeneration,
		PhaseVerification,
		PhaseSafetyDisclaimers,
	}

	for _, phase := range phases {
		logger.DebugCF("medical", "Executing phase", map[string]any{"phase": phase})

		prompt := buildPhasePrompt(phase, currentContext)

		// Create a temporary opts for this phase
		phaseOpts := opts
		phaseOpts.SessionKey = patientSessionKey
		phaseOpts.UserMessage = prompt
		phaseOpts.SendResponse = false // don't send intermediate steps to bus
		phaseOpts.EnableSummary = false

		// Run a standard single LLM execution for this phase
		phaseResult, err := al.runAgentLoop(ctx, &clonedAgent, phaseOpts)
		if err != nil {
			logger.ErrorCF("medical", "Phase failed", map[string]any{
				"phase": phase,
				"error": err.Error(),
			})
			return "", err
		}

		// Accumulate result
		finalResponse.WriteString(fmt.Sprintf("### [%s]\n%s\n\n", phase, phaseResult))

		// Pass result as context to next phase
		currentContext = currentContext + "\n\n" + fmt.Sprintf("Result of %s:\n%s", phase, phaseResult)

		// Mandatory safety check abort
		if phase == PhaseSafetyDisclaimers {
			if strings.Contains(strings.ToLower(phaseResult), "life-threatening") ||
				strings.Contains(strings.ToLower(phaseResult), "red flag") {
				finalResponse.WriteString("\n⚠️ **RED FLAG DETECTED: This requires immediate emergency medical attention.**\n")
			}
		}
	}

	// Send to bus if requested
	if opts.SendResponse {
		al.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Channel: opts.Channel,
			ChatID:  opts.ChatID,
			Content: finalResponse.String(),
		})
	}

	return finalResponse.String(), nil
}

func buildPhasePrompt(phase string, context string) string {
	basePrompt := "You are The Clinician, an expert medical reasoning agent.\nContext:\n%s\n\nTask:\n"

	switch phase {
	case PhaseExtraction:
		return fmt.Sprintf(basePrompt, context) + "Extract all clinical terminology, symptoms, and vital signs from the context."
	case PhaseTemporalCorrelation:
		return fmt.Sprintf(basePrompt, context) + "Analyze the temporal correlation of symptoms. Scan for recurring patterns in the patient's history."
	case PhaseTheoryGeneration:
		return fmt.Sprintf(basePrompt, context) + "Generate a ranked list of potential causes (DDx) and pathophysiological theories based on the extracted symptoms and patterns."
	case PhaseVerification:
		return fmt.Sprintf(basePrompt, context) + "Verify the proposed theories and any implicit remedies against the patient's Allergies and Medications profile."
	case PhaseSafetyDisclaimers:
		return fmt.Sprintf(basePrompt, context) + "Provide mandatory clinical disclaimers. Explicitly state whether any 'Life-Threatening' or 'Red Flag' symptoms were detected."
	default:
		return context
	}
}

func detectTargetPatient(message string) string {
	// Simple regex to extract patient name (e.g., "patient John Doe" or "patient: John Doe")
	re := regexp.MustCompile(`(?i)patient\s*:?\s*([a-zA-Z0-9_]+(?:\s+[a-zA-Z0-9_]+)*)`)
	matches := re.FindStringSubmatch(message)
	if len(matches) > 1 {
		// Clean up the patient name to make it directory-safe
		patientName := strings.TrimSpace(matches[1])
		patientName = strings.ReplaceAll(patientName, " ", "_")
		return patientName
	}
	return ""
}
