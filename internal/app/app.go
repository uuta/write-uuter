package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BriefPath        string
	RunDir           string
	CodexExecutable  string
	ClaudeExecutable string
	TmuxExecutable   string
	// CodexExecutableSet/ClaudeExecutableSet/TmuxExecutableSet record that the
	// caller passed the override explicitly, so an explicit empty value fails
	// instead of silently falling back to the PATH default.
	CodexExecutableSet  bool
	ClaudeExecutableSet bool
	TmuxExecutableSet   bool
	AgentTimeout        time.Duration
	PromptsDir          string
	PromptsDirSet       bool
}

type decisionBinding struct {
	RequestID      string
	Digest         string
	DecisionDigest string
}

type controller struct {
	config  Config
	brief   briefDocument
	runDir  string
	prompts *promptBundle
	policy  *modelPolicy
	// providerExecutables holds the resolved executable for every provider the
	// validated policy references, and nothing else.
	providerExecutables map[string]string
	contentRoot         string
	workflow            Workflow
	store               *artifactStore
	runtime             *tmuxRuntime
	pm                  invocation
	reachedLenses       map[int][]string
	decisionBindings    map[int]map[string]decisionBinding
	// screenshotRequests is the validated Researcher request list. It is empty
	// for every run that asked for no screenshot, and that run never reads a
	// Cloudflare credential.
	screenshotRequests []ScreenshotRequest
	// screenshotManifest is the exact controller-generated manifest handed to
	// later roles as read-only context.
	screenshotManifest []byte
	// visualInputs is the validated pool of local images this run staged. It
	// is empty for every brief without a `## Visual inputs` section whose
	// Researcher requested no screenshot, and the bytes are the controller's
	// private copies rather than re-reads of the original paths.
	visualInputs []visualInput
	// visualInputManifest is the exact controller-generated input manifest
	// handed to the Visual Editor as read-only context.
	visualInputManifest []byte
	// visualPlans records the validated plan accepted for each candidate.
	visualPlans map[int]VisualPlan
	// visualAssets records the durable assets each accepted plan placed.
	visualAssets map[int][]VisualAssetRecord
	// publishedArticle is the exact file identity this controller committed as
	// article.md. Rollback removes only that identity.
	publishedArticle os.FileInfo
}

func Run(config Config) error {
	if strings.TrimSpace(config.BriefPath) == "" || strings.TrimSpace(config.RunDir) == "" {
		return fmt.Errorf("--brief and --run-dir are required")
	}
	if config.AgentTimeout <= 0 {
		return fmt.Errorf("--timeout must be greater than zero")
	}
	if config.CodexExecutableSet && strings.TrimSpace(config.CodexExecutable) == "" {
		return fmt.Errorf("--codex was given an empty value")
	}
	if config.ClaudeExecutableSet && strings.TrimSpace(config.ClaudeExecutable) == "" {
		return fmt.Errorf("--claude was given an empty value")
	}
	if config.TmuxExecutableSet && strings.TrimSpace(config.TmuxExecutable) == "" {
		return fmt.Errorf("--tmux was given an empty value")
	}
	if config.CodexExecutable == "" {
		config.CodexExecutable = "codex"
	}
	if config.ClaudeExecutable == "" {
		config.ClaudeExecutable = "claude"
	}
	if config.TmuxExecutable == "" {
		config.TmuxExecutable = "tmux"
	}
	briefData, err := os.ReadFile(config.BriefPath)
	if err != nil {
		return fmt.Errorf("read brief: %w", err)
	}
	brief, err := parseBrief(string(briefData))
	if err != nil {
		return err
	}
	prompts, err := openPromptsBundle(config.PromptsDir, config.PromptsDirSet || config.PromptsDir != "")
	if err != nil {
		return err
	}
	defer prompts.Close()
	// The policy is bound to the same no-follow bundle boundary as the role
	// prompts and validated completely before anything is created. Every
	// rejection below therefore happens before the run directory exists and
	// before tmux is started.
	policyData, err := prompts.load(modelPolicyFile)
	if err != nil {
		return fmt.Errorf("required model policy %s: %w", modelPolicyFile, err)
	}
	policy, err := parseModelPolicy([]byte(policyData))
	if err != nil {
		return err
	}
	runDir, err := filepath.Abs(config.RunDir)
	if err != nil {
		return fmt.Errorf("resolve run directory: %w", err)
	}
	if _, err := os.Lstat(runDir); err == nil {
		return fmt.Errorf("run directory already exists: %s", runDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect run directory: %w", err)
	}
	providerExecutables := map[string]string{}
	if policy.usesProvider(providerCodex) {
		providerExecutables[providerCodex] = config.CodexExecutable
	}
	if policy.usesProvider(providerClaudeCode) {
		claudePath, resolveErr := resolveClaudeExecutable(config.ClaudeExecutable)
		if resolveErr != nil {
			return resolveErr
		}
		// The exact executable that passes the Max preflight is the one that is
		// staged and launched.
		if err := verifyClaudeMaxSubscription(claudePath); err != nil {
			return err
		}
		providerExecutables[providerClaudeCode] = claudePath
	}
	contentRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve content root: %w", err)
	}
	contentRoot, err = filepath.Abs(contentRoot)
	if err != nil {
		return fmt.Errorf("resolve content root: %w", err)
	}
	control := &controller{
		config: config, brief: brief, runDir: runDir, prompts: prompts, contentRoot: contentRoot,
		policy: policy, providerExecutables: providerExecutables,
		reachedLenses: make(map[int][]string), decisionBindings: make(map[int]map[string]decisionBinding),
		visualPlans: make(map[int]VisualPlan), visualAssets: make(map[int][]VisualAssetRecord),
	}
	// Optional visual inputs are validated and privately copied here, before
	// the run directory exists and before any agent starts, so an unsafe or
	// unusable path leaves no partial state behind.
	if err := control.loadVisualInputs(); err != nil {
		return err
	}
	if err := control.initialize(briefData); err != nil {
		return err
	}
	defer control.store.Close()
	return control.execute()
}

func (control *controller) initialize(briefData []byte) error {
	parent := filepath.Dir(control.runDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create run parent: %w", err)
	}
	temporary, err := os.MkdirTemp(parent, "."+filepath.Base(control.runDir)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary run workspace: %w", err)
	}
	keepTemporary := false
	defer func() {
		if !keepTemporary {
			_ = os.RemoveAll(temporary)
		}
	}()
	store, err := openArtifactStore(temporary)
	if err != nil {
		return err
	}
	keepStore := false
	defer func() {
		if !keepStore {
			_ = store.Close()
		}
	}()
	for _, relative := range []string{"evidence/assets", "drafts", "visuals", "reviews", "pm-decisions", ".control/prompts", ".control/exits", ".control/logs", ".control/invocations"} {
		if err := store.mkdirAll(relative, 0o755); err != nil {
			return fmt.Errorf("initialize workspace: %w", err)
		}
	}
	if err := store.writeAtomic("brief.md", briefData, 0o644); err != nil {
		return fmt.Errorf("copy brief: %w", err)
	}
	// The run keeps the exact validated policy bytes, and workflow.json keeps
	// their digest, so a completed or blocked run can prove which policy
	// produced it.
	if err := store.writeAtomic(modelPolicyArtifact, control.policy.data, 0o444); err != nil {
		return fmt.Errorf("copy model policy: %w", err)
	}
	now := time.Now().UTC()
	control.workflow = Workflow{
		SchemaVersion: workflowSchemaVersion, Status: "running", Phase: "initializing",
		ModelPolicyDigest: control.policy.digest,
		ArtifactPaths: map[string]string{
			"brief": "brief.md", "workflow": "workflow.json", "sources": "evidence/sources.md",
			"model_policy": modelPolicyArtifact, "invocations": ".control/invocations",
			"firsthand": "evidence/firsthand.md", "assets": "evidence/assets", "claim_ledger": "claim-ledger.md",
			"screenshot_requests": screenshotRequestArtifact, "screenshots": screenshotManifestArtifact,
			"screenshot_assets": screenshotAssetDir,
			"outline":           "outline.md", "drafts": "drafts", "reviews": "reviews",
			"visual_inputs": visualInputManifestArtifact, "visuals": "visuals",
			"pm_decisions": "pm-decisions", "article": "article.md",
		},
		StartedAt: now, UpdatedAt: now,
	}
	if err := store.writeJSON("workflow.json", control.workflow); err != nil {
		return fmt.Errorf("write initial workflow: %w", err)
	}
	if err := waitForCommitBarrier(); err != nil {
		return err
	}
	committed, err := renameNoReplacePath(temporary, control.runDir)
	if committed {
		// The rename consumed the temporary name. Never remove that name
		// again: anything reappearing under it is no longer this workspace.
		keepTemporary = true
	}
	if err != nil {
		if committed {
			// The rename already made the target visible, so this run owns the
			// committed identity even though the durability barrier failed.
			// Adopt the store and terminalize through the ordinary blocked
			// path; leaving the visible target at running/initializing would
			// strand it, since a retry refuses an existing run directory.
			// keepStore stays false on purpose: the deferred cleanup closes the
			// adopted store once the terminal state has been persisted.
			control.store = store
			return control.block(fmt.Sprintf("durably commit run workspace at %s: %v", control.runDir, err))
		}
		if _, targetErr := os.Lstat(control.runDir); targetErr == nil {
			return fmt.Errorf("run directory already exists: %s", control.runDir)
		}
		return fmt.Errorf("atomically commit run workspace without replacement: %w", err)
	}
	keepTemporary = true
	keepStore = true
	control.store = store
	return nil
}

func waitForCommitBarrier() error {
	barrier := os.Getenv("WRITE_UUTER_TEST_COMMIT_BARRIER")
	if barrier == "" {
		return nil
	}
	if err := os.WriteFile(barrier+".ready", []byte("ready\n"), 0o600); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(barrier + ".continue"); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("test commit barrier timed out")
}

