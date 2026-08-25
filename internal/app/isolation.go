package app

// isolationRequest describes one invocation's OS-enforced boundary. The
// provider decides which process-scoped grants the staged client receives; the
// workspace and provider home are always the only writable trees.
type isolationRequest struct {
	Provider           string
	Workspace          string
	ProviderHome       string
	RuntimeExecutables []string
}
