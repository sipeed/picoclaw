package aieos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const defaultFilename = "aieos.json"

// DefaultProfilePath returns the default AIEOS profile path for a workspace.
func DefaultProfilePath(workspace string) string {
	return filepath.Join(workspace, defaultFilename)
}

// ProfileExists checks whether an aieos.json file exists in the workspace.
func ProfileExists(workspace string) bool {
	_, err := os.Stat(DefaultProfilePath(workspace))
	return err == nil
}

// LoadProfile reads and validates an AIEOS profile from the given path.
func LoadProfile(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("aieos: read profile: %w", err)
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("aieos: parse profile: %w", err)
	}

	if err := validate(&p); err != nil {
		return nil, err
	}

	return &p, nil
}

// validate performs minimal validation on a loaded profile.
func validate(p *Profile) error {
	if p.Version == "" {
		return fmt.Errorf("aieos: version is required")
	}
	if p.Identity.Name == "" {
		return fmt.Errorf("aieos: identity.name is required")
	}
	if p.Psychology != nil {
		if err := validateOCEAN(p.Psychology); err != nil {
			return err
		}
	}
	return nil
}

// validateOCEAN checks that all OCEAN trait values are in [0.0, 1.0].
func validateOCEAN(psy *Psychology) error {
	traits := map[string]float64{
		"openness":          psy.Openness,
		"conscientiousness": psy.Conscientiousness,
		"extraversion":      psy.Extraversion,
		"agreeableness":     psy.Agreeableness,
		"neuroticism":       psy.Neuroticism,
	}
	for name, val := range traits {
		if val < 0.0 || val > 1.0 {
			return fmt.Errorf("aieos: psychology.%s must be in [0.0, 1.0], got %v", name, val)
		}
	}
	return nil
}
