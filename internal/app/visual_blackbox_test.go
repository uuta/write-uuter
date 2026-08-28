package app_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
)

type visualOpportunityRecord struct {
	ID        string `json:"id"`
	Location  string `json:"location"`
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
	Mermaid   string `json:"mermaid"`
	AssetID   string `json:"asset_id"`
	AltText   string `json:"alt_text"`
}

type visualFileRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type visualAssetRecord struct {
	ID            string `json:"id"`
	OpportunityID string `json:"opportunity_id"`
	Path          string `json:"path"`
	Origin        string `json:"origin"`
	Source        string `json:"source"`
	MediaType     string `json:"media_type"`
	ByteSize      int    `json:"byte_size"`
	SHA256        string `json:"sha256"`
	AltText       string `json:"alt_text"`
}

type visualManifestRecord struct {
	SchemaVersion         int                       `json:"schema_version"`
	Candidate             int                       `json:"candidate"`
	SourceProse           visualFileRecord          `json:"source_prose"`
	Plan                  visualFileRecord          `json:"plan"`
	Actions               []visualOpportunityRecord `json:"actions"`
	Assets                []visualAssetRecord       `json:"assets"`
	Article               visualFileRecord          `json:"article"`
	ReviewedRevision      string                    `json:"reviewed_revision"`
	ProseCharactersBefore int                       `json:"prose_characters_before"`
	ProseCharactersAfter  int                       `json:"prose_characters_after"`
}

// exampleVisualInput is the checked-in image the example brief stages.
const exampleVisualInput = "examples/assets/run-artifacts.png"

// candidateRevisionFromDisk recomputes the documented canonical candidate
// revision from the retained run, independently of the controller. A candidate
// with no referenced asset keeps the article digest; any referenced asset makes
// the revision the digest of the canonical article-plus-asset block.
func candidateRevisionFromDisk(t *testing.T, runDir string, manifest visualManifestRecord) string {
	t.Helper()
	article, err := os.ReadFile(filepath.Join(runDir, manifest.Article.Path))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Assets) == 0 {
		return revision(article)
	}
	assets := append([]visualAssetRecord(nil), manifest.Assets...)
	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	var block strings.Builder
	block.WriteString("write-uuter/candidate-revision/v1\n")
	block.WriteString("article " + revision(article) + "\n")
	fmt.Fprintf(&block, "assets %d\n", len(assets))
	for _, asset := range assets {
		data, readErr := os.ReadFile(filepath.Join(runDir, asset.Path))
		if readErr != nil {
			t.Fatal(readErr)
		}
		block.WriteString(asset.Path + " " + revision(data) + "\n")
	}
	return revision([]byte(block.String()))
}

func readVisualManifest(t *testing.T, runDir string, candidate int) visualManifestRecord {
	t.Helper()
	path := filepath.Join(runDir, "visuals", fmt.Sprintf("article-%03d", candidate), "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read visual manifest: %v", err)
	}
	var manifest visualManifestRecord
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode visual manifest: %v", err)
	}
	return manifest
}