func (control *controller) execute() error {
	pmDeadline := invocationDeadline(control.config.AgentTimeout)
	runtime, err := newTmuxRuntime(control.config.TmuxExecutable, control.providerExecutables, control.config.AgentTimeout, control.runDir)
	if err != nil {
		return control.block(err.Error())
	}
	control.runtime = runtime
	if err := ensureInvocationDeadline(pmDeadline, "PM launch preparation"); err != nil {
		return control.block(err.Error())
	}
	pmPrompt, err := control.buildPMPrompt()
	if err != nil {
		return control.block(err.Error())
	}
	pmProfile, err := control.policy.profileFor("pm")
	if err != nil {
		return control.block(err.Error())
	}
	control.pm, err = control.runtime.prepareInvocation("pm", "", 0, "", pmPrompt, pmProfile)
	if err != nil {
		return control.block(err.Error())
	}
	if err := control.publishInvocationAudit(control.pm); err != nil {
		return control.block(err.Error())
	}
	if err := ensureInvocationDeadline(pmDeadline, "PM launch preparation"); err != nil {
		return control.block(err.Error())
	}
	pmStore, err := control.runtime.workspaceStore(control.pm)
	if err != nil {
		return control.block(err.Error())
	}
	if err := pmStore.mkdirAll("inbox", 0o700); err == nil {
		err = pmStore.mkdirAll("outbox", 0o700)
	}
	_ = pmStore.Close()
	if err != nil {
		return control.block(err.Error())
	}
	if err := control.runtime.startPM(control.pm, pmDeadline); err != nil {
		return control.block(err.Error())
	}
	if err := control.runResearcher(); err != nil {
		return control.block(err.Error())
	}
	// Screenshot capture happens here, before any later role starts, so a
	// failed or unsafe request blocks the run rather than reaching the Writer.
	if err := control.captureScreenshots(); err != nil {
		return control.block(err.Error())
	}
	// Captured screenshots join the visual input pool, so the visual pass
	// consumes evidence in place instead of re-acquiring anything.
	if err := control.publishVisualInputs(); err != nil {
		return control.block(err.Error())
	}
	if err := control.runStoryEditor(); err != nil {
		return control.block(err.Error())
	}
	if err := control.produceCandidate(1); err != nil {
		return control.block(err.Error())
	}
	for candidate := 1; candidate <= 3; candidate++ {
		mustFix, blockReason, err := control.reviewCandidate(candidate)
		if err != nil {
			return control.block(err.Error())
		}
		if blockReason != "" {
			return control.block(blockReason)
		}
		if !mustFix {
			return control.succeed(candidate)
		}
		if candidate == 3 {
			return control.block("review budget exhausted: candidate article-003 has validated must-fix findings")
		}
		if err := control.produceCandidate(candidate + 1); err != nil {
			return control.block(err.Error())
		}
	}
	return control.block("review loop ended without a terminal result")
}

func (control *controller) runResearcher() error {
	base, err := control.prompts.load("researcher.md")
	if err != nil {
		return err
	}
	sourceHints, err := control.localSourceHints()
	if err != nil {
		return err
	}
	// The staged hints are the Researcher's only copy of the local sources, and
	// a model-invoked shell, glob, or directory listing is denied by the
	// sandbox on purpose. The assignment therefore names every staged path
	// exactly, so the role reads each one directly instead of discovering it.
	// localSourceHints stores them in an unordered map, so the prompt sorts the
	// names and is deterministic for the same brief.
	hintNames := make([]string, 0, len(sourceHints))
	for relative := range sourceHints {
		hintNames = append(hintNames, relative)
	}
	sort.Strings(hintNames)
	prompt := base + contextBlock("brief.md", []byte(control.brief.Raw))
	prompt += "\n\n## Assignment\n\nThe brief is staged at `context/brief.md`."
	if len(hintNames) == 0 {
		prompt += " This run staged no local source hint, so work from the brief and any URL hints it names."
	} else {
		prompt += fmt.Sprintf(" The controller resolved and staged %d local source hint(s) as read-only regular"+
			"\nfiles. The following list is complete and exact. Read each path directly with your file-read capability; no"+
			"\ndirectory listing, glob, or shell command is needed, and none is available to you.\n", len(hintNames))
		for _, relative := range hintNames {
			prompt += "\n- `" + filepath.ToSlash(filepath.Join("context", "source-hints", relative)) + "`"
		}
	}
	return control.runWorker("researcher", "", 0, "", "research", prompt,
		func(workspace *artifactStore) error {
			if err := workspace.writeAtomic("context/brief.md", []byte(control.brief.Raw), 0o444); err != nil {
				return err
			}
			for _, relative := range hintNames {
				if err := workspace.writeAtomic(filepath.Join("context", "source-hints", relative), sourceHints[relative], 0o444); err != nil {
					return err
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			if _, err := workspace.readNonEmpty("evidence/sources.md"); err != nil {
				return err
			}
			ledger, err := workspace.readNonEmpty("claim-ledger.md")
			if err != nil {
				return err
			}
			lower := strings.ToLower(string(ledger))
			for _, classification := range []string{"fact", "firsthand observation", "inference", "opinion", "unresolved"} {
				if !strings.Contains(lower, classification) {
					return fmt.Errorf("claim-ledger.md does not distinguish %s", classification)
				}
			}
			return control.validateScreenshotRequests(workspace, ledger)
		},
		func(workspace *artifactStore) error {
			if _, err := control.store.copyRegularFrom(workspace, "evidence/sources.md", "evidence/sources.md", 0o644); err != nil {
				return err
			}
			if _, err := control.store.copyRegularFrom(workspace, "claim-ledger.md", "claim-ledger.md", 0o644); err != nil {
				return err
			}
			if _, err := workspace.readRegular("evidence/firsthand.md"); err == nil {
				if _, err := control.store.copyRegularFrom(workspace, "evidence/firsthand.md", "evidence/firsthand.md", 0o644); err != nil {
					return err
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if len(control.screenshotRequests) != 0 {
				if _, err := control.store.copyRegularFrom(workspace, screenshotRequestArtifact, screenshotRequestArtifact, 0o444); err != nil {
					return err
				}
			}
			return control.store.copyTreeFrom(workspace, "evidence/assets", "evidence/assets")
		},
		func() error {
			control.screenshotRequests = nil
			return errors.Join(
				control.store.remove("evidence/sources.md"),
				control.store.remove("claim-ledger.md"),
				control.store.remove("evidence/firsthand.md"),
				control.store.remove(screenshotRequestArtifact),
				control.store.removeAll("evidence/assets"),
			)
		})
}

// validateScreenshotRequests reads and fully validates the optional Researcher
// screenshot artifact against the same claim ledger the Researcher produced.
// The controller-owned asset directory stays controller-owned: a Researcher
// that writes there fails the contract instead of forging provenance.
func (control *controller) validateScreenshotRequests(workspace *artifactStore, ledger []byte) error {
	if _, err := workspace.lstat(screenshotAssetDir); err == nil {
		return fmt.Errorf("%s is controller-owned and must not be written by the Researcher", screenshotAssetDir)
	}
	// Any other probe outcome - absent, or an unsafe parent component - is left
	// to the existing asset-tree validation performed when the tree is copied,
	// so this check never restates or masks that contract.
	data, err := workspace.readRegular(screenshotRequestArtifact)
	if errors.Is(err, os.ErrNotExist) {
		control.screenshotRequests = nil
		return nil
	}
	if err != nil {
		return err
	}
	requests, err := parseScreenshotRequests(data, ledger)
	if err != nil {
		return err
	}
	if len(requests) == 0 {
		// An explicit empty list is valid and means the same thing as no
		// artifact: no capture, and no Cloudflare credential is required.
		control.screenshotRequests = nil
		return nil
	}
	control.screenshotRequests = requests
	return nil
}

func (control *controller) localSourceHints() (map[string][]byte, error) {
	briefPath, err := filepath.Abs(control.config.BriefPath)
	if err != nil {
		return nil, err
	}
	hints := make(map[string][]byte)
	index := 0
	for _, line := range strings.Split(control.brief.Sections["Source hints"], "\n") {
		value := sourceHintValue(line)
		value = strings.Trim(value, "`<>")
		if value == "" || strings.Contains(value, "://") {
			continue
		}
		path := value
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(briefPath), path)
		}
		info, statErr := os.Lstat(path)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("stage source hint %s: %w", value, readErr)
		}
		index++
		name := fmt.Sprintf("%03d-%s", index, filepath.Base(path))
		hints[name] = data
	}
	return hints, nil
}

func sourceHintValue(line string) string {
	value := strings.TrimSpace(line)
	for _, marker := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(value, marker) {
			return strings.TrimSpace(strings.TrimPrefix(value, marker))
		}
	}
	if separator := strings.Index(value, ". "); separator > 0 {
		if _, err := strconv.Atoi(value[:separator]); err == nil {
			return strings.TrimSpace(value[separator+2:])
		}
	}
	return value
}

