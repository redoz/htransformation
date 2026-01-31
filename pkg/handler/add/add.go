package add

import (
	"net/http"
	"strings"

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
	value := getValue(a.rule.Value, a.rule.HeaderPrefix, req)

	if a.rule.SetOnResponse {
		rw.Header().Add(a.rule.Header, value)

		return
	}

	header.Add(req, a.rule.Header, value)
}

// getValue checks if prefix exists, the given prefix is present,
// and then proceeds to read the existing header (after stripping the prefix)
// to return as value.
func getValue(ruleValue, valueIsHeaderPrefix string, req *http.Request) string {
	actualValue := ruleValue

	if valueIsHeaderPrefix != "" && strings.HasPrefix(ruleValue, valueIsHeaderPrefix) {
		header := strings.TrimPrefix(ruleValue, valueIsHeaderPrefix)
		// If the resulting value after removing the prefix is empty,
		// we return the actual value,
		// which is the prefix itself.
		// This is because doing a req.Header.Get("") would not fly well.
		if header == "" {
			return actualValue
		}

		if strings.EqualFold(header, "Host") {
			actualValue = req.Host
		} else {
			actualValue = req.Header.Get(header)
		}
	}

	return actualValue
}
