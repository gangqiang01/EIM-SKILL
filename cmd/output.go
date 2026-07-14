package cmd

import (
	"encoding/json"
	"fmt"
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
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		// Marshaling our own envelope failed - fall back to plain text so
		// the caller still gets something rather than nothing.
		fmt.Fprintf(os.Stderr, "eim-skill: failed to render output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