func (control *controller) runStoryEditor() error {
	base, err := control.prompts.load("story-editor.md")
	if err != nil {
		return err
	}
	prompt := base + contextBlock("brief.md", []byte(control.brief.Raw))
	inputs := []string{"evidence/sources.md", "claim-ledger.md"}
	for _, relative := range inputs {
		data, readErr := control.store.readRegular(relative)
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(relative, data)
	}
	return control.runWorker("story_editor", "", 0, "", "story", prompt,
		func(workspace *artifactStore) error {
			if err := workspace.writeAtomic("context/brief.md", []byte(control.brief.Raw), 0o444); err != nil {
				return err
			}
			for _, relative := range inputs {
				data, readErr := control.store.readRegular(relative)
				if readErr != nil {
					return readErr
				}
				if err := workspace.writeAtomic(filepath.Join("context", relative), data, 0o444); err != nil {
					return err
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			outline, readErr := workspace.readNonEmpty("outline.md")
			if readErr != nil {
				return readErr
			}
			lower := strings.ToLower(string(outline))
			for _, field := range []string{"purpose", "supporting evidence", "reader takeaway"} {
				if !strings.Contains(lower, field) {
					return fmt.Errorf("outline.md is missing section field %q", field)
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			_, err := control.store.copyRegularFrom(workspace, "outline.md", "outline.md", 0o644)
			return err
		},
		func() error {
			return control.store.remove("outline.md")
		})
}

// produceCandidate runs the canonical per-candidate sequence: a Writer prose
// draft, a fresh sequential Visual Editor, then a fresh Writer assembly pass.
// Only the assembled candidate enters review and may become article.md. The
// visual pass never consumes one of the three review candidates.
func (control *controller) produceCandidate(candidate int) error {
	if err := control.runWriter(candidate); err != nil {
		return err
	}
	if err := control.runVisualEditor(candidate); err != nil {
		return err
	}
	return control.runWriterAssembly(candidate)
}

// runWriter produces the prose draft of one candidate. The Writer owns article
// Markdown throughout, but diagrams and images are placed by the later
// assembly pass, so the prose draft carries explanation only.
func (control *controller) runWriter(candidate int) error {
	base, err := control.prompts.load("writer.md")
	if err != nil {
		return err
	}
	target := proseDraftPath(candidate)
	prompt := base + fmt.Sprintf("\n\n## Assignment\n\nWrite the prose draft of candidate %03d to `%s` in this isolated workspace."+
		"\n\nWrite explanation only. Do not add a Mermaid block or an image reference: a separate Visual Editor pass evaluates"+
		"\nvisual opportunities next, and a fresh Writer assembly invocation places the approved visuals and shortens the prose"+
		"\nthey replace.", candidate, target)
	inputs := []string{"brief.md", "evidence/sources.md", "claim-ledger.md", "outline.md"}
	if candidate > 1 {
		previous := candidate - 1
		inputs = append(inputs, candidateDraftPath(previous), visualPlanPath(previous), fmt.Sprintf("pm-decisions/article-%03d.md", previous))
		for _, lens := range control.reachedLenses[previous] {
			inputs = append(inputs,
				filepath.Join("reviews", fmt.Sprintf("article-%03d", previous), lens, "result.json"),
				filepath.Join("reviews", fmt.Sprintf("article-%03d", previous), lens, "report.md"),
			)
		}
	}
	for _, relative := range inputs {
		data, readErr := control.store.readRegular(relative)
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(relative, data)
	}
	styleGuide, err := loadStyleGuide(control.contentRoot)
	if err != nil {
		return err
	}
	if len(styleGuide) != 0 {
		prompt += contextBlock("style-guide.md", styleGuide)
	}
	// Validated screenshot evidence is read-only Writer context. The manifest
	// is text and joins the prompt; the images are staged as files only.
	screenshotManifest, screenshotImages, err := control.screenshotContext()
	if err != nil {
		return err
	}
	if len(screenshotManifest) != 0 {
		prompt += contextBlock(screenshotManifestArtifact, screenshotManifest)
	}
	previousProseRevision := control.workflow.CurrentProseRevision
	err = control.runWorker("writer", "", candidate, "", "writing", prompt,
		func(workspace *artifactStore) error {
			for _, relative := range inputs {
				data, readErr := control.store.readRegular(relative)
				if readErr != nil {
					return readErr
				}
				if err := workspace.writeAtomic(filepath.Join("context", relative), data, 0o444); err != nil {
					return err
				}
			}
			if len(styleGuide) != 0 {
				if err := workspace.writeAtomic(filepath.Join("context", "style-guide.md"), styleGuide, 0o444); err != nil {
					return err
				}
			}
			if len(screenshotManifest) != 0 {
				if err := workspace.writeAtomic(filepath.Join("context", screenshotManifestArtifact), screenshotManifest, 0o444); err != nil {
					return err
				}
				if err := stageScreenshotImages(workspace, screenshotImages); err != nil {
					return err
				}
			}
			return nil
		},
		func(workspace *artifactStore) error {
			data, readErr := workspace.readNonEmpty(target)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(strings.ToLower(string(data)), "todo") {
				return fmt.Errorf("candidate contains unresolved TODO placeholder")
			}
			return validateProseDraft(data)
		},
		func(workspace *artifactStore) error {
			data, copyErr := control.store.copyRegularFrom(workspace, target, target, 0o644)
			if copyErr != nil {
				return copyErr
			}
			control.workflow.CurrentProseRevision = revisionFor(data)
			return nil
		},
		func() error {
			control.workflow.CurrentProseRevision = previousProseRevision
			return control.store.remove(target)
		})
	return err
}

// runVisualEditor runs one fresh sequential Visual Editor per candidate. It
// receives only the brief, outline, claim ledger, current prose draft, and the
// controller-staged visual inputs, and it never writes a durable candidate: it
// emits an inspectable plan the controller validates before the assembly pass
// applies it.
func (control *controller) runVisualEditor(candidate int) error {
	base, err := control.prompts.load("visual-editor.md")
	if err != nil {
		return err
	}
	proseRelative := proseDraftPath(candidate)
	proseData, err := control.store.readRegular(proseRelative)
	if err != nil {
		return err
	}
	proseRevision := revisionFor(proseData)
	if proseRevision != control.workflow.CurrentProseRevision {
		return fmt.Errorf("prose draft revision changed outside the controller")
	}
	available := make(map[string]visualInput, len(control.visualInputs))
	for _, input := range control.visualInputs {
		available[input.ID] = input
	}
	prompt := base + fmt.Sprintf("\n\n## Assignment\n\nCandidate: `article-%03d`\nSource prose revision: `%s`"+
		"\n\nWrite exactly two files in this workspace root:"+
		"\n\n- `plan.md`: one entry per evaluated opportunity, naming its ID, its location in the prose draft, the selected"+
		"\n  action, and the concrete reason that action improves explanation or reading flow."+
		"\n- `plan.json`: the same decisions as `{\"schema_version\": %d, \"source_revision\": \"%s\", \"opportunities\": [...]}`."+
		"\n\nSupported actions are exactly %s. Use `source_revision` verbatim. Every entry needs an `id`, a `location`, an"+
		"\n`action`, and a `rationale`. Each `id` is unique within the plan and uses only letters, digits, `-`, and `_`,"+
		"\nnever starting with `-` or `_`."+
		"\n\n- A `mermaid` entry adds the diagram body in `mermaid`. Write the body only: no ``` fence anywhere in it."+
		"\n- An `existing_local_asset` entry adds a staged `asset_id` and meaningful `alt_text`. Each staged asset may be"+
		"\n  placed at most once, and `alt_text` must contain none of the characters `[`, `]`, `(`, or `)`, because it is"+
		"\n  embedded directly in `![alt text](path)`."+
		"\n- A `restructure_text` or `none` entry adds no `mermaid`, `asset_id`, or `alt_text`."+
		"\n\nRecord at least one evaluated opportunity even when no visual is appropriate."+
		"\n\nBounds: at most %d opportunities; `location` at most %d bytes; `rationale` at most %d bytes; `alt_text` at most"+
		"\n%d bytes; a `mermaid` body at most %d bytes. Unknown fields and duplicate keys are rejected.",
		candidate, proseRevision, visualSchemaVersion, proseRevision, strings.Join(visualPlanActions, ", "),
		visualMaxOpportunities, visualMaxLocationBytes, visualMaxRationaleBytes, visualMaxAltTextBytes, visualMaxMermaidBytes)
	if len(available) != 0 {
		prompt += fmt.Sprintf("\n\nStaged visual inputs are read-only regular files under `context/visual-inputs/`. Placing one copies it to"+
			"\n`%s`, keeping its extension, which is the exact relative path the assembled article will reference.",
			visualAssetPath(candidate, "<asset_id>", ".<ext>"))
	} else {
		prompt += "\n\nThis run staged no local image, so only `mermaid`, `restructure_text`, and `none` are available."
	}
	inputs := []string{"brief.md", "outline.md", "claim-ledger.md", proseRelative}
	for _, relative := range inputs {
		data, readErr := control.store.readRegular(relative)
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(relative, data)
	}
	if len(control.visualInputManifest) != 0 {
		prompt += contextBlock(visualInputManifestArtifact, control.visualInputManifest)
	}
	if len(control.screenshotManifest) != 0 {
		prompt += contextBlock(screenshotManifestArtifact, control.screenshotManifest)
	}
	var plan VisualPlan
	var planReport []byte
	return control.runWorker("visual_editor", "", candidate, proseRevision, "visual planning", prompt,
		func(workspace *artifactStore) error {
			for _, relative := range inputs {
				data, readErr := control.store.readRegular(relative)
				if readErr != nil {
					return readErr
				}
				if err := workspace.writeAtomic(filepath.Join("context", relative), data, 0o444); err != nil {
					return err
				}
			}
			if len(control.visualInputManifest) != 0 {
				if err := workspace.writeAtomic(filepath.Join("context", visualInputManifestArtifact), control.visualInputManifest, 0o444); err != nil {
					return err
				}
			}
			if len(control.screenshotManifest) != 0 {
				if err := workspace.writeAtomic(filepath.Join("context", screenshotManifestArtifact), control.screenshotManifest, 0o444); err != nil {
					return err
				}
			}
			return control.stageVisualInputs(workspace)
		},
		func(workspace *artifactStore) error {
			stagedProse, readErr := workspace.readRegular(filepath.Join("context", proseRelative))
			if readErr != nil || revisionFor(stagedProse) != proseRevision {
				return fmt.Errorf("Visual Editor edited the prose draft input")
			}
			report, readErr := workspace.readNonEmpty("plan.md")
			if readErr != nil {
				return readErr
			}
			planData, readErr := workspace.readNonEmpty("plan.json")
			if readErr != nil {
				return readErr
			}
			validated, parseErr := parseVisualPlan(planData, proseRevision, available)
			if parseErr != nil {
				return parseErr
			}
			if err := validatePlanReport(string(report), validated); err != nil {
				return err
			}
			plan = validated
			planReport = report
			return nil
		},
		func(_ *artifactStore) error {
			assets, err := control.planAssets(candidate, plan)
			if err != nil {
				return err
			}
			if err := control.store.writeAtomic(visualPlanPath(candidate), planReport, 0o644); err != nil {
				return err
			}
			for _, asset := range assets {
				input, staged := control.visualInput(asset.ID)
				if !staged {
					return fmt.Errorf("asset %q is not a controller-staged visual input", asset.ID)
				}
				if err := control.store.writeAtomic(asset.Path, input.Data, 0o444); err != nil {
					return err
				}
			}
			control.visualPlans[candidate] = plan
			control.visualAssets[candidate] = assets
			return nil
		},
		func() error {
			delete(control.visualPlans, candidate)
			delete(control.visualAssets, candidate)
			return control.store.removeAll(visualDirPath(candidate))
		})
}

// runWriterAssembly applies one validated visual plan in a fresh Writer
// invocation. The Writer stays the only owner of article Markdown: it places
// the planned diagrams and images and removes the explanation they replace.
func (control *controller) runWriterAssembly(candidate int) error {
	base, err := control.prompts.load("writer.md")
	if err != nil {
		return err
	}
	plan, planned := control.visualPlans[candidate]
	if !planned {
		return fmt.Errorf("no validated visual plan for candidate %03d", candidate)
	}
	assets := control.visualAssets[candidate]
	planReport, err := control.store.readRegular(visualPlanPath(candidate))
	if err != nil {
		return err
	}
	planData, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("encode validated visual plan: %w", err)
	}
	planData = append(planData, '\n')
	proseRelative := proseDraftPath(candidate)
	proseData, err := control.store.readRegular(proseRelative)
	if err != nil {
		return err
	}
	target := candidateDraftPath(candidate)
	prompt := base + fmt.Sprintf("\n\n## Assignment\n\nAssemble candidate %03d to `%s` in this isolated workspace by applying the"+
		"\nvalidated visual plan to the prose draft."+
		"\n\n- Reproduce every planned Mermaid diagram at its planned location inside a fenced ```mermaid block, byte-for-byte"+
		"\n  as the plan gives it."+
		"\n- Reference every planned image exactly once as `![<alt_text>](<path>)`, using the plan's alt text and the path"+
		"\n  listed below. Add no other image."+
		"\n- Shorten, split, or reorganize the surrounding explanation so the article does not state the same thing twice in"+
		"\n  prose and in a visual, and apply every `restructure_text` entry. Where a plan entry's rationale says the visual"+
		"\n  now carries an explanation, remove or compress that explanation: do not leave a paragraph that walks through the"+
		"\n  same steps the visual draws."+
		"\n- Change nothing else: the substance of the prose draft is already reviewed material.",
		candidate, target)
	if visualPlacements(plan) != 0 {
		// The anti-duplication rule is measured, so the assignment states both
		// the measure and the exact number to stay below.
		prompt += fmt.Sprintf("\n\nThis plan places a visual, so the assembled article must carry strictly less explanation than the prose"+
			"\ndraft. Go counts every non-whitespace character outside fenced blocks and outside `![alt](path)` references."+
			"\nThe prose draft has %d such characters; the assembled article must have fewer. Take the reduction from the"+
			"\nexplanation each visual now carries rather than from unrelated sections.", proseCharacterCount(string(proseData)))
	}
	if len(assets) != 0 {
		prompt += "\n\nPlanned image paths, staged read-only at the same relative path under `context/`:\n"
		for _, asset := range assets {
			prompt += fmt.Sprintf("\n- %s (opportunity %s, alt text %q)", asset.Path, asset.OpportunityID, asset.AltText)
		}
	}
	inputs := []string{"brief.md", "evidence/sources.md", "claim-ledger.md", "outline.md", proseRelative}
	for _, relative := range inputs {
		data, readErr := control.store.readRegular(relative)
		if readErr != nil {
			return readErr
		}
		prompt += contextBlock(relative, data)
	}
	prompt += contextBlock(visualPlanPath(candidate), planReport)
	prompt += contextBlock("visual-plan.json", planData)
	styleGuide, err := loadStyleGuide(control.contentRoot)
	if err != nil {
		return err
	}
	if len(styleGuide) != 0 {
		prompt += contextBlock("style-guide.md", styleGuide)
	}
	previousCandidate := control.workflow.CurrentCandidate
	previousRevision := control.workflow.CurrentRevision
	return control.runWorker("writer", "", candidate, control.workflow.CurrentProseRevision, "assembling", prompt,
		func(workspace *artifactStore) error {
			for _, relative := range inputs {
				data, readErr := control.store.readRegular(relative)
				if readErr != nil {
					return readErr
				}
				if err := workspace.writeAtomic(filepath.Join("context", relative), data, 0o444); err != nil {
					return err
				}
			}
			if err := workspace.writeAtomic(filepath.Join("context", visualPlanPath(candidate)), planReport, 0o444); err != nil {
				return err
			}
			if err := workspace.writeAtomic(filepath.Join("context", "visual-plan.json"), planData, 0o444); err != nil {
				return err
			}
			if len(styleGuide) != 0 {
				if err := workspace.writeAtomic(filepath.Join("context", "style-guide.md"), styleGuide, 0o444); err != nil {
					return err
				}
			}
			return control.stageVisualAssets(workspace, assets)
		},
		func(workspace *artifactStore) error {
			data, readErr := workspace.readNonEmpty(target)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(strings.ToLower(string(data)), "todo") {
				return fmt.Errorf("candidate contains unresolved TODO placeholder")
			}
			return validateAssembledCandidate(data, proseData, plan, assets)
		},
		func(workspace *artifactStore) error {
			data, copyErr := control.store.copyRegularFrom(workspace, target, target, 0o644)
			if copyErr != nil {
				return copyErr
			}
			manifest := buildVisualManifest(candidate, plan, planReport, proseData, data, assets)
			manifestData, encodeErr := json.MarshalIndent(manifest, "", "  ")
			if encodeErr != nil {
				return fmt.Errorf("encode %s: %w", visualManifestPath(candidate), encodeErr)
			}
			if err := control.store.writeAtomic(visualManifestPath(candidate), append(manifestData, '\n'), 0o444); err != nil {
				return err
			}
			control.workflow.CurrentCandidate = candidate
			control.workflow.CurrentRevision = manifest.ReviewedRevision
			return nil
		},
		func() error {
			control.workflow.CurrentCandidate = previousCandidate
			control.workflow.CurrentRevision = previousRevision
			return errors.Join(
				control.store.remove(visualManifestPath(candidate)),
				control.store.remove(target),
			)
		})
}

// stageVisualAssets copies each placed asset into the assembly workspace at
// exactly the relative path the assembled article will reference.
func (control *controller) stageVisualAssets(workspace *artifactStore, assets []VisualAssetRecord) error {
	for _, asset := range assets {
		input, staged := control.visualInput(asset.ID)
		if !staged {
			return fmt.Errorf("asset %q is not a controller-staged visual input", asset.ID)
		}
		if err := workspace.writeAtomic(filepath.Join("context", asset.Path), input.Data, 0o444); err != nil {
			return err
		}
	}
	return nil
}

func (control *controller) reviewCandidate(candidate int) (bool, string, error) {
	for _, lens := range []string{"evidence", "story", "clarity", "copy"} {
		result, err := control.runReviewer(candidate, lens)
		if err != nil {
			return false, "", err
		}
		if result.Status == "blocked" {
			return false, fmt.Sprintf("%s reviewer blocked candidate %03d", lens, candidate), nil
		}
		decisions, err := control.requestPMDecision(candidate, lens, result)
		if err != nil {
			return false, "", err
		}
		mustFix, human := decisionOutcome(decisions)
		if human != nil {
			return false, fmt.Sprintf("human judgment required for finding %s: %s", human.FindingID, human.Reason), nil
		}
		if mustFix {
			return true, "", nil
		}
	}
	return false, "", nil
}

func decisionOutcome(decisions []PMDecision) (bool, *PMDecision) {
	mustFix := false
	for index := range decisions {
		decision := &decisions[index]
		if decision.Decision == "needs_human_judgment" {
			return mustFix, decision
		}
		if decision.Decision == "valid_must_fix" {
			mustFix = true
		}
	}
	return mustFix, nil
}

func (control *controller) runReviewer(candidate int, lens string) (ReviewResult, error) {
	var result ReviewResult
	base, err := control.prompts.load("reviewer-" + lens + ".md")
	if err != nil {
		return result, err
	}
	outputContract, err := control.prompts.load("reviewer-output.md")
	if err != nil {
		return result, err
	}
	base += "\n\n" + outputContract
	// The reviewed revision covers the assembled Markdown and the bytes of
	// every referenced local asset, so a malformed or stale manifest, an
	// unsafe or non-regular asset, and a replaced asset all stop the run
	// before a lens sees the candidate.
	revision, candidateData, err := control.candidateBinding(candidate)
	if err != nil {
		return result, err
	}
	if revision != control.workflow.CurrentRevision {
		return result, fmt.Errorf("candidate revision changed outside the controller")
	}
	prompt := base + fmt.Sprintf("\n\n## Assignment\n\nLens: `%s`\nCandidate: `article-%03d`\nRevision: `%s`", lens, candidate, revision)
	contextFiles, err := control.reviewerContext(candidate, lens, candidateData, revision)
	if err != nil {
		return result, err
	}
	var screenshotImages map[string][]byte
	if lens == "evidence" {
		if _, screenshotImages, err = control.screenshotContext(); err != nil {
			return result, err
		}
	}
	// Only the Evidence lens checks what a placed image actually shows, so
	// only it receives the asset bytes; every lens receives the plan.
	visualAssets := []VisualAssetRecord(nil)
	if lens == "evidence" {
		visualAssets = control.visualAssets[candidate]
	}
	contextNames := make([]string, 0, len(contextFiles))
	for label := range contextFiles {
		contextNames = append(contextNames, label)
	}
	sort.Strings(contextNames)
	for _, label := range contextNames {
		data := contextFiles[label]
		prompt += contextBlock(label, data)
	}
	resultData := []byte(nil)
	reportData := []byte(nil)
	err = control.runWorker("reviewer_"+lens, lens, candidate, revision, "reviewing", prompt,
		func(workspace *artifactStore) error {
			for _, relative := range contextNames {
				data := contextFiles[relative]
				target := filepath.Join("context", reviewerContextName(relative))
				if err := workspace.writeAtomic(target, data, 0o444); err != nil {
					return err
				}
			}
			// Only the Evidence lens inspects the screenshots themselves, so
			// only it receives the image bytes, read-only and unchanged.
			if lens == "evidence" && len(control.screenshotManifest) != 0 {
				if err := stageScreenshotImages(workspace, screenshotImages); err != nil {
					return err
				}
			}
			return control.stageVisualAssets(workspace, visualAssets)
		},
		func(workspace *artifactStore) error {
			// The reviewed revision covers the article and its assets
			// together, so the staged inputs are checked against their own
			// digests rather than against the composite.
			stagedCandidate, readErr := workspace.readRegular("context/article.md")
			if readErr != nil || revisionFor(stagedCandidate) != revisionFor(candidateData) {
				return fmt.Errorf("%s reviewer edited the candidate input", lens)
			}
			for _, asset := range visualAssets {
				stagedAsset, assetErr := workspace.readRegular(filepath.Join("context", asset.Path))
				if assetErr != nil || revisionFor(stagedAsset) != asset.SHA256 {
					return fmt.Errorf("%s reviewer edited the staged visual asset %s", lens, asset.Path)
				}
			}
			resultData, readErr = workspace.readNonEmpty("result.json")
			if readErr != nil {
				return readErr
			}
			reportData, readErr = workspace.readNonEmpty("report.md")
			if readErr != nil {
				return readErr
			}
			validated, validateErr := validateReviewData(resultData, reportData, lens, revision)
			if validateErr == nil {
				result = validated
			}
			return validateErr
		},
		func(_ *artifactStore) error {
			basePath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens)
			if err := control.store.writeAtomic(filepath.Join(basePath, "result.json"), resultData, 0o644); err != nil {
				return err
			}
			return control.store.writeAtomic(filepath.Join(basePath, "report.md"), reportData, 0o644)
		},
		func() error {
			return control.store.removeAll(filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens))
		})
	return result, err
}

func reviewerContextName(label string) string {
	switch {
	case label == "brief.md":
		return "brief.md"
	case strings.HasPrefix(label, "drafts/article-") && strings.HasSuffix(label, "-prose.md"):
		return "prose.md"
	case strings.HasPrefix(label, "drafts/article-"):
		return "article.md"
	case strings.HasPrefix(label, "visuals/article-") && strings.HasSuffix(label, "/plan.md"):
		return "visual-plan.md"
	case strings.HasPrefix(label, "visuals/article-") && strings.HasSuffix(label, "/manifest.json"):
		return "visual-manifest.json"
	default:
		return label
	}
}

func (control *controller) reviewerContext(candidate int, lens string, candidateData []byte, revision string) (map[string][]byte, error) {
	files := map[string][]byte{
		"brief.md":                    []byte(control.brief.Raw),
		candidateDraftPath(candidate): candidateData,
		"revision.txt":                []byte(revision + "\n"),
	}
	read := func(relative string) error {
		data, err := control.store.readRegular(relative)
		if err != nil {
			return err
		}
		files[relative] = data
		return nil
	}
	// Every lens reviews the same visual decisions, so every lens receives the
	// inspectable plan and the manifest that binds it to this exact revision.
	for _, relative := range []string{visualPlanPath(candidate), visualManifestPath(candidate)} {
		if err := read(relative); err != nil {
			return nil, err
		}
	}
	switch lens {
	case "evidence":
		for _, relative := range []string{"evidence/sources.md", "claim-ledger.md"} {
			if err := read(relative); err != nil {
				return nil, err
			}
		}
		if len(control.screenshotManifest) != 0 {
			files[screenshotManifestArtifact] = control.screenshotManifest
		}
		if data, err := control.store.readRegular("evidence/firsthand.md"); err == nil {
			files["evidence/firsthand.md"] = data
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	case "story":
		if err := read("outline.md"); err != nil {
			return nil, err
		}
	case "clarity":
		files["clarity-fields.md"] = []byte(fmt.Sprintf("Audience:\n%s\n\nConstraints:\n%s\n", control.brief.Sections["Audience"], control.brief.Sections["Constraints"]))
		// The Clarity lens judges whether a visual replaced explanation or
		// merely duplicated it, so it also receives the source prose draft.
		if err := read(proseDraftPath(candidate)); err != nil {
			return nil, err
		}
	case "copy":
		styleGuide, styleErr := loadStyleGuide(control.contentRoot)
		if styleErr != nil {
			return nil, styleErr
		}
		if len(styleGuide) != 0 {
			files["style-guide.md"] = styleGuide
		}
	}
	return files, nil
}

func validateReviewData(resultData, reportData []byte, lens, revision string) (ReviewResult, error) {
	var result ReviewResult
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(resultData, &fields); err != nil {
		return result, fmt.Errorf("invalid reviewer JSON: %w", err)
	}
	for _, required := range []string{"status", "lens", "reviewed_revision", "findings"} {
		if _, found := fields[required]; !found {
			return result, fmt.Errorf("review result is missing required field %q", required)
		}
	}
	if strings.TrimSpace(string(fields["findings"])) == "null" {
		return result, fmt.Errorf("review findings must be an array")
	}
	if err := decodeStrictJSON(resultData, &result); err != nil {
		return result, err
	}
	if result.Lens != lens {
		return result, fmt.Errorf("review lens mismatch: got %q, want %q", result.Lens, lens)
	}
	if result.ReviewedRevision != revision {
		return result, fmt.Errorf("stale review revision: got %q, want %q", result.ReviewedRevision, revision)
	}
	if result.Status != "clean" && result.Status != "fix_required" && result.Status != "blocked" {
		return result, fmt.Errorf("invalid reviewer status %q", result.Status)
	}
	if result.Status == "clean" && len(result.Findings) != 0 {
		return result, fmt.Errorf("clean review contains findings")
	}
	if result.Status == "fix_required" && len(result.Findings) == 0 {
		return result, fmt.Errorf("fix_required review has no findings")
	}
	seen := make(map[string]bool)
	for _, finding := range result.Findings {
		if strings.TrimSpace(finding.ID) == "" || strings.TrimSpace(finding.Severity) == "" || strings.TrimSpace(finding.Location) == "" || strings.TrimSpace(finding.Problem) == "" || strings.TrimSpace(finding.SuggestedDirection) == "" {
			return result, fmt.Errorf("review finding fields must be non-empty")
		}
		if seen[finding.ID] {
			return result, fmt.Errorf("duplicate finding ID %q", finding.ID)
		}
		seen[finding.ID] = true
	}
	reportFindings, err := parseReportFindings(string(reportData))
	if err != nil {
		return result, err
	}
	if len(reportFindings) != len(result.Findings) {
		return result, fmt.Errorf("human report finding count does not match result.json")
	}
	for index, finding := range result.Findings {
		if reportFindings[index] != finding {
			return result, fmt.Errorf("human report entry does not match finding %s", finding.ID)
		}
	}
	return result, nil
}

func parseReportFindings(report string) ([]Finding, error) {
	lines := strings.Split(strings.ReplaceAll(report, "\r\n", "\n"), "\n")
	labels := []string{"id", "severity", "location", "problem", "suggested direction"}
	var findings []Finding
	for index := 0; index < len(lines); index++ {
		label, _, isField := reportField(lines[index])
		if !isField || label != labels[0] {
			if isField {
				for _, expected := range labels[1:] {
					if label == expected {
						return nil, fmt.Errorf("human report contains an incomplete finding entry")
					}
				}
			}
			continue
		}
		if index+len(labels) > len(lines) {
			return nil, fmt.Errorf("human report contains a truncated finding entry")
		}
		values := make([]string, len(labels))
		cursor := index
		for offset, expected := range labels {
			for cursor < len(lines) && strings.TrimSpace(lines[cursor]) == "" {
				cursor++
			}
			if cursor >= len(lines) {
				return nil, fmt.Errorf("human report contains a truncated finding entry")
			}
			actual, value, found := reportField(lines[cursor])
			if !found || actual != expected {
				return nil, fmt.Errorf("human report finding entry is missing %s", expected)
			}
			values[offset] = value
			cursor++
		}
		findings = append(findings, Finding{
			ID: values[0], Severity: values[1], Location: values[2], Problem: values[3], SuggestedDirection: values[4],
		})
		index = cursor - 1
	}
	return findings, nil
}

func reportField(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "- ") {
		line = strings.TrimPrefix(line, "- ")
	}
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	label := strings.ToLower(strings.TrimSpace(parts[0]))
	label = strings.ReplaceAll(label, "_", " ")
	switch label {
	case "id", "severity", "location", "problem", "suggested direction":
		return label, strings.TrimSpace(parts[1]), true
	default:
		return "", "", false
	}
}

func (control *controller) requestPMDecision(candidate int, lens string, result ReviewResult) ([]PMDecision, error) {
	pmTimeout := control.config.AgentTimeout
	if injected, err := time.ParseDuration(os.Getenv("WRITE_UUTER_TEST_PM_DECISION_TIMEOUT")); err == nil && injected > 0 {
		pmTimeout = injected
	}
	deadlineUnixNano := invocationDeadline(pmTimeout)
	resultPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "result.json")
	reportPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "report.md")
	resultData, err := control.store.readRegular(resultPath)
	if err != nil {
		return nil, err
	}
	reportData, err := control.store.readRegular(reportPath)
	if err != nil {
		return nil, err
	}
	requestID := randomToken()
	requestPath := filepath.Join("inbox", requestID+".json")
	contextDirectory := filepath.Join("requests", requestID)
	outputPath := filepath.Join("outbox", requestID+".md")
	request := pmRequest{
		RequestID: requestID, Candidate: candidate, Lens: lens,
		ReviewedRevision: control.workflow.CurrentRevision, ReviewDigest: reviewDigest(resultData, reportData),
		ResultPath: filepath.Join(contextDirectory, "result.json"), ReportPath: filepath.Join(contextDirectory, "report.md"),
		DecisionPath:     fmt.Sprintf("pm-decisions/article-%03d.md", candidate),
		RequestPath:      requestPath,
		ContextDirectory: contextDirectory, OutputPath: outputPath,
	}
	pmStore, err := control.runtime.workspaceStore(control.pm)
	if err != nil {
		return nil, err
	}
	defer pmStore.Close()
	if err := control.stagePMContext(pmStore, request, resultData, reportData); err != nil {
		return nil, err
	}
	if err := pmStore.writeJSON(requestPath, request); err != nil {
		return nil, err
	}
	testInvocationDelay("WRITE_UUTER_TEST_AFTER_PM_REQUEST_DELAY")
	defer pmStore.remove(requestPath)
	defer pmStore.remove(outputPath)
	defer pmStore.removeAll(contextDirectory)
	control.workflow.ActiveRole = "pm"
	if err := control.saveWorkflow(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, deadlineUnixNano))
	defer cancel()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if time.Now().UnixNano() >= deadlineUnixNano {
			return nil, fmt.Errorf("PM timed out deciding %s review: wall-clock deadline exceeded", lens)
		}
		data, readErr := pmStore.readNonEmpty(outputPath)
		if readErr == nil {
			decisions, validateErr := control.validatePMDecisionData(data, candidate, lens, result, request)
			if validateErr == nil {
				if err := ensureInvocationDeadline(deadlineUnixNano, "PM decision acceptance"); err != nil {
					return nil, err
				}
				live, liveErr := control.runtime.invocationLive(control.pm)
				if liveErr != nil {
					return nil, fmt.Errorf("verify persistent PM at decision acceptance: %w", liveErr)
				}
				if !live {
					return nil, fmt.Errorf("PM exited before %s decision acceptance", lens)
				}
				decisionPath := fmt.Sprintf("pm-decisions/article-%03d.md", candidate)
				previousDecision, previousErr := control.store.readRegular(decisionPath)
				if previousErr != nil && !errors.Is(previousErr, os.ErrNotExist) {
					return nil, previousErr
				}
				rollbackDecision := func() error {
					if previousErr == nil {
						return control.store.writeAtomic(decisionPath, previousDecision, 0o644)
					}
					return control.store.remove(decisionPath)
				}
				if err := control.store.writeAtomic(decisionPath, data, 0o644); err != nil {
					return nil, err
				}
				if err := waitForTestBarrier("WRITE_UUTER_TEST_PM_EXIT_DECISION_COMMIT_BARRIER"); err != nil {
					return nil, errors.Join(err, wrappedOptionalError("PM decision rollback", rollbackDecision()))
				}
				if err := ensureInvocationDeadline(deadlineUnixNano, "PM decision commit"); err != nil {
					return nil, errors.Join(err, wrappedOptionalError("PM decision rollback", rollbackDecision()))
				}
				live, liveErr = control.runtime.invocationLive(control.pm)
				if liveErr != nil {
					return nil, errors.Join(fmt.Errorf("verify persistent PM after decision commit: %w", liveErr), wrappedOptionalError("PM decision rollback", rollbackDecision()))
				}
				if !live {
					return nil, errors.Join(fmt.Errorf("PM exited during %s decision commit", lens), wrappedOptionalError("PM decision rollback", rollbackDecision()))
				}
				if control.decisionBindings[candidate] == nil {
					control.decisionBindings[candidate] = make(map[string]decisionBinding)
				}
				previousBinding, hadPreviousBinding := control.decisionBindings[candidate][lens]
				previousReachedLength := len(control.reachedLenses[candidate])
				rollbackAcceptance := func(commitErr error) error {
					if hadPreviousBinding {
						control.decisionBindings[candidate][lens] = previousBinding
					} else {
						delete(control.decisionBindings[candidate], lens)
					}
					control.reachedLenses[candidate] = control.reachedLenses[candidate][:previousReachedLength]
					return errors.Join(commitErr, wrappedOptionalError("PM decision rollback", rollbackDecision()))
				}
				control.decisionBindings[candidate][lens] = decisionBinding{RequestID: requestID, Digest: request.ReviewDigest, DecisionDigest: decisionListDigest(decisions)}
				control.reachedLenses[candidate] = append(control.reachedLenses[candidate], lens)
				control.workflow.ActiveRole = ""
				if err := control.saveWorkflow(); err != nil {
					return nil, rollbackAcceptance(err)
				}
				live, liveErr = control.runtime.invocationLive(control.pm)
				if liveErr != nil {
					return nil, rollbackAcceptance(fmt.Errorf("verify persistent PM across decision workflow commit: %w", liveErr))
				}
				if !live {
					return nil, rollbackAcceptance(fmt.Errorf("PM exited across %s decision workflow commit", lens))
				}
				return decisions, nil
			}
			if !errors.Is(validateErr, errNotReady) {
				return nil, validateErr
			}
		} else if !errors.Is(readErr, errNotReady) {
			return nil, readErr
		}
		if status, exited, statusErr := control.runtime.exitStatus(control.pm); statusErr != nil {
			return nil, fmt.Errorf("read PM completion: %w", statusErr)
		} else if exited {
			return nil, fmt.Errorf("PM exited unexpectedly with status %d before deciding %s review", status, lens)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("PM timed out deciding %s review: %w", lens, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (control *controller) stagePMContext(pmStore *artifactStore, request pmRequest, resultData, reportData []byte) error {
	files := map[string][]byte{
		"result.json": resultData,
		"report.md":   reportData,
		"brief.md":    []byte(control.brief.Raw),
	}
	for _, relative := range []string{"evidence/sources.md", "claim-ledger.md", "outline.md",
		candidateDraftPath(request.Candidate), visualPlanPath(request.Candidate)} {
		data, err := control.store.readRegular(relative)
		if err != nil {
			return err
		}
		files[relative] = data
	}
	if previous, err := control.store.readRegular(fmt.Sprintf("pm-decisions/article-%03d.md", request.Candidate)); err == nil {
		files["previous-decision.md"] = previous
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for relative, data := range files {
		if err := pmStore.writeAtomic(filepath.Join(request.ContextDirectory, relative), data, 0o444); err != nil {
			return err
		}
	}
	return nil
}

func (control *controller) validatePMDecisionData(data []byte, candidate int, lens string, result ReviewResult, request pmRequest) ([]PMDecision, error) {
	document, err := parsePMDecisionDocument(data)
	if err != nil {
		return nil, err
	}
	if document.ReviewedRevision != control.workflow.CurrentRevision {
		return nil, fmt.Errorf("PM decision revision mismatch: got %q, want %q", document.ReviewedRevision, control.workflow.CurrentRevision)
	}
	expected := append(append([]string{}, control.reachedLenses[candidate]...), lens)
	if len(document.Lenses) != len(expected) {
		return nil, fmt.Errorf("PM decision lens history mismatch: got %d entries, want %d", len(document.Lenses), len(expected))
	}
	for _, expectedLens := range expected {
		record, found := document.Lenses[expectedLens]
		if !found {
			return nil, fmt.Errorf("PM decision missing reached lens %q", expectedLens)
		}
		resultPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), expectedLens, "result.json")
		reportPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), expectedLens, "report.md")
		resultData, err := control.store.readRegular(resultPath)
		if err != nil {
			return nil, err
		}
		reportData, err := control.store.readRegular(reportPath)
		if err != nil {
			return nil, err
		}
		validatedResult, err := validateReviewData(resultData, reportData, expectedLens, control.workflow.CurrentRevision)
		if err != nil {
			return nil, err
		}
		binding := control.decisionBindings[candidate][expectedLens]
		if expectedLens == lens {
			binding = decisionBinding{RequestID: request.RequestID, Digest: request.ReviewDigest, DecisionDigest: decisionListDigest(record.Decisions)}
		}
		if record.RequestID != binding.RequestID || record.ReviewDigest != binding.Digest {
			return nil, fmt.Errorf("PM decision for %s is not bound to the active review request", expectedLens)
		}
		if err := validateDecisionList(record.Decisions, validatedResult); err != nil {
			return nil, err
		}
		if decisionListDigest(record.Decisions) != binding.DecisionDigest {
			return nil, fmt.Errorf("PM decision history changed accepted classifications for %s", expectedLens)
		}
	}
	return document.Lenses[lens].Decisions, nil
}

