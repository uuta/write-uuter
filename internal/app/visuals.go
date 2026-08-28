package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Visual and reading-flow pass contract.
//
// The Visual Editor runs once per candidate, between the Writer prose draft
// and the four review lenses. It never edits a durable candidate: it emits an
// inspectable plan plus the artifacts a fresh Writer assembly invocation
// applies. Every documented bound below is part of the contract and is
// asserted by the black-box suite.
const (
	// visualInputManifestArtifact is the controller-generated record of every
	// staged visual input. No agent writes it.
	visualInputManifestArtifact = "visual-inputs.json"
	// visualSchemaVersion is the only plan/manifest schema this binary accepts.
	visualSchemaVersion = 1
	// visualMaxInputBytes is the documented per-file limit for a staged local
	// image (10 MiB), matching the screenshot ceiling.
	visualMaxInputBytes = 10 << 20
	// visualMaxInputs bounds how many local images one brief may stage.
	visualMaxInputs = 20
	// visualMaxOpportunities bounds one plan, so an inspectable artifact stays
	// inspectable.
	visualMaxOpportunities = 20
	visualMaxMermaidBytes  = 8192
	// visualMaxLocationBytes/visualMaxRationaleBytes/visualMaxAltTextBytes
	// bound the plan's free text. Alt text gets the widest budget: a dense
	// diagram needs a real description, not a caption. Every bound is stated
	// in the Visual Editor assignment, so the contract is knowable in advance.
	visualMaxLocationBytes  = 512
	visualMaxRationaleBytes = 1024
	visualMaxAltTextBytes   = 1024
	// visualBriefSection is the optional level-two brief heading. An absent or
	// empty section is valid and means the run stages no local image.
	visualBriefSection = "Visual inputs"
	// visualInputOriginBrief/Screenshot record where a staged input came from.
	visualInputOriginBrief      = "brief"
	visualInputOriginScreenshot = "screenshot"
)

// visualPlanActions is the exact action vocabulary of this slice. There is no
// default and no fallback: an unknown action fails the plan.
var visualPlanActions = []string{"mermaid", "existing_local_asset", "restructure_text", "none"}

// visualMediaType binds an accepted media type to its accepted extensions and
// its file signature. Both the extension and the signature must agree, so a
// renamed or mistyped file is rejected before any agent starts.
type visualMediaType struct {
	mediaType  string
	extensions []string
	matches    func([]byte) bool
}

var visualMediaTypes = []visualMediaType{
	{
		mediaType:  "image/png",
		extensions: []string{".png"},
		matches:    func(data []byte) bool { return bytes.HasPrefix(data, pngSignature) },
	},
	{
		mediaType:  "image/jpeg",
		extensions: []string{".jpg", ".jpeg"},
		matches: func(data []byte) bool {
			return len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
		},
	},
	{
		mediaType:  "image/webp",
		extensions: []string{".webp"},
		matches: func(data []byte) bool {
			return len(data) >= 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
		},
	},
}

func visualMediaForExtension(extension string) (visualMediaType, bool) {
	for _, media := range visualMediaTypes {
		for _, candidate := range media.extensions {
			if candidate == extension {
				return media, true
			}
		}
	}
	return visualMediaType{}, false
}

func visualSupportedExtensions() string {
	var all []string
	for _, media := range visualMediaTypes {
		all = append(all, media.extensions...)
	}
	return strings.Join(all, ", ")
}

// visualInput is one validated image the controller staged for this run. The
// bytes are held by the controller, so replacing the source file after
// validation cannot change what an agent or a candidate sees.
type visualInput struct {
	ID        string
	Origin    string
	Source    string
	Extension string
	MediaType string
	Data      []byte
}

// StagedPath is where the controller stages this input inside an agent
// workspace, below that workspace's read-only `context/` directory.
func (input visualInput) StagedPath() string {
	return "visual-inputs/" + input.ID + input.Extension
}

// VisualInputManifest is controller-generated. No agent writes it.
type VisualInputManifest struct {
	SchemaVersion int                 `json:"schema_version"`
	Inputs        []VisualInputRecord `json:"inputs"`
}