// assertReviewsBoundTo proves all four lenses reviewed exactly the given
// revision, which is what the accepted article and its assets hash to.
func assertReviewsBoundTo(t *testing.T, runDir string, candidate int, want string) {
	t.Helper()
	for _, lens := range []string{"evidence", "story", "clarity", "copy"} {
		path := filepath.Join(runDir, "reviews", fmt.Sprintf("article-%03d", candidate), lens, "result.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s review: %v", lens, err)
		}
		var result struct {
			ReviewedRevision string `json:"reviewed_revision"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if result.ReviewedRevision != want {
			t.Errorf("%s lens reviewed %q, want %q", lens, result.ReviewedRevision, want)
		}
	}
}

func assertVisualPassOrder(t *testing.T, records []invocationRecord, candidate int) {
	t.Helper()
	var sequence []string
	for _, record := range records {
		if record.Candidate != candidate {
			continue
		}
		switch {
		case record.Role == "writer":
			if workspaceFileSet(record)["context/visual-plan.json"] {
				sequence = append(sequence, record.Invocation+":writer-assembly")
			} else {
				sequence = append(sequence, record.Invocation+":writer-prose")
			}
		case record.Role == "visual_editor":
			sequence = append(sequence, record.Invocation+":visual_editor")
		case strings.HasPrefix(record.Role, "reviewer_"):
			sequence = append(sequence, record.Invocation+":"+record.Lens)
		}
	}
	sort.Strings(sequence)
	var roles []string
	for _, item := range sequence {
		roles = append(roles, strings.SplitN(item, ":", 2)[1])
	}
	want := []string{"writer-prose", "visual_editor", "writer-assembly", "evidence", "story", "clarity", "copy"}
	if strings.Join(roles, ",") != strings.Join(want, ",") {
		t.Errorf("candidate %03d ran %v, want %v", candidate, roles, want)
	}
}

// A visual opportunity produces a plan, an inline Mermaid diagram, an
// assembled candidate, four bound reviews, and a published article.
func TestBlackBoxVisualPassProducesMermaidCandidate(t *testing.T) {
	run := executeScenario(t, "happy")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "succeeded" || state.CurrentCandidate != 1 {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	for _, relative := range []string{
		"visual-inputs.json", "drafts/article-001-prose.md", "drafts/article-001.md",
		"visuals/article-001/plan.md", "visuals/article-001/manifest.json", "article.md",
	} {
		if data, err := os.ReadFile(filepath.Join(run.runDir, relative)); err != nil || len(data) == 0 {
			t.Errorf("required artifact %s is missing/empty: %v", relative, err)
		}
	}
	prose, err := os.ReadFile(filepath.Join(run.runDir, "drafts", "article-001-prose.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(prose), "```mermaid") {
		t.Error("prose draft already contained the diagram the visual pass owns")
	}
	article, err := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(article), "```mermaid") {
		t.Fatalf("accepted article carries no Mermaid diagram:\n%s", article)
	}
	plan, err := os.ReadFile(filepath.Join(run.runDir, "visuals", "article-001", "plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"vis-001", "mermaid", "vis-002", "none"} {
		if !strings.Contains(string(plan), expected) {
			t.Errorf("plan.md does not record %q:\n%s", expected, plan)
		}
	}
	manifest := readVisualManifest(t, run.runDir, 1)
	if manifest.SchemaVersion != 1 || manifest.Candidate != 1 {
		t.Errorf("unexpected manifest header: %+v", manifest)
	}
	if manifest.SourceProse.SHA256 != revision(prose) {
		t.Errorf("manifest is not bound to the source prose revision")
	}
	if manifest.Article.SHA256 != revision(article) {
		t.Errorf("manifest is not bound to the assembled candidate")
	}
	if len(manifest.Assets) != 0 {
		t.Errorf("a Mermaid-only plan placed assets: %+v", manifest.Assets)
	}
	if manifest.ProseCharactersAfter >= manifest.ProseCharactersBefore {
		t.Errorf("assembled candidate did not shorten the explanation: %d -> %d",
			manifest.ProseCharactersBefore, manifest.ProseCharactersAfter)
	}
	want := candidateRevisionFromDisk(t, run.runDir, manifest)
	if manifest.ReviewedRevision != want || state.CurrentRevision != want {
		t.Errorf("revision binding: manifest=%q workflow=%q recomputed=%q",
			manifest.ReviewedRevision, state.CurrentRevision, want)
	}
	assertReviewsBoundTo(t, run.runDir, 1, want)
	records := readInvocationRecords(t, run.fixtureDir)
	assertVisualPassOrder(t, records, 1)
	assertProcessesGone(t, records)
}

