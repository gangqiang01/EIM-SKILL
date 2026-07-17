package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// result is the single JSON envelope every eim-skill command prints to
// stdout: {"status":200,"msg":"","data":...} on success, or
// {"status":500,"msg":"..."} on failure. Scripts/agents can always parse
// stdout as JSON and branch on "status" rather than having to special-case
// each command's output shape or watch stderr separately.
type result struct {
	Status int         `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data,omitempty"`
}

const (
	statusOK    = 200
	statusError = 500
)

// out is where printJSON/printErrorResult write. It defaults to the real
// stdout for normal CLI use; the MCP server (see mcp.go) temporarily
// swaps it per tool call via withCapturedOutput so it can return each
// command's JSON as the tool result instead of writing to the process's
// actual stdout (which the MCP stdio transport itself is using for the
// JSON-RPC channel).
var out io.Writer = os.Stdout

// withCapturedOutput runs fn with package output redirected to a buffer,
// returning what was written. Not safe for concurrent use - callers must
// serialize invocations (see mcp.go's invocation mutex).
func withCapturedOutput(fn func()) string {
	var buf bytes.Buffer
	prev := out
	out = &buf
	defer func() { out = prev }()
	fn()
	return buf.String()
}

// printJSON prints data wrapped in the unified success envelope.
func printJSON(data interface{}) {
	printEnvelope(result{Status: statusOK, Msg: "", Data: data})
}

// printRawJSON prints an already-encoded JSON payload (e.g. a
// json.RawMessage passed straight through from the server), wrapped in the
// unified success envelope, for responses we deliberately don't model with
// a Go struct.
func printRawJSON(raw json.RawMessage) {
	if len(raw) == 0 {
		printJSON(nil)
		return
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		// Not valid JSON (shouldn't happen) - fall back to the raw string
		// rather than losing the payload.
		printJSON(string(raw))
		return
	}
	printJSON(v)
}

// printErrorResult prints err wrapped in the unified failure envelope.
func printErrorResult(err error) {
	printEnvelope(result{Status: statusError, Msg: err.Error()})
}

func printEnvelope(r result) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// json.Marshal/MarshalIndent HTML-escape "<", ">", and "&" by default
	// (meant for embedding JSON in HTML - not relevant here, and it mangles
	// legitimate output like a usage string containing "<endpoint-or-name>"
	// into "<endpoint-or-name>"). Encoder lets us turn that off.
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		// Marshaling our own envelope failed (e.g. a NaN float somewhere in
		// Data) - write a hand-built fallback rather than exiting the
		// process, since this now also runs inside the long-running MCP
		// server (see mcp.go), where os.Exit would kill every in-flight
		// tool call, not just this one.
		fmt.Fprintf(out, "{\"status\":%d,\"msg\":%q}\n", statusError, "failed to render output: "+err.Error())
		return
	}
	fmt.Fprint(out, buf.String())
}