type VisualInputRecord struct {
	ID         string `json:"id"`
	Origin     string `json:"origin"`
	Source     string `json:"source"`
	MediaType  string `json:"media_type"`
	ByteSize   int    `json:"byte_size"`
	SHA256     string `json:"sha256"`
	StagedPath string `json:"staged_path"`
}

// VisualPlan is the Visual Editor's machine-readable plan. It is validated
// completely before any asset is placed and before the assembly pass starts.
type VisualPlan struct {
	SchemaVersion  int                 `json:"schema_version"`
	SourceRevision string              `json:"source_revision"`
	Opportunities  []VisualOpportunity `json:"opportunities"`
}

// VisualOpportunity is one evaluated explanation, its location, the selected
// action, and the concrete reason that action should improve explanation or
// reading flow.
type VisualOpportunity struct {
	ID        string `json:"id"`
	Location  string `json:"location"`
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
	Mermaid   string `json:"mermaid,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	AltText   string `json:"alt_text,omitempty"`
}

// VisualManifest is controller-generated after the assembly pass. It binds the
// plan, the source prose revision, the assembled candidate, and every
// referenced local asset into the single revision the four lenses review.
type VisualManifest struct {
	SchemaVersion         int                 `json:"schema_version"`
	Candidate             int                 `json:"candidate"`
	SourceProse           VisualFileRecord    `json:"source_prose"`
	Plan                  VisualFileRecord    `json:"plan"`
	Actions               []VisualOpportunity `json:"actions"`
	Assets                []VisualAssetRecord `json:"assets"`
	Article               VisualFileRecord    `json:"article"`
	ReviewedRevision      string              `json:"reviewed_revision"`
	ProseCharactersBefore int                 `json:"prose_characters_before"`
	ProseCharactersAfter  int                 `json:"prose_characters_after"`
}

type VisualFileRecord struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type VisualAssetRecord struct {
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

func proseDraftPath(candidate int) string {
	return fmt.Sprintf("drafts/article-%03d-prose.md", candidate)
}

func candidateDraftPath(candidate int) string {
	return fmt.Sprintf("drafts/article-%03d.md", candidate)
}

func visualDirPath(candidate int) string {
	return fmt.Sprintf("visuals/article-%03d", candidate)
}

func visualPlanPath(candidate int) string {
	return visualDirPath(candidate) + "/plan.md"
}

func visualManifestPath(candidate int) string {
	return visualDirPath(candidate) + "/manifest.json"
}

func visualAssetPath(candidate int, id, extension string) string {
	return visualDirPath(candidate) + "/assets/" + id + extension
}

// candidateRevision is the canonical deterministic revision of one candidate.
// It covers the assembled Markdown and the bytes of every referenced local
// asset, so an asset cannot be replaced after review without changing the
// revision that all four lenses accepted.
//
// A candidate that references no local asset keeps the digest of its Markdown,
// so a run without visual assets is byte-identical to one produced before this
// pass existed. Any referenced asset makes the revision the digest of a fixed
// canonical block naming the article digest and every asset path and digest in
// lexicographic path order.
func candidateRevision(article []byte, assets []VisualAssetRecord) string {
	if len(assets) == 0 {
		return revisionFor(article)
	}
	ordered := append([]VisualAssetRecord(nil), assets...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	var block strings.Builder
	block.WriteString("write-uuter/candidate-revision/v1\n")
	block.WriteString("article " + revisionFor(article) + "\n")
	fmt.Fprintf(&block, "assets %d\n", len(ordered))
	for _, asset := range ordered {
		block.WriteString(asset.Path + " " + asset.SHA256 + "\n")
	}
	return revisionFor([]byte(block.String()))
}

// loadVisualInputs validates every image the brief lists under the optional
// `## Visual inputs` section and holds a private copy of its bytes. It runs
// before the run directory is created, so an unsafe or unusable path fails
// before anything exists and before any agent starts.
func (control *controller) loadVisualInputs() error {
	section := strings.TrimSpace(control.brief.Sections[visualBriefSection])
	if section == "" {
		return nil
	}
	root, err := os.OpenRoot(control.contentRoot)
	if err != nil {
		return fmt.Errorf("open content root for visual inputs: %w", err)
	}
	defer root.Close()
	seen := make(map[string]bool)
	index := 0
	for _, line := range strings.Split(section, "\n") {
		value := strings.Trim(sourceHintValue(line), "`<>")
		if value == "" {
			continue
		}
		if index == visualMaxInputs {
			return fmt.Errorf("brief section %q lists more than %d visual inputs", visualBriefSection, visualMaxInputs)
		}
		input, err := loadVisualInput(root, value)
		if err != nil {
			return fmt.Errorf("visual input %q: %w", value, err)
		}
		if seen[strings.ToLower(input.Source)] {
			return fmt.Errorf("brief section %q lists visual input %q twice", visualBriefSection, value)
		}
		seen[strings.ToLower(input.Source)] = true
		index++
		input.ID = fmt.Sprintf("vin-%03d", index)
		control.visualInputs = append(control.visualInputs, input)
	}
	return nil
}

// loadVisualInput validates one content-root-relative image and returns a
// private copy of its bytes. Absolute paths, parent traversal, symlinked path
// components, symlinked or non-regular targets, unsupported extensions,
// signature mismatches, and oversized files are all rejected here.
func loadVisualInput(root *os.Root, name string) (visualInput, error) {
	var input visualInput
	if filepath.IsAbs(name) || strings.HasPrefix(name, "~") {
		return input, fmt.Errorf("path must be relative to the content root")
	}
	clean := filepath.Clean(name)
	if !filepath.IsLocal(clean) {
		return input, fmt.Errorf("path escapes the content root")
	}
	extension := strings.ToLower(filepath.Ext(clean))
	media, supported := visualMediaForExtension(extension)
	if !supported {
		return input, fmt.Errorf("extension %q is not one of the supported formats %s", extension, visualSupportedExtensions())
	}
	if err := validateRootParents(root, clean); err != nil {
		return input, err
	}
	expected, err := root.Lstat(clean)
	if err != nil {
		return input, err
	}
	if expected.Mode()&os.ModeSymlink != 0 {
		return input, fmt.Errorf("path is a symlink")
	}
	if expected.IsDir() {
		return input, fmt.Errorf("path is a directory")
	}
	if !expected.Mode().IsRegular() {
		return input, fmt.Errorf("path is not a regular file")
	}
	file, err := root.OpenFile(clean, os.O_RDONLY, 0)
	if err != nil {
		return input, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return input, err
	}
	if !os.SameFile(expected, opened) {
		return input, fmt.Errorf("file identity changed while it was opened")
	}
	if !opened.Mode().IsRegular() {
		return input, fmt.Errorf("path is not a regular file")
	}
	if opened.Size() > visualMaxInputBytes {
		return input, fmt.Errorf("file is %d bytes, more than the %d byte limit", opened.Size(), visualMaxInputBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, visualMaxInputBytes+1))
	if err != nil {
		return input, err
	}
	if len(data) > visualMaxInputBytes {
		return input, fmt.Errorf("file exceeds the %d byte limit", visualMaxInputBytes)
	}
	if len(data) == 0 {
		return input, fmt.Errorf("file is empty")
	}
	if !media.matches(data) {
		return input, fmt.Errorf("file signature does not match the %s extension", extension)
	}
	return visualInput{
		Origin: visualInputOriginBrief, Source: filepath.ToSlash(clean),
		Extension: extension, MediaType: media.mediaType, Data: data,
	}, nil
}

// adoptScreenshotVisualInputs adds every controller-captured screenshot to the
// visual input pool. Their bytes are already validated and immutable, so the
// visual pass consumes them in place instead of re-acquiring anything.
func (control *controller) adoptScreenshotVisualInputs() error {
	for _, request := range control.screenshotRequests {
		relative := screenshotAssetPath(request.ID)
		data, err := control.store.readRegular(relative)
		if err != nil {
			return fmt.Errorf("read captured screenshot %s: %w", relative, err)
		}
		control.visualInputs = append(control.visualInputs, visualInput{
			ID: request.ID, Origin: visualInputOriginScreenshot, Source: relative,
			Extension: ".png", MediaType: screenshotMediaType, Data: data,
		})
	}
	return nil
}

// publishVisualInputs writes the controller-generated input manifest once the
// pool is final. A run whose brief lists nothing and whose Researcher
// requested nothing writes no manifest and behaves exactly as before.
func (control *controller) publishVisualInputs() error {
	if err := control.adoptScreenshotVisualInputs(); err != nil {
		return err
	}
	if len(control.visualInputs) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(control.visualInputs))
	manifest := VisualInputManifest{SchemaVersion: visualSchemaVersion}
	for _, input := range control.visualInputs {
		folded := strings.ToLower(input.ID)
		if seen[folded] {
			return fmt.Errorf("visual input ID %q is used twice; IDs become file names and are compared case-insensitively", input.ID)
		}
		seen[folded] = true
		manifest.Inputs = append(manifest.Inputs, VisualInputRecord{
			ID: input.ID, Origin: input.Origin, Source: input.Source,
			MediaType: input.MediaType, ByteSize: len(input.Data),
			SHA256: revisionFor(input.Data), StagedPath: input.StagedPath(),
		})
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", visualInputManifestArtifact, err)
	}
	data = append(data, '\n')
	if err := control.store.writeAtomic(visualInputManifestArtifact, data, 0o444); err != nil {
		return fmt.Errorf("write %s: %w", visualInputManifestArtifact, err)
	}
	control.visualInputManifest = data
	return nil
}

