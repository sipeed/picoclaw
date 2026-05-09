package evolution

import "strings"

func skillDraftPromptInstructions() []string {
	return []string{
		"body_or_patch must contain the complete draft body or patch content as plain text.",
		"body_or_patch is an internal draft and review artifact, so it may include concise learning provenance, source task evidence, and source skill summaries when useful for human review.",
		"If change_kind is create, body_or_patch must be a complete SKILL.md file with exactly two parts: YAML frontmatter and a Markdown body.",
		"The YAML frontmatter must contain only name and description fields.",
		"The description field must and only describe what this skill can do and when to use it.",
		"The deployable Markdown body should only contain what the skill is useful for and how to use it.",
		"The Markdown body is loaded only after the skill triggers, so focus on concise usage guidance and the execution steps needed to complete the task.",
		"When describing an operation process in the body, do not use vague summaries; provide detailed step-by-step instructions for the exact operation or execution process.",
		"When creating a combined shortcut skill, summarize the functional purpose and result of the provided SKILL.md inputs; do not copy or directly include other skills' instructions.",
		"Extract only the necessary operations from source skills and evidence, such as formulas, ordered transformations, commands, inputs, outputs, and boundary conditions.",
		"The operational part of the generated skill must be directly usable by a future agent without reading the original task records or source skills.",
		"Keep operational instructions separable from audit/provenance notes because the final deployed SKILL.md will be rendered without learning traces.",
	}
}

func skillDraftPromptText() string {
	return strings.Join(skillDraftPromptInstructions(), "\n")
}

func learningTraceReplacer() *strings.Replacer {
	return strings.NewReplacer(
		"## Learned Shortcut Update", "## Shortcut Update",
		"## Learned Evolution", "## Usage Notes",
		"## Learned Pattern", "## Usage Pattern",
		"## Learned Context", "## Procedure Notes",
		"## Source Evidence", "## Validation",
		"## Source Skills", "## Procedure Details",
		"### Source Skills", "### Procedure Details",
		"## Learned Shortcut", "## Shortcut",
		"### Learned Shortcut", "### Shortcut",
		"Learned workflow for ", "Workflow for ",
		"learned workflow for ", "workflow for ",
		"from learned pattern: ", "for: ",
		"Learned task:", "Task:",
		"learned task:", "task:",
		"Learned pattern:", "Pattern:",
		"learned pattern:", "pattern:",
		"Learned from", "Based on",
		"learned from", "based on",
		"Source evidence", "Validation",
		"source evidence", "validation",
		"task records", "validated examples",
		"Task records", "Validated examples",
	)
}

func renderDeployableSkillBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return body
	}
	frontmatter, markdownBody := splitSkillFrontmatter(body)
	if frontmatter != "" {
		body = "---\n" + frontmatter + "\n---\n" + learningTraceReplacer().Replace(strings.TrimLeft(markdownBody, "\n"))
	} else {
		body = learningTraceReplacer().Replace(body)
	}
	body = normalizeDeployableDescription(body)
	return removeDeployOnlyProvenanceLines(body)
}

func normalizeDeployableDescription(body string) string {
	lines := strings.Split(body, "\n")
	inFrontmatter := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && trimmed == "---" {
			break
		}
		if !inFrontmatter || !strings.HasPrefix(trimmed, "description:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
		value = cleanDeployableDescription(value)
		lines[i] = "description: " + value
		break
	}
	return strings.Join(lines, "\n")
}

func cleanDeployableDescription(description string) string {
	description = strings.TrimSpace(strings.Trim(description, `"'`))
	for _, marker := range []string{
		" for: ",
		" from learned pattern: ",
		" for learned pattern: ",
	} {
		if idx := strings.Index(strings.ToLower(description), marker); idx >= 0 {
			description = strings.TrimSpace(description[idx+len(marker):])
			break
		}
	}
	description = strings.TrimPrefix(description, "Create combined shortcut ")
	description = strings.TrimPrefix(description, "Refresh combined shortcut ")
	description = strings.TrimPrefix(description, "Create shortcut ")
	description = strings.TrimPrefix(description, "Refresh shortcut ")
	description = strings.TrimSpace(description)
	if description == "" {
		return "Use this skill when the task matches its documented workflow."
	}
	return description
}

func sentenceFragment(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "complete the documented workflow"
	}
	runes := []rune(text)
	if len(runes) > 0 && runes[0] >= 'A' && runes[0] <= 'Z' {
		runes[0] = runes[0] + ('a' - 'A')
	}
	return string(runes)
}

func trimAtReadableBoundary(content string, maxLen int) string {
	content = strings.TrimSpace(content)
	runes := []rune(content)
	if content == "" || maxLen <= 0 || len(runes) <= maxLen {
		return content
	}

	cut := maxLen
	searchStart := maxLen - minInt(maxLen/2, 240)
	if searchStart < 0 {
		searchStart = 0
	}
	for i := maxLen; i >= searchStart; i-- {
		switch runes[i-1] {
		case '\n', '.', '!', '?', ';', ':', '。', '！', '？', '；', '：':
			cut = i
			goto done
		}
	}
	for i := maxLen; i >= searchStart; i-- {
		if runes[i-1] == ' ' || runes[i-1] == '\t' {
			cut = i
			goto done
		}
	}

done:
	return strings.TrimRight(strings.TrimSpace(string(runes[:cut])), ".,;:，。；：") + "..."
}

func removeDeployOnlyProvenanceLines(body string) string {
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "- evidence:") {
			continue
		}
		if strings.HasPrefix(lower, "- validated examples:") {
			continue
		}
		if strings.Contains(lower, "source_record_id") || strings.Contains(lower, "source record") {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