func parsePMDecisionDocument(data []byte) (PMDecisionDocument, error) {
	var document PMDecisionDocument
	trimmed := strings.TrimSpace(string(data))
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 || lines[0] != "```json" || lines[len(lines)-1] != "```" {
		return document, fmt.Errorf("PM decision must contain exactly one complete fenced JSON document")
	}
	bodyLines := lines[1 : len(lines)-1]
	for _, line := range bodyLines {
		if strings.Contains(line, "```") {
			return document, fmt.Errorf("PM decision must contain exactly one complete fenced JSON document")
		}
	}
	if strings.TrimSpace(strings.Join(bodyLines, "\n")) == "" {
		return document, fmt.Errorf("PM decision must contain exactly one complete fenced JSON document")
	}
	payload := strings.TrimSpace(strings.Join(bodyLines, "\n"))
	if !strings.HasPrefix(payload, "{") || !strings.HasSuffix(payload, "}") {
		return document, fmt.Errorf("PM decision must contain exactly one complete fenced JSON document")
	}
	var raw any
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return document, fmt.Errorf("invalid PM decision JSON: %w", err)
	}
	if err := rejectNullControlFields(raw, "$"); err != nil {
		return document, err
	}
	var shape struct {
		ReviewedRevision string `json:"reviewed_revision"`
		Lenses           map[string]struct {
			RequestID    string          `json:"request_id"`
			ReviewDigest string          `json:"review_digest"`
			Decisions    json.RawMessage `json:"decisions"`
		} `json:"lenses"`
	}
	if err := decodeStrictJSON([]byte(payload), &shape); err != nil {
		return document, fmt.Errorf("PM decision must contain exactly one complete fenced JSON document: %w", err)
	}
	for lens, record := range shape.Lenses {
		decisions := bytes.TrimSpace(record.Decisions)
		if len(decisions) == 0 || bytes.Equal(decisions, []byte("null")) || decisions[0] != '[' {
			return document, fmt.Errorf("PM decision for %s must contain a non-null decisions array", lens)
		}
	}
	if err := decodeStrictJSON([]byte(payload), &document); err != nil {
		return document, fmt.Errorf("PM decision must contain exactly one complete fenced JSON document: %w", err)
	}
	return document, nil
}

