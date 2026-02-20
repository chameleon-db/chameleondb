// engine/registry.go
package engine

var mutationFactory MutationFactory

func RegisterMutationFactory(factory MutationFactory) {
	if factory == nil {
		return
	}
	if mutationFactory == nil {
		mutationFactory = factory
	}
}

func getMutationFactory() MutationFactory {
	return mutationFactory
}