func (control *controller) visualInput(id string) (visualInput, bool) {
	for _, input := range control.visualInputs {
		if input.ID == id {
			return input, true
		}
	}
	return visualInput{}, false
}

// stageVisualInputs copies every staged image into a role workspace as a
// read-only regular file. The bytes are the controller's private copy, never a
// re-read of the original path.
func (control *controller) stageVisualInputs(workspace *artifactStore) error {
	for _, input := range control.visualInputs {
		target := filepath.Join("context", input.StagedPath())
		if err := workspace.writeAtomic(target, input.Data, 0o444); err != nil {
			return err
		}
	}
	return nil
}

// parseVisualPlan validates the Visual Editor plan completely. Every rejection
// here happens before an asset is placed and before the assembly pass starts.
func parseVisualPlan(data []byte, proseRevision string, available map[string]visualInput) (VisualPlan, error) {
	var plan VisualPlan
	if strings.TrimSpace(string(data)) == "" {
		return plan, fmt.Errorf("visual plan is empty")
	}
	// Require the control fields to be present and correctly shaped before the
	// typed decode, so an absent or null list cannot decode to "no plan".
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return plan, fmt.Errorf("invalid visual plan: %w", err)
	}
	for _, required := range []string{"schema_version", "source_revision", "opportunities"} {
		if _, found := fields[required]; !found {
			return plan, fmt.Errorf("visual plan is missing required field %q", required)
		}
	}
	if trimmed := bytes.TrimSpace(fields["opportunities"]); len(trimmed) == 0 || trimmed[0] != '[' {
		return plan, fmt.Errorf("visual plan field %q must be an array", "opportunities")
	}
	if err := decodeStrictJSON(data, &plan); err != nil {
		return plan, fmt.Errorf("invalid visual plan: %w", err)
	}
	if plan.SchemaVersion != visualSchemaVersion {
		return plan, fmt.Errorf("unsupported visual plan schema_version %d: this binary supports %d", plan.SchemaVersion, visualSchemaVersion)
	}
	if plan.SourceRevision != proseRevision {
		return plan, fmt.Errorf("stale visual plan source revision: got %q, want %q", plan.SourceRevision, proseRevision)
	}
	if len(plan.Opportunities) == 0 {
		return plan, fmt.Errorf("visual plan records no evaluated opportunity")
	}
	if len(plan.Opportunities) > visualMaxOpportunities {
		return plan, fmt.Errorf("visual plan records %d opportunities, the limit is %d", len(plan.Opportunities), visualMaxOpportunities)
	}
	seenID := make(map[string]bool, len(plan.Opportunities))
	seenAsset := make(map[string]bool, len(plan.Opportunities))
	for index := range plan.Opportunities {
		opportunity := &plan.Opportunities[index]
		if err := validateVisualOpportunity(opportunity, available, seenID, seenAsset); err != nil {
			return plan, fmt.Errorf("visual plan entry %d: %w", index, err)
		}
	}
	return plan, nil
}

