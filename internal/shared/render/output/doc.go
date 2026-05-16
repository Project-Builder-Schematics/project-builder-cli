// Package output declares the Output port and its adapter convention.
//
// # Port + Adapter Layout
//
// The Output port follows the same pattern as the Renderer port (ADR-01):
//
//   - output/output.go       — the port (interface) + Call struct
//   - output/themed/         — production adapter (charmbracelet/lipgloss)
//   - output/outputtest/     — test peer (spy recording ordered calls)
//
// Feature handlers accept output.Output as a constructor dependency; they never
// reference themed.Adapter or outputtest.Spy directly. composeApp is the only
// site that instantiates themed.Adapter.
//
// # Prompt vs Stream
//
// Two distinct flows exist for user-interactive output:
//
//   - Output.Prompt(text) — synchronous; blocks until the user types a reply.
//     Use for simple CLI questions (e.g., "What is the MCP server name?").
//
//   - Output.Stream(ctx, ch) — asynchronous; consumes an events.Event channel.
//     Engine-driven prompts (e.g., schematic InputRequested events) arrive here.
//
// Never conflate the two: Prompt is for simple Q&A; Stream is for the engine
// event protocol. Using Prompt inside a Stream handler would deadlock.
package output
