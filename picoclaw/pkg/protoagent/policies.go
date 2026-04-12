package protoagent

import (
	"fmt"
	"strings"
)

// generateOPAPolicies creates Open Policy Agent policies from security requirements.
func (e *Engine) generateOPAPolicies(reqs *RequirementsDocument) ([]PolicyDefinition, error) {
	var policies []PolicyDefinition

	// Generate RBAC policy if security requirements exist
	if len(reqs.SecurityRequirements) > 0 {
		rbacPolicy := e.generateRBACPolicy(reqs)
		policies = append(policies, rbacPolicy)
	}

	// Generate authorization policies from NFRs
	for _, nfr := range reqs.NonFunctionalRequirements {
		if nfr.Category == "security" {
			authPolicy := e.generateAuthorizationPolicy(nfr)
			if authPolicy != nil {
				policies = append(policies, *authPolicy)
			}
		}
	}

	// Generate data access policies
	dataPolicy := e.generateDataAccessPolicy(reqs)
	if dataPolicy != nil {
		policies = append(policies, *dataPolicy)
	}

	return policies, nil
}

// generateRBACPolicy creates a Role-Based Access Control policy.
func (e *Engine) generateRBACPolicy(reqs *RequirementsDocument) PolicyDefinition {
	// Collect all roles from security requirements
	roleSet := make(map[string]bool)
	permissionSet := make(map[string]bool)

	for _, secReq := range reqs.SecurityRequirements {
		for _, role := range secReq.Roles {
			roleSet[role] = true
		}
		for _, perm := range secReq.Permissions {
			permissionSet[perm] = true
		}
	}

	// Add default roles if none specified
	if len(roleSet) == 0 {
		roleSet["admin"] = true
		roleSet["user"] = true
		roleSet["viewer"] = true
	}

	// Build Rego policy
	var rego strings.Builder
	rego.WriteString("package authz.rbac\n\n")
	rego.WriteString("# Auto-generated RBAC policy from requirements\n\n")
	
	rego.WriteString("# Default deny\n")
	rego.WriteString("default allow = false\n\n")
	
	rego.WriteString("# Role definitions\n")
	rego.WriteString("roles := {\n")
	for role := range roleSet {
		rego.WriteString(fmt.Sprintf("  \"%s\",\n", role))
	}
	rego.WriteString("}\n\n")

	rego.WriteString("# Permission definitions\n")
	rego.WriteString("permissions := {\n")
	for perm := range permissionSet {
		rego.WriteString(fmt.Sprintf("  \"%s\",\n", perm))
	}
	rego.WriteString("}\n\n")

	rego.WriteString("# Role-permission mapping\n")
	rego.WriteString("role_permissions := {\n")
	rego.WriteString("  \"admin\": {\"read\", \"write\", \"delete\", \"admin\"},\n")
	rego.WriteString("  \"user\": {\"read\", \"write\"},\n")
	rego.WriteString("  \"viewer\": {\"read\"}\n")
	rego.WriteString("}\n\n")

	rego.WriteString("# Allow if user has required permission\n")
	rego.WriteString("allow {\n")
	rego.WriteString("  some role in input.user.roles\n")
	rego.WriteString("  some perm in role_permissions[role]\n")
	rego.WriteString("  perm == input.permission\n")
	rego.WriteString("}\n\n")

	rego.WriteString("# Admin bypass\n")
	rego.WriteString("allow {\n")
	rego.WriteString("  some role in input.user.roles\n")
	rego.WriteString("  role == \"admin\"\n")
	rego.WriteString("}\n")

	return PolicyDefinition{
		Name:        "rbac_policy",
		Package:     "authz.rbac",
		Description: "Role-Based Access Control policy",
		Rego:        rego.String(),
	}
}

// generateAuthorizationPolicy creates an authorization policy from NFR.
func (e *Engine) generateAuthorizationPolicy(nfr NonFunctionalRequirement) *PolicyDefinition {
	if len(nfr.Constraints) == 0 {
		return nil
	}

	var rego strings.Builder
	rego.WriteString("package authz.custom\n\n")
	rego.WriteString(fmt.Sprintf("# Policy: %s\n", nfr.Name))
	rego.WriteString(fmt.Sprintf("# Description: %s\n\n", nfr.Description))

	rego.WriteString("default allow = false\n\n")

	// Generate rules from constraints
	for constraint, value := range nfr.Constraints {
		ruleName := strings.ReplaceAll(strings.ToLower(constraint), " ", "_")
		
		rego.WriteString(fmt.Sprintf("%s {\n", ruleName))
		rego.WriteString(fmt.Sprintf("  input.%s == \"%s\"\n", constraint, value))
		rego.WriteString("}\n\n")
	}

	rego.WriteString("allow {\n")
	for constraint := range nfr.Constraints {
		ruleName := strings.ReplaceAll(strings.ToLower(constraint), " ", "_")
		rego.WriteString(fmt.Sprintf("  %s\n", ruleName))
	}
	rego.WriteString("}\n")

	return &PolicyDefinition{
		Name:        fmt.Sprintf("%s_policy", strings.ToLower(nfr.Name)),
		Package:     "authz.custom",
		Description: nfr.Description,
		Rego:        rego.String(),
	}
}

// generateDataAccessPolicy creates data access control policies.
func (e *Engine) generateDataAccessPolicy(reqs *RequirementsDocument) *PolicyDefinition {
	if len(reqs.SecurityRequirements) == 0 {
		return nil
	}

	var hasDataClassification bool
	for _, secReq := range reqs.SecurityRequirements {
		if secReq.DataClassification != "" {
			hasDataClassification = true
			break
		}
	}

	if !hasDataClassification {
		return nil
	}

	var rego strings.Builder
	rego.WriteString("package authz.data_access\n\n")
	rego.WriteString("# Data access control policy based on classification\n\n")
	
	rego.WriteString("default allow = false\n\n")

	rego.WriteString("# Allow access based on data classification\n")
	rego.WriteString("allow {\n")
	rego.WriteString("  input.data_classification == \"public\"\n")
	rego.WriteString("}\n\n")

	rego.WriteString("allow {\n")
	rego.WriteString("  input.data_classification == \"internal\"\n")
	rego.WriteString("  input.user.clearance_level >= 1\n")
	rego.WriteString("}\n\n")

	rego.WriteString("allow {\n")
	rego.WriteString("  input.data_classification == \"confidential\"\n")
	rego.WriteString("  input.user.clearance_level >= 2\n")
	rego.WriteString("  input.user.department == input.data.owner_department\n")
	rego.WriteString("}\n\n")

	rego.WriteString("allow {\n")
	rego.WriteString("  input.data_classification == \"restricted\"\n")
	rego.WriteString("  input.user.clearance_level >= 3\n")
	rego.WriteString("  input.purpose == \"authorized\"\n")
	rego.WriteString("}\n")

	return &PolicyDefinition{
		Name:        "data_access_policy",
		Package:     "authz.data_access",
		Description: "Data access control based on classification levels",
		Rego:        rego.String(),
	}
}

// validateOPAPolicies validates generated OPA policies.
func (e *Engine) validateOPAPolicies(policies []PolicyDefinition) []ValidationError {
	var errors []ValidationError

	for i, policy := range policies {
		// Check for required fields
		if policy.Package == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("policies[%d].package", i),
				Message: "Package is required",
			})
		}

		if policy.Rego == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("policies[%d].rego", i),
				Message: "Rego code is required",
			})
		}

		// Basic syntax validation
		if !strings.Contains(policy.Rego, "package ") {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("policies[%d].rego", i),
				Message: "Missing package declaration",
			})
		}
	}

	return errors
}
