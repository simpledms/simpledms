package actionx

// TxPolicy defines how the router manages the request transactions of an action.
type TxPolicy int

const (
	// TxPolicyReadOnly runs the handler in read-only transactions and streams its response.
	TxPolicyReadOnly TxPolicy = iota
	// TxPolicyReadWrite runs the handler in write transactions and buffers its response until
	// both transactions are committed, so rollbacks never release success feedback.
	TxPolicyReadWrite
	// TxPolicyManual closes the read-only authorization transactions before the handler runs.
	// The handler opens and commits its own transactions, for example around storage I/O.
	TxPolicyManual
)