func rejectNullControlFields(value any, path string) error {
	switch current := value.(type) {
	case nil:
		if strings.HasSuffix(path, ".decisions") {
			return fmt.Errorf("PM decision must contain a non-null decisions array")
		}
		return fmt.Errorf("PM decision control field %s must not be null", path)
	case map[string]any:
		for key, child := range current {
			if err := rejectNullControlFields(child, path+"."+key); err != nil {
				return err
			}
		}
	case []any:
		for index, child := range current {
			if err := rejectNullControlFields(child, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func decisionListDigest(decisions []PMDecision) string {
	data, err := json.Marshal(decisions)
	if err != nil {
		panic(fmt.Sprintf("marshal PM decisions: %v", err))
	}
	return revisionFor(data)
}

func validateDecisionList(decisions []PMDecision, result ReviewResult) error {
	findings := make(map[string]Finding, len(result.Findings))
	for _, finding := range result.Findings {
		findings[finding.ID] = finding
	}
	seen := make(map[string]bool)
	for _, decision := range decisions {
		if _, exists := findings[decision.FindingID]; !exists {
			return fmt.Errorf("PM decision references unknown finding %q", decision.FindingID)
		}
		if seen[decision.FindingID] {
			return fmt.Errorf("PM decision duplicates finding %q", decision.FindingID)
		}
		seen[decision.FindingID] = true
		switch decision.Decision {
		case "valid_must_fix", "valid_optional", "needs_human_judgment":
		case "invalid":
			if strings.TrimSpace(decision.Reason) == "" {
				return fmt.Errorf("invalid decision for %s requires a reason", decision.FindingID)
			}
		default:
			return fmt.Errorf("invalid PM classification %q", decision.Decision)
		}
	}
	if len(seen) != len(findings) {
		var missing []string
		for id := range findings {
			if !seen[id] {
				missing = append(missing, id)
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("PM decision missing findings: %s", strings.Join(missing, ", "))
	}
	return nil
}

// workerLaunchDeadline bounds one worker invocation. A test may bound the
// launch transition independently of the PM setup budget, so a slow host
// cannot retarget the boundary the test intends to exercise.
func (control *controller) workerLaunchDeadline() int64 {
	if budget, err := time.ParseDuration(os.Getenv("WRITE_UUTER_TEST_WORKER_LAUNCH_TIMEOUT")); err == nil && budget > 0 {
		return invocationDeadline(budget)
	}
	return invocationDeadline(control.config.AgentTimeout)
}

func (control *controller) runWorker(role, lens string, candidate int, revision, phase, prompt string, prepare, validate, commit func(*artifactStore) error, rollback func() error) error {
	deadlineUnixNano := control.workerLaunchDeadline()
	profile, err := control.policy.profileFor(role)
	if err != nil {
		return err
	}
	testInvocationDelay("WRITE_UUTER_TEST_BEFORE_WORKER_START_DELAY")
	inv, err := control.runtime.prepareInvocation(role, lens, candidate, revision, prompt, profile)
	if err != nil {
		return err
	}
	if err := control.publishInvocationAudit(inv); err != nil {
		return err
	}
	workspace, err := control.runtime.workspaceStore(inv)
	if err != nil {
		return err
	}
	defer workspace.Close()
	if err := prepare(workspace); err != nil {
		return err
	}
	control.workflow.Phase = phase
	control.workflow.ActiveRole = role
	if err := control.saveWorkflow(); err != nil {
		return err
	}
	if err := ensureInvocationDeadline(deadlineUnixNano, role+" launch"); err != nil {
		return err
	}
	if os.Getenv("WRITE_UUTER_TEST_FAIL_BEFORE_LAUNCH") == role {
		return fmt.Errorf("injected %s pre-launch failure", role)
	}
	launchErr := control.runtime.startWorker(control.pm, inv, deadlineUnixNano)
	// Count a reviewer only once tmux has been asked to launch it. A launch
	// request that timed out ambiguously may still have started a process, so
	// it is counted; every failure before the request is not.
	if strings.HasPrefix(role, "reviewer_") && control.runtime.launchAttempted(inv.ID) {
		control.workflow.ReviewAttemptCount++
		if err := control.saveWorkflow(); err != nil {
			return errors.Join(launchErr, err)
		}
	}
	if launchErr != nil {
		return launchErr
	}
	if readyRelative := os.Getenv("WRITE_UUTER_TEST_WORKER_READY_FILE"); readyRelative != "" {
		if err := waitForTestWorkerArtifact(workspace, readyRelative, deadlineUnixNano); err != nil {
			return err
		}
	}
	if activeTimeout, parseErr := time.ParseDuration(os.Getenv("WRITE_UUTER_TEST_WORKER_ACTIVE_TIMEOUT")); parseErr == nil && activeTimeout > 0 {
		deadlineUnixNano = invocationDeadline(activeTimeout)
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, deadlineUnixNano))
	defer cancel()
	if err := control.runtime.waitForWorker(ctx, deadlineUnixNano, control.pm, inv); err != nil {
		return err
	}
	if err := ensureInvocationDeadline(deadlineUnixNano, role+" artifact validation"); err != nil {
		return err
	}
	pmLive, err := control.runtime.invocationLive(control.pm)
	if err != nil {
		return fmt.Errorf("verify PM before %s artifact commit: %w", role, err)
	}
	if !pmLive {
		return fmt.Errorf("PM exited before %s artifact commit", role)
	}
	if err := validate(workspace); err != nil {
		return fmt.Errorf("%s artifact contract failed after process completion: %w", role, err)
	}
	if err := commit(workspace); err != nil {
		return errors.Join(err, wrappedOptionalError(role+" artifact rollback", rollback()))
	}
	rollbackFailure := func(commitErr error) error {
		return errors.Join(commitErr, wrappedOptionalError(role+" artifact rollback", rollback()))
	}
	if err := waitForTestBarrier("WRITE_UUTER_TEST_PM_EXIT_WORKER_COMMIT_BARRIER"); err != nil {
		return rollbackFailure(err)
	}
	if err := ensureInvocationDeadline(deadlineUnixNano, role+" artifact commit"); err != nil {
		return rollbackFailure(err)
	}
	pmLive, err = control.runtime.invocationLive(control.pm)
	if err != nil {
		return rollbackFailure(fmt.Errorf("verify PM after %s artifact commit: %w", role, err))
	}
	if !pmLive {
		return rollbackFailure(fmt.Errorf("PM exited during %s artifact commit", role))
	}
	control.workflow.ActiveRole = ""
	if err := control.saveWorkflow(); err != nil {
		return rollbackFailure(err)
	}
	pmLive, err = control.runtime.invocationLive(control.pm)
	if err != nil {
		return rollbackFailure(fmt.Errorf("verify PM across %s workflow commit: %w", role, err))
	}
	if !pmLive {
		return rollbackFailure(fmt.Errorf("PM exited across %s workflow commit", role))
	}
	return nil
}

func waitForTestWorkerArtifact(workspace *artifactStore, relative string, deadlineUnixNano int64) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for time.Now().UnixNano() < deadlineUnixNano {
		if _, err := workspace.readRegular(relative); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("wait for test worker artifact %s: %w", relative, err)
		}
		<-ticker.C
	}
	return fmt.Errorf("timed out waiting for test worker artifact %s", relative)
}

func invocationDeadline(timeout time.Duration) int64 {
	return time.Now().Add(timeout).UnixNano()
}

func ensureInvocationDeadline(deadlineUnixNano int64, action string) error {
	if time.Now().UnixNano() >= deadlineUnixNano {
		return fmt.Errorf("%s exceeded its wall-clock deadline", action)
	}
	return nil
}

func testInvocationDelay(name string) {
	if delay, err := time.ParseDuration(os.Getenv(name)); err == nil && delay > 0 {
		time.Sleep(delay)
	}
}

func waitForTestBarrier(name string) error {
	barrier := os.Getenv(name)
	if barrier == "" {
		return nil
	}
	if err := os.WriteFile(barrier+".ready", []byte("ready\n"), 0o600); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(barrier + ".continue"); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("test barrier %s timed out", name)
}

func (control *controller) succeed(candidate int) error {
	if err := control.runtime.cleanup(true, control.pm); err != nil {
		return control.block(fmt.Sprintf("terminal cleanup failed: %v", err))
	}
	if err := control.runtime.archiveAll(control.store); err != nil {
		return control.block(fmt.Sprintf("archive controller audit files: %v", err))
	}
	candidateRevision, _, err := control.candidateBinding(candidate)
	if err != nil {
		return control.block(fmt.Sprintf("read accepted candidate: %v", err))
	}
	if candidateRevision != control.workflow.CurrentRevision {
		return control.block(fmt.Sprintf("accepted candidate changed before publication: got %s, want %s", candidateRevision, control.workflow.CurrentRevision))
	}
	if err := control.validateFinalAudit(candidate); err != nil {
		return control.block(fmt.Sprintf("final review audit invalid: %v", err))
	}
	if err := control.runtime.closePrivate(); err != nil {
		return control.block(fmt.Sprintf("private runtime cleanup failed: %v", err))
	}
	control.runtime = nil
	if err := waitForTestBarrier("WRITE_UUTER_TEST_FINAL_BOUNDARY_BARRIER"); err != nil {
		return control.block(err.Error())
	}
	if err := control.validateFinalAudit(candidate); err != nil {
		return control.block(fmt.Sprintf("final review audit changed at publication boundary: %v", err))
	}
	finalRevision, finalCandidateData, err := control.candidateBinding(candidate)
	if err != nil {
		return control.block(fmt.Sprintf("re-read accepted candidate at publication boundary: %v", err))
	}
	if finalRevision != control.workflow.CurrentRevision {
		return control.block(fmt.Sprintf("accepted candidate changed at publication boundary: got %s, want %s", finalRevision, control.workflow.CurrentRevision))
	}
	published, err := control.store.writeAtomicNoReplace("article.md", finalCandidateData, 0o644)
	// Record the identity before handling the error. A failure after the
	// rename still created the article, and only a recorded identity can be
	// rolled back out of a blocked run.
	control.publishedArticle = published
	if err != nil {
		return control.block(fmt.Sprintf("stage final article: %v", err))
	}
	committedRevision, committedCandidate, candidateErr := control.candidateBinding(candidate)
	committedArticle, articleErr := control.store.readRegular("article.md")
	if candidateErr != nil || articleErr != nil || committedRevision != control.workflow.CurrentRevision || !bytes.Equal(committedCandidate, committedArticle) {
		_ = control.rollbackPublishedArticle()
		return control.block(fmt.Sprintf("accepted candidate/article changed at final commit boundary: candidate=%v article=%v", candidateErr, articleErr))
	}
	now := time.Now().UTC()
	control.workflow.Status = "succeeded"
	control.workflow.Phase = "complete"
	control.workflow.ActiveRole = ""
	control.workflow.CompletedAt = &now
	control.workflow.BlockReason = ""
	if err := control.saveWorkflow(); err != nil {
		_ = control.rollbackPublishedArticle()
		return control.block(fmt.Sprintf("persist succeeded workflow: %v", err))
	}
	committedRevision, committedCandidate, candidateErr = control.candidateBinding(candidate)
	committedArticle, articleErr = control.store.readRegular("article.md")
	if candidateErr != nil || articleErr != nil || committedRevision != control.workflow.CurrentRevision || !bytes.Equal(committedCandidate, committedArticle) {
		_ = control.rollbackPublishedArticle()
		return control.block(fmt.Sprintf("accepted candidate/article changed across succeeded-state commit: candidate=%v article=%v", candidateErr, articleErr))
	}
	if err := control.validateFinalAudit(candidate); err != nil {
		_ = control.rollbackPublishedArticle()
		return control.block(fmt.Sprintf("final review audit changed across succeeded-state commit: %v", err))
	}
	return nil
}

func (control *controller) validateFinalAudit(candidate int) error {
	if len(control.reachedLenses[candidate]) != 4 {
		return fmt.Errorf("final candidate has %d reached lenses, want 4", len(control.reachedLenses[candidate]))
	}
	decisionData, err := control.store.readRegular(fmt.Sprintf("pm-decisions/article-%03d.md", candidate))
	if err != nil {
		return err
	}
	document, err := parsePMDecisionDocument(decisionData)
	if err != nil {
		return err
	}
	if document.ReviewedRevision != control.workflow.CurrentRevision || len(document.Lenses) != 4 {
		return fmt.Errorf("PM decision final revision/history mismatch")
	}
	for _, lens := range []string{"evidence", "story", "clarity", "copy"} {
		resultPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "result.json")
		reportPath := filepath.Join("reviews", fmt.Sprintf("article-%03d", candidate), lens, "report.md")
		resultData, err := control.store.readRegular(resultPath)
		if err != nil {
			return err
		}
		reportData, err := control.store.readRegular(reportPath)
		if err != nil {
			return err
		}
		result, err := validateReviewData(resultData, reportData, lens, control.workflow.CurrentRevision)
		if err != nil {
			return err
		}
		record, found := document.Lenses[lens]
		binding := control.decisionBindings[candidate][lens]
		if !found || record.RequestID != binding.RequestID || record.ReviewDigest != reviewDigest(resultData, reportData) || record.ReviewDigest != binding.Digest || decisionListDigest(record.Decisions) != binding.DecisionDigest {
			return fmt.Errorf("PM decision binding mismatch for %s", lens)
		}
		if err := validateDecisionList(record.Decisions, result); err != nil {
			return err
		}
		mustFix, human := decisionOutcome(record.Decisions)
		if human != nil {
			return fmt.Errorf("final PM audit requires human judgment for %s", human.FindingID)
		}
		if mustFix {
			return fmt.Errorf("final PM audit contains must-fix routing for %s", lens)
		}
	}
	return nil
}

func (control *controller) buildPMPrompt() (string, error) {
	base, err := control.prompts.load("pm.md")
	if err != nil {
		return "", err
	}
	runtimeProtocol, err := control.prompts.load("pm-runtime.md")
	if err != nil {
		return "", err
	}
	return base + "\n\n" + runtimeProtocol, nil
}

func (control *controller) saveWorkflow() error {
	if control.workflow.Status == "succeeded" && os.Getenv("WRITE_UUTER_TEST_FAIL_SUCCESS_SAVE") == "1" {
		return fmt.Errorf("injected succeeded workflow persistence failure")
	}
	if control.workflow.Status == "blocked" && os.Getenv("WRITE_UUTER_TEST_FAIL_BLOCK_SAVE") == "1" {
		return fmt.Errorf("injected blocked workflow persistence failure")
	}
	control.workflow.UpdatedAt = time.Now().UTC()
	return control.store.writeJSON("workflow.json", control.workflow)
}

func (control *controller) block(reason string) error {
	var cleanupErr error
	var archiveErr error
	if control.runtime != nil {
		if cleanupErr = control.runtime.cleanup(false, control.pm); cleanupErr != nil {
			reason += fmt.Sprintf("; cleanup failed: %v", cleanupErr)
		}
		if archiveErr = control.runtime.archiveAll(control.store); archiveErr != nil {
			reason += fmt.Sprintf("; audit archive failed: %v", archiveErr)
		}
	}
	// Roll back only the article identity this controller published. A
	// competing article.md that this run never committed is left untouched.
	if removeErr := control.rollbackPublishedArticle(); removeErr != nil {
		reason += fmt.Sprintf("; failed to remove success-only article: %v", removeErr)
	}
	now := time.Now().UTC()
	control.workflow.Status = "blocked"
	control.workflow.Phase = "blocked"
	control.workflow.ActiveRole = ""
	control.workflow.BlockReason = reason
	control.workflow.CompletedAt = &now
	persistErr := control.saveWorkflow()
	var privateErr error
	if control.runtime != nil {
		if cleanupErr == nil && archiveErr == nil {
			privateErr = control.runtime.closePrivate()
		} else {
			// Preserve controller-owned process identities for a later
			// verified cleanup attempt. closeCredentials keeps the staged
			// credentials until every owned identity has exited, then removes
			// and verifies them; the non-secret ownership and control state
			// stays behind either way so the failure can be diagnosed.
			privateErr = control.runtime.closeCredentials()
		}
	}
	if persistErr != nil || privateErr != nil {
		return errors.Join(errors.New(reason),
			wrappedOptionalError("persist blocked workflow", persistErr),
			wrappedOptionalError("private runtime cleanup", privateErr))
	}
	return errors.New(reason)
}

// rollbackPublishedArticle removes article.md while it is still the identity
// this controller committed, and forgets that identity once it is gone.
func (control *controller) rollbackPublishedArticle() error {
	if control.publishedArticle == nil {
		return nil
	}
	err := control.store.removeOwned("article.md", control.publishedArticle)
	if err == nil || errors.Is(err, errCompetingArtifact) {
		// Either the article this run created is gone, or the name belongs to
		// someone else and this run has nothing left at it to remove.
		control.publishedArticle = nil
	}
	return err
}

func wrappedOptionalError(label string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", label, err)
}

func findStyleGuide(contentRoot string) (string, error) {
	root, err := os.OpenRoot(contentRoot)
	if err != nil {
		return "", err
	}
	defer root.Close()
	for _, relative := range []string{"STYLE.md", "style-guide.md", "docs/style-guide.md"} {
		if parentErr := validateRootParents(root, relative); parentErr != nil && !errors.Is(parentErr, os.ErrNotExist) {
			return "", parentErr
		}
		info, statErr := root.Lstat(relative)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil {
			return "", statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("style guide is a symlink: %s", relative)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("style guide is not a regular file: %s", relative)
		}
		return relative, nil
	}
	return "", nil
}

func loadStyleGuide(contentRoot string) ([]byte, error) {
	stylePath, err := findStyleGuide(contentRoot)
	if err != nil || stylePath == "" {
		return nil, err
	}
	style, err := loadPrompt(contentRoot, stylePath)
	if err != nil {
		return nil, err
	}
	return []byte(style), nil
}

// resolveClaudeExecutable resolves and canonicalizes the Claude client before
// the run directory exists, so a missing executable is a pre-run failure.
func resolveClaudeExecutable(configured string) (string, error) {
	path, err := exec.LookPath(configured)
	if err != nil {
		return "", fmt.Errorf("locate Claude Code executable %q: %w", configured, err)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve Claude Code executable %q: %w", configured, err)
	}
	return canonical, nil
}

// publishInvocationAudit atomically publishes one immutable invocation record
// before the process can be considered ready. The record is written from the
// invocation's own bound profile, which is the profile that built its argument
// vector, so the artifact and the launched command cannot disagree.
func (control *controller) publishInvocationAudit(inv invocation) error {
	role := inv.Role
	if inv.Lens != "" {
		role = strings.TrimSuffix(role, "_"+inv.Lens)
	}
	record := InvocationAudit{
		Invocation: inv.ID, Role: role, Lens: inv.Lens, Candidate: inv.Candidate,
		Provider: inv.Profile.Provider, Model: inv.Profile.Model,
		ReasoningEffort: inv.Profile.ReasoningEffort, ModelPolicyDigest: control.policy.digest,
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode invocation audit for %s: %w", inv.ID, err)
	}
	data = append(data, '\n')
	if _, err := control.store.writeAtomicNoReplace(filepath.Join(".control", "invocations", inv.ID+".json"), data, 0o444); err != nil {
		return fmt.Errorf("publish invocation audit for %s: %w", inv.ID, err)
	}
	return nil
}
