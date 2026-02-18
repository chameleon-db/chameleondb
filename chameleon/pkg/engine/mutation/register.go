package mutation

import "github.com/chameleon-db/chameleondb/chameleon/pkg/engine"

// Auto-register the mutation factory on package import
// This happens automatically when mutation package is imported anywhere
func init() {
	engine.RegisterMutationFactory(NewFactory())
}
