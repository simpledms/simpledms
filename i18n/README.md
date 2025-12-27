`gotext update` seems to be the command that executes all necessary extraction and generation steps. `gotext extract` and `gotext generate` have a smaller scope.

Can be easily investigated in the source code (https://cs.opensource.google/go/x/text/+/refs/tags/v0.24.0:cmd/gotext/update.go).
