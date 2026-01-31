package add

import (
	"net/http"

	"github.com/tomMoulard/htransformation/pkg/types"
	"github.com/tomMoulard/htransformation/pkg/utils/header"
)

type Add struct {
	rule *types.Rule
}

func New(rule types.Rule) (types.Handler, error) {
	return &Add{rule: &rule}, nil
}

func (a *Add) Validate() error {
	if a.rule.Header == "" {
		return types.ErrMissingRequiredFields
	}

	return nil
}

func (a *Add) Handle(rw http.ResponseWriter, req *http.Request) {
	if a.rule.SetOnResponse {
		rw.Header().Add(a.rule.Header, a.rule.Value)

		return
	}

	header.Add(req, a.rule.Header, a.rule.Value)
}