func validateVisualOpportunity(opportunity *VisualOpportunity, available map[string]visualInput, seenID, seenAsset map[string]bool) error {
	if err := validatePlainIdentifier("opportunity ID", opportunity.ID); err != nil {
		return err
	}
	if seenID[opportunity.ID] {
		return fmt.Errorf("duplicate opportunity ID %q", opportunity.ID)
	}
	seenID[opportunity.ID] = true
	if err := validateBoundedText("location", opportunity.Location, visualMaxLocationBytes); err != nil {
		return err
	}
	if err := validateBoundedText("rationale", opportunity.Rationale, visualMaxRationaleBytes); err != nil {
		return err
	}
	supported := false
	for _, action := range visualPlanActions {
		if opportunity.Action == action {
			supported = true
			break
		}
	}
	if !supported {
		return fmt.Errorf("unsupported action %q: this slice supports exactly %s", opportunity.Action, strings.Join(visualPlanActions, ", "))
	}
	switch opportunity.Action {
	case "mermaid":
		if err := validateBoundedText("mermaid diagram", opportunity.Mermaid, visualMaxMermaidBytes); err != nil {
			return err
		}
		if strings.Contains(opportunity.Mermaid, "```") {
			return fmt.Errorf("mermaid diagram contains a code fence")
		}
		if opportunity.AssetID != "" || opportunity.AltText != "" {
			return fmt.Errorf("mermaid action must not name an asset or alt text")
		}
	case "existing_local_asset":
		if err := validatePlainIdentifier("asset ID", opportunity.AssetID); err != nil {
			return err
		}
		if _, staged := available[opportunity.AssetID]; !staged {
			return fmt.Errorf("asset %q is not a controller-staged visual input", opportunity.AssetID)
		}
		if seenAsset[opportunity.AssetID] {
			return fmt.Errorf("asset %q is placed more than once", opportunity.AssetID)
		}
		seenAsset[opportunity.AssetID] = true
		if err := validateBoundedText("alt text", opportunity.AltText, visualMaxAltTextBytes); err != nil {
			return err
		}
		if strings.ContainsAny(opportunity.AltText, "[]()") {
			return fmt.Errorf("alt text contains a Markdown link delimiter")
		}
		if opportunity.Mermaid != "" {
			return fmt.Errorf("existing_local_asset action must not carry a mermaid diagram")
		}
	case "restructure_text", "none":
		if opportunity.Mermaid != "" || opportunity.AssetID != "" || opportunity.AltText != "" {
			return fmt.Errorf("%s action must not carry a diagram, asset, or alt text", opportunity.Action)
		}
	}
	return nil
}

