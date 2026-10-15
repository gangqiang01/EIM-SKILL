package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run eim-skill as an MCP server over stdio",
	Long: `Starts eim-skill as a Model Context Protocol (MCP) server that talks
JSON-RPC over stdio. Every eim-skill command is exposed as an MCP tool with
the same name (command path joined by "_", e.g. "device_list"), the same
positional arguments and flags, and returns the same unified JSON envelope
({"status":200,"msg":"","data":...} or {"status":500,"msg":"..."}) as its
text content - so an MCP-compatible agent can drive eim-skill directly
instead of shelling out to the CLI.`,
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPServer()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

// argSpec describes one positional argument parsed out of a command's Use
// string (e.g. "<endpoint-or-name>" or "[id...]").
type argSpec struct {
	name     string
	required bool
	variadic bool
}

// flagSpec describes one non-inherited flag on a command.
type flagSpec struct {
	name     string
	typ      string // pflag.Value.Type(): "string", "bool", "int", "int64", "float64"
	required bool
	usage    string
}

// toolSpec is everything needed to expose one leaf cobra command as an MCP
// tool: how to build its input schema, and how to turn a tool call's input
// map back into an argv slice for rootCmd.
type toolSpec struct {
	name        string
	description string
	cmdPath     []string
	args        []argSpec
	flags       []flagSpec
	schema      *jsonschema.Schema
}

// argTokenPattern matches "<name>" or "[name]" Use-string tokens.
var argTokenPattern = regexp.MustCompile(`^<([^>]+)>$|^\[([^\]]+)\]$`)

// parsePositionalArgs extracts positional argument specs from a command's
// Use string, stopping at the first token that isn't a bracketed arg
// placeholder (some Use strings document a flag inline, e.g.
// "uninstall <a> <b> <c> --tool <tool>" - "--tool"/"<tool>" there are the
// --tool flag's own usage text, not a 4th positional argument).
func parsePositionalArgs(use string) []argSpec {
	fields := strings.Fields(use)
	if len(fields) <= 1 {
		return nil
	}

	var specs []argSpec
	for _, tok := range fields[1:] {
		m := argTokenPattern.FindStringSubmatch(tok)
		if m == nil {
			break
		}
		required := m[1] != ""
		raw := m[1]
		if raw == "" {
			raw = m[2]
		}
		variadic := strings.HasSuffix(raw, "...")
		name := strings.TrimSuffix(raw, "...")
		specs = append(specs, argSpec{name: name, required: required, variadic: variadic})
	}
	return specs
}

// flagJSONType maps a pflag value type to a JSON Schema type.
func flagJSONType(pflagType string) string {
	switch pflagType {
	case "bool":
		return "boolean"
	case "int", "int64":
		return "integer"
	case "float64":
		return "number"
	default:
		return "string"
	}
}

func isFlagRequired(f *pflag.Flag) bool {
	anns, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
	return ok && len(anns) > 0 && anns[0] == "true"
}

func buildToolSpec(cmd *cobra.Command, path []string) *toolSpec {
	desc := strings.TrimSpace(cmd.Short)
	if desc == "" {
		desc = strings.TrimSpace(cmd.Long)
	}

	argSpecs := parsePositionalArgs(cmd.Use)

	var flagSpecs []flagSpec
	cmd.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" {
			return
		}
		flagSpecs = append(flagSpecs, flagSpec{
			name:     f.Name,
			typ:      f.Value.Type(),
			required: isFlagRequired(f),
			usage:    f.Usage,
		})
	})

	properties := map[string]*jsonschema.Schema{}
	var required []string

	for _, a := range argSpecs {
		if a.variadic {
			properties[a.name] = &jsonschema.Schema{
				Type:        "array",
				Items:       &jsonschema.Schema{Type: "string"},
				Description: fmt.Sprintf("positional argument %q (repeatable)", a.name),
			}
		} else {
			properties[a.name] = &jsonschema.Schema{
				Type:        "string",
				Description: fmt.Sprintf("positional argument %q", a.name),
			}
		}
		if a.required {
			required = append(required, a.name)
		}
	}

	for _, f := range flagSpecs {
		properties[f.name] = &jsonschema.Schema{
			Type:        flagJSONType(f.typ),
			Description: f.usage,
		}
		if f.required {
			required = append(required, f.name)
		}
	}

	return &toolSpec{
		name:        strings.Join(path, "_"),
		description: desc,
		cmdPath:     append([]string{}, path...),
		args:        argSpecs,
		flags:       flagSpecs,
		schema: &jsonschema.Schema{
			Type:       "object",
			Properties: properties,
			Required:   required,
		},
	}
}

// collectToolSpecs walks the command tree under cmd, returning one toolSpec
// per runnable (leaf) command. Group commands (e.g. "device", "ota android")
// that exist only to hold subcommands are not runnable and are skipped.
func collectToolSpecs(cmd *cobra.Command, path []string) []*toolSpec {
	var specs []*toolSpec
	for _, c := range cmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		childPath := append(append([]string{}, path...), c.Name())
		if c.Runnable() {
			specs = append(specs, buildToolSpec(c, childPath))
		}
		if c.HasSubCommands() {
			specs = append(specs, collectToolSpecs(c, childPath)...)
		}
	}
	return specs
}

