package weave

// Compiler validates and compiles adapter-bound Predicate values. A Compiler
// must be request-stateless and safe for concurrent use. It may retain
// immutable semantic configuration and read-only field metadata, but it must
// not retain per-request state, database handles, sessions, contexts, loggers,
// or transactions. Direct Compile calls bypass Factory identity, structure,
// capability, and error safeguards, so implementations must still validate
// their own adapter contracts defensively.
type Compiler[C, E any] interface {
	// Compile validates predicate for the adapter and emits a backend condition.
	Compile(Predicate[C, E]) (C, error)
	// Capabilities returns the compiler's stable capability commitment.
	Capabilities() Capabilities
}