func validatePlainIdentifier(label, value string) error {
	if value == "" {
		return fmt.Errorf("%s is empty", label)
	}
	if len(value) > 64 {
		return fmt.Errorf("%s %q is longer than 64 characters", label, value)
	}
	for index, character := range value {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9':
		case (character == '-' || character == '_') && index > 0:
		default:
			return fmt.Errorf("%s %q is not a plain identifier", label, value)
		}
	}
	return nil
}

func validateBoundedText(label, value string, limit int) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is empty", label)
	}
	if len(value) > limit {
		return fmt.Errorf("%s is longer than %d bytes", label, limit)
	}
	for _, character := range value {
		if character != '\n' && character != '\t' && (character < 0x20 || character == 0x7f) {
			return fmt.Errorf("%s contains a control character", label)
		}
	}
	return nil
}

// validatePlanReport proves the human-readable plan actually documents every
// evaluated opportunity, so `plan.md` stays inspectable rather than decorative.
func validatePlanReport(report string, plan VisualPlan) error {
	for _, opportunity := range plan.Opportunities {
		if !strings.Contains(report, opportunity.ID) {
			return fmt.Errorf("plan.md does not record opportunity %s", opportunity.ID)
		}
		if !strings.Contains(report, opportunity.Action) {
			return fmt.Errorf("plan.md does not record the %s action selected for %s", opportunity.Action, opportunity.ID)
		}
	}
	return nil
}

