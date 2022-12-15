package exec

// Result describes the result of a run Cmd.
type Result struct {
	Command  string
	Dir      string
	Output   []byte
	ExitCode int
}

// StrOutput returns Result.Output as string.
func (r *Result) StrOutput() string {
	return string(r.Output)
}