// A staged regular image is copied, referenced by a safe relative path,
// included in the candidate revision, and published with the article.
func TestBlackBoxVisualPassPlacesStagedLocalAsset(t *testing.T) {
	run := executeScenario(t, "visual_asset")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "succeeded" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	manifest := readVisualManifest(t, run.runDir, 1)
	if len(manifest.Assets) != 1 {
		t.Fatalf("got %d placed assets, want 1: %+v", len(manifest.Assets), manifest.Assets)
	}
	asset := manifest.Assets[0]
	if asset.Path != "visuals/article-001/assets/vin-001.png" {
		t.Errorf("asset path = %q", asset.Path)
	}
	if asset.Origin != "brief" || asset.Source != exampleVisualInput {
		t.Errorf("asset lost its provenance: %+v", asset)
	}
	if asset.MediaType != "image/png" || strings.TrimSpace(asset.AltText) == "" {
		t.Errorf("asset record is incomplete: %+v", asset)
	}
	info, err := os.Lstat(filepath.Join(run.runDir, asset.Path))
	if err != nil {
		t.Fatalf("placed asset missing: %v", err)
	}
	if info.Mode().Perm() != 0o444 || !info.Mode().IsRegular() {
		t.Errorf("placed asset mode = %v, want a read-only regular file", info.Mode())
	}
	source, err := os.ReadFile(filepath.Join(repositoryRoot(t), exampleVisualInput))
	if err != nil {
		t.Fatal(err)
	}
	assertExactFiles(t, filepath.Join(repositoryRoot(t), exampleVisualInput), filepath.Join(run.runDir, asset.Path))
	if asset.SHA256 != revision(source) || asset.ByteSize != len(source) {
		t.Errorf("asset record does not describe the staged bytes: %+v", asset)
	}
	article, err := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	if err != nil {
		t.Fatal(err)
	}
	reference := fmt.Sprintf("![%s](%s)", asset.AltText, asset.Path)
	if !strings.Contains(string(article), reference) {
		t.Fatalf("accepted article does not reference the placed asset as %q:\n%s", reference, article)
	}
	want := candidateRevisionFromDisk(t, run.runDir, manifest)
	if want == revision(article) {
		t.Error("a candidate with a referenced asset must not keep the plain article digest")
	}
	if manifest.ReviewedRevision != want || state.CurrentRevision != want {
		t.Errorf("revision binding: manifest=%q workflow=%q recomputed=%q",
			manifest.ReviewedRevision, state.CurrentRevision, want)
	}
	assertReviewsBoundTo(t, run.runDir, 1, want)
	records := readInvocationRecords(t, run.fixtureDir)
	assertVisualPassOrder(t, records, 1)
	assertProcessesGone(t, records)
}

