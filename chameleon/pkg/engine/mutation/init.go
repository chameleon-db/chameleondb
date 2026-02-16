package mutation

import "github.com/chameleon-db/chameleondb/chameleon/pkg/engine"

// InitFactory initializes the default SQL mutation factory for an engine
func InitFactory(eng *engine.Engine) {
	factory := NewFactory(eng.GetSchema())
	eng.SetMutationFactory(factory)
}
