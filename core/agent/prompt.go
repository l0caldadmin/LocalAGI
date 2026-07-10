package agent

import "github.com/l0caldadmin/LocalAGI/core/types"

type DynamicPrompt interface {
	Render(a *Agent) (types.PromptResult, error)
	Role() string
}