// planAssets resolves the durable asset records one validated plan places.
func (control *controller) planAssets(candidate int, plan VisualPlan) ([]VisualAssetRecord, error) {
	var assets []VisualAssetRecord
	for _, opportunity := range plan.Opportunities {
		if opportunity.Action != "existing_local_asset" {
			continue
		}
		input, staged := control.visualInput(opportunity.AssetID)
		if !staged {
			return nil, fmt.Errorf("asset %q is not a controller-staged visual input", opportunity.AssetID)
		}
		assets = append(assets, VisualAssetRecord{
			ID: input.ID, OpportunityID: opportunity.ID,
			Path:      visualAssetPath(candidate, input.ID, input.Extension),
			Origin:    input.Origin,
			Source:    input.Source,
			MediaType: input.MediaType, ByteSize: len(input.Data),
			SHA256: revisionFor(input.Data), AltText: opportunity.AltText,
		})
	}
	return assets, nil
}

// visualPlacements counts the plan entries that put a diagram or an image into
// the article. Only those entries make the anti-duplication rule apply.
func visualPlacements(plan VisualPlan) int {
	count := 0
	for _, opportunity := range plan.Opportunities {
		if opportunity.Action == "mermaid" || opportunity.Action == "existing_local_asset" {
			count++
		}
	}
	return count
}

// validateAssembledCandidate proves the assembly pass applied exactly the
// validated plan: every planned diagram and image is present at its planned
// location, no unplanned image reference exists, and a candidate that gained a
// visual actually lost explanation instead of duplicating it.
func validateAssembledCandidate(article, prose []byte, plan VisualPlan, assets []VisualAssetRecord) error {
	text := normalizeNewlines(string(article))
	placed := make(map[string]VisualAssetRecord, len(assets))
	for _, asset := range assets {
		placed[asset.OpportunityID] = asset
	}
	for _, opportunity := range plan.Opportunities {
		switch opportunity.Action {
		case "mermaid":
			block := "```mermaid\n" + strings.TrimRight(normalizeNewlines(opportunity.Mermaid), "\n") + "\n```"
			if !strings.Contains(text, block) {
				return fmt.Errorf("assembled candidate does not contain the planned Mermaid diagram for %s", opportunity.ID)
			}
		case "existing_local_asset":
			asset, found := placed[opportunity.ID]
			if !found {
				return fmt.Errorf("no staged asset resolved for %s", opportunity.ID)
			}
			reference := fmt.Sprintf("![%s](%s)", opportunity.AltText, asset.Path)
			if !strings.Contains(text, reference) {
				return fmt.Errorf("assembled candidate does not reference %s as %s", asset.Path, reference)
			}
		}
	}
	allowed := make(map[string]bool, len(assets))
	for _, asset := range assets {
		allowed[asset.Path] = true
	}
	for _, target := range markdownImageTargets(text) {
		if !allowed[target] {
			return fmt.Errorf("assembled candidate references image %q, which the validated plan did not place", target)
		}
	}
	if visualPlacements(plan) == 0 {
		return nil
	}
	before := proseCharacterCount(string(prose))
	after := proseCharacterCount(text)
	if after >= before {
		return fmt.Errorf("assembled candidate kept %d explanation characters against %d in the prose draft: a placed visual must shorten or replace the explanation it carries, not duplicate it", after, before)
	}
	return nil
}

