package schema

import "encoding/json"

// SchemaVersion is the only suite document version this mcpdev reads and writes.
const SchemaVersion = 1

// Suite is one reviewable YAML or JSON test file.
// Secret values are never stored here; only environment variable names.
type Suite struct {
	Variables     map[string]string `json:"variables,omitzero"`
	Secrets       map[string]Secret `json:"secrets,omitzero"`
	Target        Target            `json:"target"`
	Defaults      Defaults          `json:"defaults,omitzero"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitzero"`
	Protocol      string            `json:"protocol,omitzero"`
	Setup         []Hook            `json:"setup,omitzero"`
	Cleanup       []Hook            `json:"cleanup,omitzero"`
	Cases         []Case            `json:"cases"`
	SchemaVersion int               `json:"schema_version"`
}

// Target selects exactly one connection. Command is an argv array, never a shell line.
type Target struct {
	Profile string   `json:"profile,omitzero"`
	URL     string   `json:"url,omitzero"`
	Command []string `json:"command,omitzero"`
}

// Defaults apply when a case omits a field. Durations are Go strings such as "10s".
type Defaults struct {
	Timeout        string `json:"timeout,omitzero"`
	ConnectTimeout string `json:"connect_timeout,omitzero"`
}

// Secret names an environment variable. There is no value field.
type Secret struct {
	FromEnv string `json:"from_env"`
}

// Hook is a narrow pre/post action. Exec is argv only (same rule as profile command).
type Hook struct {
	Snapshot *HookSnapshot `json:"snapshot,omitzero"`
	Exec     *HookExec     `json:"exec,omitzero"`
}

// HookSnapshot names a baseline file the later snapshot step may compare.
type HookSnapshot struct {
	Path string `json:"path"`
	As   string `json:"as,omitzero"`
}

// HookExec runs a local argv before connect or after the suite. Not a shell string.
type HookExec struct {
	CWD  string   `json:"cwd,omitzero"`
	Argv []string `json:"argv"`
}

// Case is an ordered list of steps with an optional timeout override.
type Case struct {
	Name    string `json:"name"`
	Timeout string `json:"timeout,omitzero"`
	Steps   []Step `json:"steps"`
}

// Step is exactly one MCP operation plus optional captures, assertions, and expected failure.
type Step struct {
	ID            string         `json:"id,omitzero"`
	Inspect       *InspectOp     `json:"inspect,omitzero"`
	Call          *CallOp        `json:"call,omitzero"`
	Prompt        *PromptOp      `json:"prompt,omitzero"`
	Read          *ReadOp        `json:"read,omitzero"`
	Snapshot      *SnapshotOp    `json:"snapshot,omitzero"`
	ExpectFailure *ExpectFailure `json:"expect_failure,omitzero"`
	Capture       []Capture      `json:"capture,omitzero"`
	Assert        []Assertion    `json:"assert,omitzero"`
}

// InspectOp lists tools, prompts, resources, templates, or all (empty Kind).
type InspectOp struct {
	Kind string `json:"kind,omitzero"`
}

// CallOp is tools/call. Arguments are opaque JSON (may interpolate ${var}).
// Keys such as command are tool parameters, not target.command.
type CallOp struct {
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments,omitzero"`
}

// PromptOp is prompts/get. Argument values are strings.
type PromptOp struct {
	Arguments map[string]string `json:"arguments,omitzero"`
	Name      string            `json:"name"`
}

// ReadOp is resources/read, optionally expanding a URI template first.
type ReadOp struct {
	Arguments map[string]string `json:"arguments,omitzero"`
	URI       string            `json:"uri,omitzero"`
	Template  string            `json:"template,omitzero"`
}

// SnapshotOp compares the live public interface to a baseline file.
type SnapshotOp struct {
	Compare string   `json:"compare"`
	FailOn  string   `json:"fail_on,omitzero"`
	Ignore  []string `json:"ignore,omitzero"`
	Rename  []string `json:"rename,omitzero"`
}

// ExpectFailure marks the step as required to fail. Category is an errclass name or tool_error.
type ExpectFailure struct {
	Category string `json:"category"`
}

// Capture binds a JSON Path from this step's structured result to a case-local variable.
type Capture struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Assertion is exactly one check. A missing path at run time is an error, not a pass.
type Assertion struct {
	Equal         *EqualAssert         `json:"equal,omitzero"`
	Contains      *ContainsAssert      `json:"contains,omitzero"`
	JSONPath      *JSONPathAssert      `json:"json_path,omitzero"`
	Schema        *SchemaAssert        `json:"schema,omitzero"`
	ContentType   *ContentTypeAssert   `json:"content_type,omitzero"`
	Latency       *LatencyAssert       `json:"latency,omitzero"`
	ErrorCategory *ErrorCategoryAssert `json:"error_category,omitzero"`
}

// EqualAssert compares a JSON Path to a literal or to a previously captured variable.
type EqualAssert struct {
	Path     string `json:"path,omitzero"`
	Value    any    `json:"value,omitzero"`
	Variable string `json:"variable,omitzero"`
}

// ContainsAssert requires the path value (string or array) to contain Value.
type ContainsAssert struct {
	Value any    `json:"value"`
	Path  string `json:"path"`
}

// JSONPathAssert checks existence or match count at a JSON Path.
type JSONPathAssert struct {
	Exists *bool  `json:"exists,omitzero"`
	Count  *int   `json:"count,omitzero"`
	Path   string `json:"path"`
}

// SchemaAssert validates the path value against inline JSON Schema or a file.
type SchemaAssert struct {
	Path   string          `json:"path,omitzero"`
	File   string          `json:"file,omitzero"`
	Schema json.RawMessage `json:"schema,omitzero"`
}

// ContentTypeAssert checks MIME type of a resource body or content block.
type ContentTypeAssert struct {
	Equals string `json:"equals"`
}

// LatencyAssert is a ceiling on step wall time (Go duration string).
type LatencyAssert struct {
	Max string `json:"max"`
}

// ErrorCategoryAssert checks the classified failure (same vocabulary as expect_failure).
type ErrorCategoryAssert struct {
	Equals string `json:"equals"`
}
