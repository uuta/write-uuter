# Public Contract Review

## Summary

The CLI selection behavior and retained smoke artifacts agree with the implementation, including optional-provider resolution and `sha256:<hex>` digests. Two public-contract gaps remain: nested policy-field errors do not identify the offending role, and the artifact documentation does not fully specify digest grammar or optional-field presence.

## Findings

### 1. Unknown profile fields are reported without the offending role

- **Severity**: Medium
- **Location**: `internal/app/policy.go:71`
- **Description**: Schema-v1 validation delegates the whole document to the generic strict JSON decoder before iterating role profiles. For an unknown field inside one of the eight role objects, the resulting error names only the field (for example, `json: unknown field "fallback"`) and not the role that contains it. This makes a valid rejection non-actionable when the same profile shape is repeated across every role, and it does not meet the contract's requirement that errors identify the offending role/field.
- **Evidence**: `parseModelPolicy` calls `decodeStrictJSON(data, &document)` at lines 69-72; `decodeStrictJSON` enables `decoder.DisallowUnknownFields()` and returns the decoder error unchanged at `internal/app/strict_json.go:24-27`. The dedicated case at `internal/app/model_policy_blackbox_test.go:443-445` inserts `fallback` under `writer` but asserts only the generic substring `unknown field`, so it does not require role context. The targeted command `rtk go test ./internal/app -run 'TestBlackBox(ProviderExecutableSelectionIsIndependent|InvalidPolicyFailsBeforeRunCreation|ExplicitEmptyExecutableOverridesFailBeforeRunInitialization|EveryRoleLaunchesItsDeclaredProfile)' -count=1` passed 24 tests, confirming the current weak assertion passes.
- **Suggestion**: Decode role profiles with path-aware validation (or retain raw role objects and strictly decode each one separately), then wrap unknown/duplicate-field errors as `models.json role "writer" field "fallback" ...`; add a black-box assertion for both the role and field names.

### 2. Artifact documentation leaves digest grammar and field omission rules implicit

- **Severity**: Low
- **Location**: `docs/artifacts.md:42`
- **Description**: The public artifact documentation calls `workflow.json.model_policy_digest` a SHA-256 digest and shows `"sha256:..."` only in a reviewer example, but never defines the machine-readable grammar (`sha256:` followed by 64 lowercase hexadecimal characters). It also does not state that `lens` is present only for reviewer invocations and `candidate` is present only for Writer/reviewer invocations. Consumers therefore cannot distinguish intentional omission from an incomplete audit record using the documented schema alone.
- **Evidence**: The implementation uses `omitempty` for `lens` and `candidate` at `internal/app/models.go:76-77`, while `publishInvocationAudit` supplies zero/empty values at `internal/app/app.go:1618-1621`. In `smoke-runs/mixed-20260824-1/.control/invocations/`, `001-pm.json`, `002-researcher.json`, and `003-story-editor.json` omit both fields, while `004-writer.json` contains `candidate: 1` and omits `lens`. `rtk shasum -a 256 prompts/models.json smoke-runs/mixed-20260824-1/model-policy.json` produced the same raw hash, `f5a91c462dea2aebe51b2870213c4a193ce0c1576f95ec983b018ae30b16752b`, and `workflow.json` plus every retained invocation record store it as `sha256:f5a91c462dea2aebe51b2870213c4a193ce0c1576f95ec983b018ae30b16752b`.
- **Suggestion**: Document the exact digest regex (for example, `^sha256:[0-9a-f]{64}$`) and add a field table stating the required/conditional presence of `lens` and `candidate` for PM, Researcher, Story Editor, Writer, and reviewer records.