// markdownImageTargets returns the target of every image written in the exact
// inline form the Writer assembly contract emits: `![alt](target)` with the
// target on a single line. That is the only form a validated plan can bind to
// a staged asset and to the candidate revision, so it is the only form Go
// checks.
//
// write-uuter is not a CommonMark or HTML parser. Other Markdown image
// syntaxes and raw HTML are prohibited editorial output: the Writer and Visual
// Editor contracts forbid them and the Copy lens reviews Markdown mechanics
// and relative paths. Go's job is narrower and exact - every image written in
// the supported form must be one the validated plan placed.
//
// The scan never stops at text it does not recognize. Anything that is not the
// supported form advances the offset and the scan continues, so it cannot hide
// a later unplanned inline image.
func markdownImageTargets(markdown string) []string {
	var targets []string
	for offset := 0; offset < len(markdown); {
		start := strings.Index(markdown[offset:], "![")
		if start < 0 {
			break
		}
		start += offset
		closeAlt := strings.IndexByte(markdown[start:], ']')
		if closeAlt < 0 {
			// No `]` remains, so no supported form remains either.
			break
		}
		open := start + closeAlt + len("]")
		offset = open
		if open >= len(markdown) || markdown[open] != '(' {
			continue
		}
		open += len("(")
		closeTarget := strings.IndexAny(markdown[open:], ")\n")
		if closeTarget < 0 || markdown[open+closeTarget] == '\n' {
			continue
		}
		targets = append(targets, markdown[open:open+closeTarget])
		offset = open + closeTarget + len(")")
	}
	return targets
}

// proseCharacterCount measures explanation only. Fenced blocks and image
// references are excluded, so replacing a paragraph with a diagram registers as
// a reduction rather than as more characters.
func proseCharacterCount(markdown string) int {
	var prose strings.Builder
	fenced := false
	for _, line := range strings.Split(normalizeNewlines(markdown), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		prose.WriteString(line)
		prose.WriteByte('\n')
	}
	text := prose.String()
	for _, target := range markdownImageTargets(text) {
		if index := strings.Index(text, "]("+target+")"); index >= 0 {
			if start := strings.LastIndex(text[:index], "!["); start >= 0 {
				text = text[:start] + text[index+len("]("+target+")"):]
			}
		}
	}
	count := 0
	for _, character := range text {
		if !unicode.IsSpace(character) {
			count++
		}
	}
	return count
}

// validateProseDraft keeps ownership of visuals with the visual pass. The
// prose draft is explanation only; diagrams and images arrive in assembly.
func validateProseDraft(draft []byte) error {
	text := normalizeNewlines(string(draft))
	if strings.Contains(text, "```mermaid") {
		return fmt.Errorf("prose draft contains a Mermaid block; the Visual Editor pass owns diagrams")
	}
	if len(markdownImageTargets(text)) != 0 {
		return fmt.Errorf("prose draft contains an image reference; the Visual Editor pass owns visuals")
	}
	return nil
}