// An explicit `none` or `restructure_text` decision finishes the run without
// inventing an image.
func TestBlackBoxVisualPassCanFinishWithNoVisual(t *testing.T) {
	run := executeScenario(t, "visual_none")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "succeeded" {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	manifest := readVisualManifest(t, run.runDir, 1)
	if len(manifest.Assets) != 0 {
		t.Errorf("a no-visual plan placed assets: %+v", manifest.Assets)
	}
	actions := map[string]bool{}
	for _, action := range manifest.Actions {
		actions[action.Action] = true
	}
	if !actions["none"] || !actions["restructure_text"] {
		t.Errorf("plan did not record the explicit no-visual decisions: %+v", manifest.Actions)
	}
	if _, err := os.Lstat(filepath.Join(run.runDir, "visuals", "article-001", "assets")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("no-visual candidate created an asset directory: %v", err)
	}
	article, err := os.ReadFile(filepath.Join(run.runDir, "article.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(article), "```mermaid") || strings.Contains(string(article), "![") {
		t.Errorf("no-visual candidate invented a visual:\n%s", article)
	}
	want := candidateRevisionFromDisk(t, run.runDir, manifest)
	if want != revision(article) {
		t.Errorf("a candidate with no referenced asset must keep the plain article digest")
	}
	assertReviewsBoundTo(t, run.runDir, 1, want)
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

// A PM-validated finding creates the next candidate, reruns the visual and
// assembly passes for it, and restarts review at Evidence.
func TestBlackBoxVisualPassRerunsForEveryRevisionCandidate(t *testing.T) {
	run := executeScenario(t, "mustfix_once")
	if run.err != nil {
		t.Fatalf("CLI failed: %v\n%s", run.err, run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "succeeded" || state.CurrentCandidate != 2 {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	for _, relative := range []string{
		"drafts/article-001-prose.md", "drafts/article-001.md",
		"visuals/article-001/plan.md", "visuals/article-001/manifest.json",
		"drafts/article-002-prose.md", "drafts/article-002.md",
		"visuals/article-002/plan.md", "visuals/article-002/manifest.json",
	} {
		if _, err := os.Stat(filepath.Join(run.runDir, relative)); err != nil {
			t.Errorf("missing preserved artifact %s: %v", relative, err)
		}
	}
	// The visual pass does not consume a candidate: the run still finished
	// inside the unchanged three-candidate budget.
	if _, err := os.Stat(filepath.Join(run.runDir, "drafts", "article-003-prose.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the visual pass consumed an extra candidate: %v", err)
	}
	records := readInvocationRecords(t, run.fixtureDir)
	assertVisualPassOrder(t, records, 2)
	// Candidate 001 stopped at Evidence, so its later lenses never ran.
	if _, err := os.Stat(filepath.Join(run.runDir, "reviews", "article-001", "story")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("candidate 001 continued past the must-fix lens")
	}
	manifest := readVisualManifest(t, run.runDir, 2)
	want := candidateRevisionFromDisk(t, run.runDir, manifest)
	if state.CurrentRevision != want {
		t.Errorf("workflow revision %q is not the candidate 002 binding %q", state.CurrentRevision, want)
	}
	assertReviewsBoundTo(t, run.runDir, 2, want)
	assertProcessesGone(t, records)
}

// A plan bound to another prose revision cannot advance.
func TestBlackBoxStaleVisualPlanCannotAdvance(t *testing.T) {
	run := executeScenario(t, "visual_stale")
	if run.err == nil {
		t.Fatalf("CLI accepted a stale visual plan: %s", run.output)
	}
	state := readWorkflow(t, run.runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "stale visual plan source revision") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	for _, forbidden := range []string{"drafts/article-001.md", "visuals/article-001/plan.md", "article.md"} {
		if _, err := os.Lstat(filepath.Join(run.runDir, forbidden)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s survived a rejected visual plan: %v", forbidden, err)
		}
	}
	// The prose draft the plan was measured against is preserved for inspection.
	if _, err := os.Stat(filepath.Join(run.runDir, "drafts", "article-001-prose.md")); err != nil {
		t.Errorf("prose draft was not preserved: %v", err)
	}
	assertProcessesGone(t, readInvocationRecords(t, run.fixtureDir))
}

// Replacing a referenced asset after review changes the revision and prevents
// publication.
func TestBlackBoxAssetReplacementAfterReviewPreventsPublication(t *testing.T) {
	binary, fake, runDir, fixtureDir := prepareScenario(t, "visual_asset")
	barrier := filepath.Join(t.TempDir(), "asset-boundary")
	command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
	command.Env = append(command.Env, "WRITE_UUTER_TEST_FINAL_BOUNDARY_BARRIER="+barrier)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForPath(t, barrier+".ready")
	asset := filepath.Join(runDir, "visuals", "article-001", "assets", "vin-001.png")
	original, err := os.ReadFile(asset)
	if err != nil {
		t.Fatal(err)
	}
	replacement := append(append([]byte(nil), original...), []byte("\n<!-- replaced after review -->\n")...)
	if err := os.Chmod(asset, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(asset, replacement, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(barrier+".continue", []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("CLI published an article whose reviewed asset had been replaced")
	}
	state := readWorkflow(t, runDir)
	if state.Status != "blocked" || !strings.Contains(state.BlockReason, "no longer matches the reviewed bytes") {
		t.Fatalf("unexpected workflow: %+v", state)
	}
	assertNoArticle(t, runDir)
	assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
}

// A malformed manifest or a stale source prose revision discovered at the
// publication boundary prevents publication.
func TestBlackBoxTamperedVisualBindingPreventsPublication(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		relative string
		contents string
		reason   string
	}{
		{"malformed manifest", "visuals/article-001/manifest.json", "{ not json\n", "invalid visual manifest"},
		{"stale source prose", "drafts/article-001-prose.md", "# Replaced prose draft\n", "stale source prose revision"},
		{"rewritten plan", "visuals/article-001/plan.md", "# Replaced plan\n", "changed after it was accepted"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
			barrier := filepath.Join(t.TempDir(), "binding-boundary")
			command := newRunCommand(t, binary, fake, runDir, "5s", filepath.Join(repositoryRoot(t), "examples", "brief.md"))
			command.Env = append(command.Env, "WRITE_UUTER_TEST_FINAL_BOUNDARY_BARRIER="+barrier)
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			waitForPath(t, barrier+".ready")
			target := filepath.Join(runDir, testCase.relative)
			if err := os.Chmod(target, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte(testCase.contents), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(barrier+".continue", []byte("continue\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := command.Wait(); err == nil {
				t.Fatalf("CLI published after %s", testCase.name)
			}
			state := readWorkflow(t, runDir)
			if state.Status != "blocked" || !strings.Contains(state.BlockReason, testCase.reason) {
				t.Fatalf("unexpected workflow: %+v", state)
			}
			assertNoArticle(t, runDir)
			assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
		})
	}
}

// Every documented visual-pass rejection blocks the run, preserves the
// artifacts produced so far, publishes no article, and leaves no run-owned
// process behind.
func TestBlackBoxVisualPassRejectionsBlockAndCleanUp(t *testing.T) {
	for _, testCase := range []struct {
		scenario string
		timeout  string
		reason   string
	}{
		{"visual_unstaged", "5s", "is not a controller-staged visual input"},
		{"visual_bad_action", "5s", "unsupported action"},
		{"visual_bad_json", "5s", "invalid visual plan"},
		{"visual_missing_report", "5s", "visual_editor artifact contract failed"},
		{"visual_edit_prose", "5s", "Visual Editor edited the prose draft input"},
		{"visual_duplicate_prose", "5s", "must shorten or replace the explanation it carries"},
		// The supported inline form is exactly what Go checks: a target the
		// validated plan never placed, and a planned asset written at a
		// relative path other than the staged one it bound.
		{"visual_unplanned_inline", "5s", `references image "visuals/article-999/assets/unplanned.png", which the validated plan did not place`},
		{"visual_wrong_asset_target", "5s", "assembled candidate does not reference"},
		{"visual_timeout", "2s", "timed out"},
	} {
		t.Run(testCase.scenario, func(t *testing.T) {
			run := executeScenarioWithTimeout(t, testCase.scenario, testCase.timeout)
			if run.err == nil {
				t.Fatalf("CLI accepted %s: %s", testCase.scenario, run.output)
			}
			state := readWorkflow(t, run.runDir)
			if state.Status != "blocked" {
				t.Fatalf("status = %q, want blocked (%s)", state.Status, run.output)
			}
			if !strings.Contains(state.BlockReason, testCase.reason) {
				t.Fatalf("block reason %q does not contain %q", state.BlockReason, testCase.reason)
			}
			assertNoArticle(t, run.runDir)
			records := readInvocationRecords(t, run.fixtureDir)
			assertProcessesGone(t, records)
			privatePaths, _ := filepath.Glob(filepath.Join(filepath.Dir(run.runDir), ".write-uuter-private-*"))
			if len(privatePaths) != 0 {
				t.Fatalf("%s left private runtime roots: %v", testCase.scenario, privatePaths)
			}
		})
	}
}

// visualInputBriefTemplate is a complete brief whose only variable part is the
// optional `## Visual inputs` list under test.
const visualInputBriefTemplate = `# Brief

## Question

How does the controller validate a staged visual input?

## Audience

Engineers reviewing the visual input contract.

## Provisional takeaway

Unsafe and unusable local images are rejected before any agent starts.

## Scope

The visual input contract.

## Out of scope

Everything else.

## Publication target

A repository note.

## Constraints

Keep it short.

## Done when

The contract is described.

## Source hints

## Visual inputs

%s
`

// runWithVisualInputs runs the CLI with a brief whose visual inputs are the
// given content-root-relative values, resolved against the given content root.
func runWithVisualInputs(t *testing.T, contentRoot string, values ...string) (string, string, error) {
	t.Helper()
	binary, fake, runDir, _ := prepareScenario(t, "happy")
	var list strings.Builder
	for _, value := range values {
		list.WriteString("- " + value + "\n")
	}
	briefPath := filepath.Join(t.TempDir(), "brief.md")
	if err := os.WriteFile(briefPath, []byte(fmt.Sprintf(visualInputBriefTemplate, list.String())), 0o644); err != nil {
		t.Fatal(err)
	}
	command := newRunCommand(t, binary, fake, runDir, "5s", briefPath)
	command.Dir = contentRoot
	output, err := command.CombinedOutput()
	return string(output), runDir, err
}

// Unsafe, unusable, and unsupported visual inputs are rejected before the run
// directory exists, so nothing escapes the run boundary and no agent starts.
func TestBlackBoxUnsafeVisualInputsAreRejectedBeforeAnyAgentStarts(t *testing.T) {
	contentRoot := t.TempDir()
	assets := filepath.Join(contentRoot, "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(repositoryRoot(t), exampleVisualInput))
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) string {
		path := filepath.Join(assets, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	real := write("real.png", source)
	write("wrong-signature.png", []byte("this is not a PNG at all"))
	write("unsupported.gif", []byte("GIF89a"))
	write("oversized.png", make([]byte, (10<<20)+1))
	if err := os.Symlink(real, filepath.Join(assets, "link.png")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(assets, "directory.png"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(assets, "special.png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(assets, filepath.Join(contentRoot, "linked-assets")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.png")
	if err := os.WriteFile(outside, source, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, testCase := range []struct {
		name   string
		value  string
		reason string
	}{
		{"absolute path", outside, "path must be relative to the content root"},
		{"parent traversal", "../outside.png", "path escapes the content root"},
		{"home shorthand", "~/outside.png", "path must be relative to the content root"},
		{"symlinked file", "assets/link.png", "path is a symlink"},
		{"symlinked parent", "linked-assets/real.png", "path component is not a directory"},
		{"directory", "assets/directory.png", "path is a directory"},
		{"special file", "assets/special.png", "path is not a regular file"},
		{"missing file", "assets/missing.png", "no such file or directory"},
		{"unsupported format", "assets/unsupported.gif", "is not one of the supported formats"},
		{"signature mismatch", "assets/wrong-signature.png", "file signature does not match"},
		{"oversized file", "assets/oversized.png", "byte limit"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			output, runDir, err := runWithVisualInputs(t, contentRoot, testCase.value)
			if err == nil {
				t.Fatalf("CLI accepted %s: %s", testCase.name, output)
			}
			if !strings.Contains(output, testCase.reason) {
				t.Fatalf("output %q does not contain %q", output, testCase.reason)
			}
			if _, statErr := os.Lstat(runDir); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("rejected visual input created a run directory: %v", statErr)
			}
			if _, statErr := os.Lstat(outside); statErr != nil {
				t.Fatalf("rejection disturbed a file outside the content root: %v", statErr)
			}
		})
	}

	t.Run("safe in-root regular file is accepted", func(t *testing.T) {
		output, runDir, err := runWithVisualInputs(t, contentRoot, "assets/real.png")
		if err != nil {
			t.Fatalf("CLI rejected a safe in-root regular image: %v\n%s", err, output)
		}
		data, readErr := os.ReadFile(filepath.Join(runDir, "visual-inputs.json"))
		if readErr != nil {
			t.Fatalf("read visual input manifest: %v", readErr)
		}
		var manifest struct {
			SchemaVersion int `json:"schema_version"`
			Inputs        []struct {
				ID        string `json:"id"`
				Origin    string `json:"origin"`
				Source    string `json:"source"`
				MediaType string `json:"media_type"`
				ByteSize  int    `json:"byte_size"`
				SHA256    string `json:"sha256"`
			} `json:"inputs"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.SchemaVersion != 1 || len(manifest.Inputs) != 1 {
			t.Fatalf("unexpected input manifest: %s", data)
		}
		input := manifest.Inputs[0]
		if input.ID != "vin-001" || input.Origin != "brief" || input.Source != "assets/real.png" ||
			input.MediaType != "image/png" || input.ByteSize != len(source) || input.SHA256 != revision(source) {
			t.Fatalf("input manifest lost provenance: %+v", input)
		}
		info, statErr := os.Lstat(filepath.Join(runDir, "visual-inputs.json"))
		if statErr != nil || info.Mode().Perm() != 0o444 {
			t.Fatalf("visual-inputs.json is not a read-only regular artifact: %v %v", info, statErr)
		}
	})
}

// An empty or absent `## Visual inputs` section stays backward compatible:
// Mermaid and text restructuring remain available and no manifest is written.
func TestBlackBoxEmptyOrAbsentVisualInputsSectionStaysCompatible(t *testing.T) {
	contentRoot := t.TempDir()
	t.Run("empty section", func(t *testing.T) {
		output, runDir, err := runWithVisualInputs(t, contentRoot)
		if err != nil {
			t.Fatalf("CLI rejected an empty Visual inputs section: %v\n%s", err, output)
		}
		if _, statErr := os.Lstat(filepath.Join(runDir, "visual-inputs.json")); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("a run that staged nothing wrote a visual input manifest: %v", statErr)
		}
		article, readErr := os.ReadFile(filepath.Join(runDir, "article.md"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(article), "```mermaid") {
			t.Errorf("Mermaid was not available without staged images:\n%s", article)
		}
	})

	t.Run("absent section", func(t *testing.T) {
		binary, fake, runDir, fixtureDir := prepareScenario(t, "happy")
		briefPath := filepath.Join(t.TempDir(), "brief.md")
		body := fmt.Sprintf(visualInputBriefTemplate, "")
		body = strings.ReplaceAll(body, "\n## Visual inputs\n", "")
		if err := os.WriteFile(briefPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		command := newRunCommand(t, binary, fake, runDir, "5s", briefPath)
		command.Dir = contentRoot
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI rejected a brief without a Visual inputs section: %v\n%s", err, output)
		}
		if state := readWorkflow(t, runDir); state.Status != "succeeded" {
			t.Fatalf("unexpected workflow: %+v", state)
		}
		assertProcessesGone(t, readInvocationRecords(t, fixtureDir))
	})
}
