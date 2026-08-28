package app

import (
	"fmt"
	"sort"
	"strings"
)

// modelPolicySchemaVersion is the only schema this binary accepts. A newer
// bundle must be run by the binary that introduced it, never by silently
// ignoring fields this version does not understand.
const modelPolicySchemaVersion = 1

const providerCodex = "codex"
const providerClaudeCode = "claude_code"

// policyRoles is the exact role set a schema-version-1 policy must declare.
// Every controller-launched invocation resolves its profile by these keys, so
// a missing key cannot fall back and an unknown key cannot be silently
// ignored. Human Editor is a human role and deliberately has no profile.
var policyRoles = []string{
	"pm",
	"researcher",
	"story_editor",
	"visual_editor",
	"writer",
	"reviewer_evidence",
	"reviewer_story",
	"reviewer_clarity",
	"reviewer_copy",
}

// providerEfforts records the reasoning-effort vocabulary each provider CLI
// accepts. There is deliberately no model allowlist: exact model availability
// is decided by the selected CLI, which fails the run rather than falling back.
var providerEfforts = map[string][]string{
	providerCodex:      {"minimal", "low", "medium", "high"},
	providerClaudeCode: {"low", "medium", "high", "xhigh", "max"},
}

// providerForbiddenModelPrefix rejects a model that plainly belongs to the
// other provider before any CLI is started.
var providerForbiddenModelPrefix = map[string]string{
	providerCodex:      "claude",
	providerClaudeCode: "gpt",
}

type roleProfile struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort"`
}

type modelPolicyDocument struct {
	SchemaVersion int                    `json:"schema_version"`
	Roles         map[string]roleProfile `json:"roles"`
}

// modelPolicy is a validated policy bound to the exact bytes it was validated
// from. Both the durable run copy and every launched argv are produced from
// this value, so they cannot disagree.
type modelPolicy struct {
	data   []byte
	digest string
	roles  map[string]roleProfile
}

// parseModelPolicy validates a bundle policy completely. It is called before
// the run directory is created, so every rejection here is a pre-run failure.
func parseModelPolicy(data []byte) (*modelPolicy, error) {
	var document modelPolicyDocument
	if err := decodeStrictJSON(data, &document); err != nil {
		return nil, fmt.Errorf("invalid models.json: %w", err)
	}
	if document.SchemaVersion != modelPolicySchemaVersion {
		return nil, fmt.Errorf("unsupported models.json schema_version %d: this binary supports %d", document.SchemaVersion, modelPolicySchemaVersion)
	}
	if document.Roles == nil {
		return nil, fmt.Errorf("models.json is missing the roles object")
	}
	required := make(map[string]bool, len(policyRoles))
	for _, role := range policyRoles {
		required[role] = true
	}
	var unknown []string
	for role := range document.Roles {
		if !required[role] {
			unknown = append(unknown, role)
		}
	}
	if len(unknown) != 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("models.json declares unknown role(s) %s: schema version %d supports exactly %s",
			strings.Join(unknown, ", "), modelPolicySchemaVersion, strings.Join(policyRoles, ", "))
	}
	var missing []string
	for _, role := range policyRoles {
		if _, found := document.Roles[role]; !found {
			missing = append(missing, role)
		}
	}
	if len(missing) != 0 {
		return nil, fmt.Errorf("models.json is missing required role profile(s) %s", strings.Join(missing, ", "))
	}
	roles := make(map[string]roleProfile, len(document.Roles))
	for _, role := range policyRoles {
		profile, err := validateRoleProfile(role, document.Roles[role])
		if err != nil {
			return nil, err
		}
		roles[role] = profile
	}
	return &modelPolicy{data: append([]byte(nil), data...), digest: revisionFor(data), roles: roles}, nil
}

func validateRoleProfile(role string, profile roleProfile) (roleProfile, error) {
	efforts, supported := providerEfforts[profile.Provider]
	if !supported {
		return roleProfile{}, fmt.Errorf("models.json role %q declares unsupported provider %q: schema version %d supports %s and %s",
			role, profile.Provider, modelPolicySchemaVersion, providerClaudeCode, providerCodex)
	}
	model := strings.TrimSpace(profile.Model)
	if model == "" {
		return roleProfile{}, fmt.Errorf("models.json role %q declares an empty model", role)
	}
	if model != profile.Model {
		return roleProfile{}, fmt.Errorf("models.json role %q declares a padded model %q", role, profile.Model)
	}
	if prefix := providerForbiddenModelPrefix[profile.Provider]; strings.HasPrefix(strings.ToLower(model), prefix) {
		return roleProfile{}, fmt.Errorf("models.json role %q assigns %s model %q to provider %q", role, prefix, model, profile.Provider)
	}
	valid := false
	for _, effort := range efforts {
		if profile.ReasoningEffort == effort {
			valid = true
			break
		}
	}
	if !valid {
		return roleProfile{}, fmt.Errorf("models.json role %q declares reasoning_effort %q, which provider %q does not accept: use one of %s",
			role, profile.ReasoningEffort, profile.Provider, strings.Join(efforts, ", "))
	}
	return roleProfile{Provider: profile.Provider, Model: model, ReasoningEffort: profile.ReasoningEffort}, nil
}

// profileFor returns the declared profile for a controller role key. There is
// no default and no sharing: an unmapped key is a controller defect and fails
// the run rather than reaching a CLI without an explicit profile.
func (policy *modelPolicy) profileFor(role string) (roleProfile, error) {
	if policy == nil {
		return roleProfile{}, fmt.Errorf("model policy is not loaded")
	}
	profile, found := policy.roles[role]
	if !found {
		return roleProfile{}, fmt.Errorf("no validated model profile for role %q", role)
	}
	return profile, nil
}

// usesProvider reports whether any declared role selects the provider. Only
// providers a policy actually references are resolved, staged, or preflighted.
func (policy *modelPolicy) usesProvider(provider string) bool {
	if policy == nil {
		return false
	}
	for _, profile := range policy.roles {
		if profile.Provider == provider {
			return true
		}
	}
	return false
}

func (profile roleProfile) describe() string {
	return fmt.Sprintf("provider=%s model=%s reasoning_effort=%s", profile.Provider, profile.Model, profile.ReasoningEffort)
}