func normalizeNewlines(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

// buildVisualManifest binds the plan, the source prose revision, the assembled
// candidate, and every referenced local asset to the exact revision the four
// lenses review.
func buildVisualManifest(candidate int, plan VisualPlan, planReport, prose, article []byte, assets []VisualAssetRecord) VisualManifest {
	if assets == nil {
		assets = []VisualAssetRecord{}
	}
	return VisualManifest{
		SchemaVersion: visualSchemaVersion, Candidate: candidate,
		SourceProse:           VisualFileRecord{Path: proseDraftPath(candidate), SHA256: revisionFor(prose)},
		Plan:                  VisualFileRecord{Path: visualPlanPath(candidate), SHA256: revisionFor(planReport)},
		Actions:               plan.Opportunities,
		Assets:                assets,
		Article:               VisualFileRecord{Path: candidateDraftPath(candidate), SHA256: revisionFor(article)},
		ReviewedRevision:      candidateRevision(article, assets),
		ProseCharactersBefore: proseCharacterCount(string(prose)),
		ProseCharactersAfter:  proseCharacterCount(string(article)),
	}
}

// candidateBinding revalidates every durable artifact one candidate is bound
// to and returns its canonical reviewed revision plus the assembled Markdown.
// It is called before each lens starts and again at the publication boundary,
// so a malformed or stale manifest, a missing, unsafe, or non-regular asset, a
// stale source prose revision, and a post-review asset replacement all stop the
// run instead of reaching a reviewer or `article.md`.
func (control *controller) candidateBinding(candidate int) (string, []byte, error) {
	manifestData, err := control.store.readRegular(visualManifestPath(candidate))
	if err != nil {
		return "", nil, fmt.Errorf("read visual manifest for candidate %03d: %w", candidate, err)
	}
	var manifest VisualManifest
	if err := decodeStrictJSON(manifestData, &manifest); err != nil {
		return "", nil, fmt.Errorf("invalid visual manifest for candidate %03d: %w", candidate, err)
	}
	if manifest.SchemaVersion != visualSchemaVersion {
		return "", nil, fmt.Errorf("unsupported visual manifest schema_version %d: this binary supports %d", manifest.SchemaVersion, visualSchemaVersion)
	}
	if manifest.Candidate != candidate {
		return "", nil, fmt.Errorf("visual manifest names candidate %03d, want %03d", manifest.Candidate, candidate)
	}
	if manifest.SourceProse.Path != proseDraftPath(candidate) ||
		manifest.Plan.Path != visualPlanPath(candidate) ||
		manifest.Article.Path != candidateDraftPath(candidate) {
		return "", nil, fmt.Errorf("visual manifest for candidate %03d names an unexpected artifact path", candidate)
	}
	prose, err := control.store.readRegular(manifest.SourceProse.Path)
	if err != nil {
		return "", nil, err
	}
	if revisionFor(prose) != manifest.SourceProse.SHA256 {
		return "", nil, fmt.Errorf("stale source prose revision for candidate %03d", candidate)
	}
	planReport, err := control.store.readRegular(manifest.Plan.Path)
	if err != nil {
		return "", nil, err
	}
	if revisionFor(planReport) != manifest.Plan.SHA256 {
		return "", nil, fmt.Errorf("visual plan for candidate %03d changed after it was accepted", candidate)
	}
	article, err := control.store.readRegular(manifest.Article.Path)
	if err != nil {
		return "", nil, err
	}
	if revisionFor(article) != manifest.Article.SHA256 {
		return "", nil, fmt.Errorf("assembled candidate %03d changed outside the controller", candidate)
	}
	seen := make(map[string]bool, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		if asset.Path != visualAssetPath(candidate, asset.ID, filepath.Ext(asset.Path)) {
			return "", nil, fmt.Errorf("visual asset %q is outside the candidate asset directory", asset.Path)
		}
		if seen[asset.Path] {
			return "", nil, fmt.Errorf("visual manifest lists asset %q twice", asset.Path)
		}
		seen[asset.Path] = true
		media, supported := visualMediaForExtension(strings.ToLower(filepath.Ext(asset.Path)))
		if !supported || media.mediaType != asset.MediaType {
			return "", nil, fmt.Errorf("visual asset %q declares media type %q", asset.Path, asset.MediaType)
		}
		data, err := control.store.readRegular(asset.Path)
		if err != nil {
			return "", nil, fmt.Errorf("read visual asset %s: %w", asset.Path, err)
		}
		if len(data) != asset.ByteSize || revisionFor(data) != asset.SHA256 {
			return "", nil, fmt.Errorf("visual asset %s no longer matches the reviewed bytes", asset.Path)
		}
		if !media.matches(data) {
			return "", nil, fmt.Errorf("visual asset %s is no longer a %s file", asset.Path, asset.MediaType)
		}
	}
	plan := VisualPlan{SchemaVersion: visualSchemaVersion, SourceRevision: manifest.SourceProse.SHA256, Opportunities: manifest.Actions}
	if err := validatePlanReport(string(planReport), plan); err != nil {
		return "", nil, err
	}
	if err := validateAssembledCandidate(article, prose, plan, manifest.Assets); err != nil {
		return "", nil, err
	}
	revision := candidateRevision(article, manifest.Assets)
	if revision != manifest.ReviewedRevision {
		return "", nil, fmt.Errorf("visual manifest for candidate %03d records revision %q, but its bound bytes hash to %q",
			candidate, manifest.ReviewedRevision, revision)
	}
	return revision, article, nil
}
