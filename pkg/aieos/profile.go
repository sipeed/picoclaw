package aieos

// Profile represents an AIEOS v1.1 (AI Entity Object Specification) profile.
// It defines the identity, personality, and behavior of an AI agent through
// structured JSON rather than free-form markdown.
type Profile struct {
	Version      string       `json:"version"`
	Identity     Identity     `json:"identity"`
	Capabilities []Capability `json:"capabilities,omitempty"`
	Psychology   *Psychology  `json:"psychology,omitempty"`
	Linguistics  *Linguistics `json:"linguistics,omitempty"`
	Motivations  *Motivations `json:"motivations,omitempty"`
	Boundaries   *Boundaries  `json:"boundaries,omitempty"`
	Metadata     *Metadata    `json:"metadata,omitempty"`
}

// Identity defines who the agent is.
type Identity struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
	Philosophy  string `json:"philosophy,omitempty"`
}

// Capability describes a single agent capability.
type Capability struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Psychology holds OCEAN personality traits, each in the range 0.0-1.0.
type Psychology struct {
	Openness          float64 `json:"openness"`
	Conscientiousness float64 `json:"conscientiousness"`
	Extraversion      float64 `json:"extraversion"`
	Agreeableness     float64 `json:"agreeableness"`
	Neuroticism       float64 `json:"neuroticism"`
}

// Linguistics controls the agent's communication style.
type Linguistics struct {
	Formality float64  `json:"formality"` // 0.0 (casual) to 1.0 (formal)
	Verbosity float64  `json:"verbosity"` // 0.0 (terse) to 1.0 (verbose)
	Idiolect  []string `json:"idiolect,omitempty"`
}

// Motivations captures the agent's values and goals.
type Motivations struct {
	Values []string `json:"values,omitempty"`
	Goals  []string `json:"goals,omitempty"`
}

// Boundaries defines hard and soft behavioral limits.
type Boundaries struct {
	HardLimits []string `json:"hard_limits,omitempty"`
	SoftLimits []string `json:"soft_limits,omitempty"`
}

// Metadata holds authorship and licensing information.
type Metadata struct {
	Author    string `json:"author,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	License   string `json:"license,omitempty"`
}
