package webui

import (
	"github.com/l0caldadmin/LocalAGI/core/agent"
	"github.com/l0caldadmin/LocalAGI/core/state"
	"github.com/l0caldadmin/LocalAGI/webui/collections"
)

// CollectionsRAGProviderFromState delegates to the collections sub-package.
func CollectionsRAGProviderFromState(cs *CollectionsState) func(collectionName string) (agent.RAGDB, state.KBCompactionClient, bool) {
	return collections.RAGProviderFromState(cs)
}

// CollectionsRAGProvider returns a provider that the pool can use when no LocalRAG URL is set.
func (app *App) CollectionsRAGProvider() func(collectionName string) (agent.RAGDB, state.KBCompactionClient, bool) {
	return CollectionsRAGProviderFromState(app.collectionsState)
}