// toArgString renders a decoded-JSON value as a CLI argument/flag-value
// string. Numbers come from encoding/json as float64; integer-valued floats
// are rendered without a trailing ".0" so they parse cleanly as --int flags.
func toArgString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func toStringSlice(v any) ([]string, error) {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, toArgString(item))
		}
		return out, nil
	case string:
		return []string{t}, nil
	default:
		return nil, fmt.Errorf("expected an array of strings, got %T", v)
	}
}

// buildArgs reconstructs a rootCmd-compatible argv slice from a tool call's
// decoded JSON input, in the order: command path, positional args (in Use
// order), then flags as --name=value.
func buildArgs(spec *toolSpec, input map[string]any) ([]string, error) {
	args := append([]string{}, spec.cmdPath...)

	for _, a := range spec.args {
		v, ok := input[a.name]
		if !ok || v == nil {
			if a.required {
				return nil, fmt.Errorf("missing required argument %q", a.name)
			}
			continue
		}
		if a.variadic {
			items, err := toStringSlice(v)
			if err != nil {
				return nil, fmt.Errorf("argument %q: %w", a.name, err)
			}
			args = append(args, items...)
		} else {
			args = append(args, toArgString(v))
		}
	}

	for _, f := range spec.flags {
		v, ok := input[f.name]
		if !ok || v == nil {
			if f.required {
				return nil, fmt.Errorf("missing required flag %q", f.name)
			}
			continue
		}
		args = append(args, fmt.Sprintf("--%s=%s", f.name, toArgString(v)))
	}

	return args, nil
}

// resetCommandTree resets every changed flag (local and persistent) across
// the whole command tree back to its default. rootCmd and its subcommands
// are package-level singletons reused across every MCP tool call, so without
// this a flag set on one call (e.g. --group=foo) would silently leak into
// the next call that omits it.
type flagState struct {
	value   string
	changed bool
}

func snapshotCommandTree(cmd *cobra.Command) map[string]flagState {
	snapshot := make(map[string]flagState)
	take := func(fs *pflag.FlagSet, cmdName string) {
		fs.VisitAll(func(f *pflag.Flag) {
			snapshot[cmdName+"::"+f.Name] = flagState{
				value:   f.Value.String(),
				changed: f.Changed,
			}
		})
	}

	take(cmd.Flags(), cmd.CommandPath())
	take(cmd.PersistentFlags(), cmd.CommandPath())
	for _, c := range cmd.Commands() {
		for k, v := range snapshotCommandTree(c) {
			snapshot[k] = v
		}
	}
	return snapshot
}

func restoreCommandTree(cmd *cobra.Command, snapshot map[string]flagState) {
	restore := func(fs *pflag.FlagSet, cmdName string) {
		fs.VisitAll(func(f *pflag.Flag) {
			state, ok := snapshot[cmdName+"::"+f.Name]
			if !ok {
				return
			}
			_ = f.Value.Set(state.value)
			f.Changed = state.changed
		})
	}
	restore(cmd.Flags(), cmd.CommandPath())
	restore(cmd.PersistentFlags(), cmd.CommandPath())
	for _, c := range cmd.Commands() {
		restoreCommandTree(c, snapshot)
	}
}

func resetCommandTree(cmd *cobra.Command) {
	reset := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			if f.Changed {
				_ = f.Value.Set(f.DefValue)
				f.Changed = false
			}
		})
	}
	reset(cmd.Flags())
	reset(cmd.PersistentFlags())
	for _, c := range cmd.Commands() {
		resetCommandTree(c)
	}
}

// envelopeStatus is just enough of the output.go result envelope to tell
// whether a captured command output represents success or failure.
type envelopeStatus struct {
	Status int `json:"status"`
}

func isErrorEnvelope(capturedJSON string) bool {
	var s envelopeStatus
	if err := json.Unmarshal([]byte(capturedJSON), &s); err != nil {
		return false
	}
	return s.Status != statusOK
}

// invokeCommand runs spec's command against input as if it were invoked
// from the CLI, returning its captured JSON envelope output. Not safe for
// concurrent use - callers must hold the server-wide invocation mutex (see
// runMCPServer), since it mutates the shared rootCmd flag state and the
// package-level output writer.
func invokeCommand(spec *toolSpec, input map[string]any) string {
	args, err := buildArgs(spec, input)
	if err != nil {
		return withCapturedOutput(func() { printErrorResult(err) })
	}

	startup := snapshotCommandTree(rootCmd)
	resetCommandTree(rootCmd)
	restoreCommandTree(rootCmd, startup)
	rootCmd.SetArgs(args)

	result := withCapturedOutput(func() {
		if err := rootCmd.Execute(); err != nil {
			printErrorResult(err)
		}
	})

	resetCommandTree(rootCmd)
	restoreCommandTree(rootCmd, startup)
	return result
}

func runMCPServer() error {
	server := mcp.NewServer(&mcp.Implementation{Name: "eim-skill", Version: rootCmd.Version}, nil)

	specs := collectToolSpecs(rootCmd, nil)

	var mu sync.Mutex
	for _, spec := range specs {
		mcp.AddTool(server, &mcp.Tool{
			Name:        spec.name,
			Description: spec.description,
			InputSchema: spec.schema,
		}, func(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
			mu.Lock()
			defer mu.Unlock()

			text := invokeCommand(spec, input)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
				IsError: isErrorEnvelope(text),
			}, nil, nil
		})
	}

	return server.Run(context.Background(), &mcp.StdioTransport{})
}
