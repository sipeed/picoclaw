package agent

// ---------- Plan passthrough methods ----------
// These delegate to MemoryStore and are separated to reduce upstream conflicts.

// HasActivePlan returns true if MEMORY.md contains an active plan.
func (cb *ContextBuilder) HasActivePlan() bool {
	return cb.memory.HasActivePlan()
}

// GetPlanStatus returns the plan status: "interviewing", "executing", or "".
func (cb *ContextBuilder) GetPlanStatus() string {
	return cb.memory.GetPlanStatus()
}

// IsPlanComplete returns true if all steps in all phases are [x].
func (cb *ContextBuilder) IsPlanComplete() bool {
	return cb.memory.IsPlanComplete()
}

// IsCurrentPhaseComplete returns true if all steps in the current phase are [x].
func (cb *ContextBuilder) IsCurrentPhaseComplete() bool {
	return cb.memory.IsCurrentPhaseComplete()
}

// AdvancePhase increments the current phase number by 1.
func (cb *ContextBuilder) AdvancePhase() error {
	return cb.memory.AdvancePhase()
}

// SetCurrentPhase sets the current phase number to n.
func (cb *ContextBuilder) SetCurrentPhase(n int) error {
	return cb.memory.SetPhase(n)
}

// GetCurrentPhase returns the current phase number.
func (cb *ContextBuilder) GetCurrentPhase() int {
	return cb.memory.GetCurrentPhase()
}

// GetTotalPhases returns the total number of phases in the plan.
func (cb *ContextBuilder) GetTotalPhases() int {
	return cb.memory.GetTotalPhases()
}

// FormatPlanDisplay returns a user-facing display of the full plan.
func (cb *ContextBuilder) FormatPlanDisplay() string {
	return cb.memory.FormatPlanDisplay()
}

// MarkStep marks a step as done in the specified phase.
func (cb *ContextBuilder) MarkStep(phase, step int) error {
	return cb.memory.MarkStep(phase, step)
}

// AddStep appends a new step to the given phase.
func (cb *ContextBuilder) AddStep(phase int, desc string) error {
	return cb.memory.AddStep(phase, desc)
}

// ValidatePlanStructure validates plan structure for interview->review transition.
func (cb *ContextBuilder) ValidatePlanStructure() error {
	return cb.memory.ValidatePlanStructure()
}

// SetPlanStatus sets the plan status.
func (cb *ContextBuilder) SetPlanStatus(status string) error {
	return cb.memory.SetStatus(status)
}

// GetPlanWorkDir returns the WorkDir from the plan metadata, or "".
func (cb *ContextBuilder) GetPlanWorkDir() string {
	return cb.memory.GetPlanWorkDir()
}

// GetPlanTaskName returns the task description from the plan metadata, or "".
func (cb *ContextBuilder) GetPlanTaskName() string {
	return cb.memory.GetPlanTaskName()
}
