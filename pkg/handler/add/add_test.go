package add_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tomMoulard/htransformation/pkg/handler/add"
	"github.com/tomMoulard/htransformation/pkg/tests/assert"
	"github.com/tomMoulard/htransformation/pkg/tests/require"
	"github.com/tomMoulard/htransformation/pkg/types"
)

func TestAddHandler(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		rule            types.Rule
		requestHeaders  map[string][]string
		expectedHeaders map[string][]string
	}{
		{
			name: "Add single header",
			rule: types.Rule{
				Header: "X-Test",
				Value:  "TestValue",
			},
			requestHeaders: map[string][]string{
				"Foo": {"Bar"},
			},
			expectedHeaders: map[string][]string{
				"Foo":    {"Bar"},
				"X-Test": {"TestValue"},
			},
		},
		{
			name: "Add header when it already exists",
			rule: types.Rule{
				Header: "X-Test",
				Value:  "NewValue",
			},
			requestHeaders: map[string][]string{
				"X-Test": {"ExistingValue"},
			},
			expectedHeaders: map[string][]string{
				"X-Test": {"ExistingValue", "NewValue"},
			},
		},
		{
			name: "Add Set-Cookie header",
			rule: types.Rule{
				Header: "Set-Cookie",
				Value:  "session=abc123; Path=/; HttpOnly",
			},
			requestHeaders: map[string][]string{
				"Set-Cookie": {"user=john; Path=/"},
			},
			expectedHeaders: map[string][]string{
				"Set-Cookie": {"user=john; Path=/", "session=abc123; Path=/; HttpOnly"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/foo", nil)
			require.NoError(t, err)

			for hName, hVals := range test.requestHeaders {
				for _, hVal := range hVals {
					req.Header.Add(hName, hVal)
				}
			}

			addHandler, err := add.New(test.rule)
			require.NoError(t, err)

			addHandler.Handle(nil, req)

			for hName, expectedVals := range test.expectedHeaders {
				actualVals := req.Header.Values(hName)
				assert.Equal(t, len(expectedVals), len(actualVals))

				for i, expectedVal := range expectedVals {
					assert.Equal(t, expectedVal, actualVals[i])
				}
			}
		})
	}
}

func TestAddHandlerOnResponse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		rule            types.Rule
		responseHeaders map[string][]string
		expectedHeaders map[string][]string
	}{
		{
			name: "Add single header on response",
			rule: types.Rule{
				Header:        "X-Custom",
				Value:         "CustomValue",
				SetOnResponse: true,
			},
			responseHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"application/json"},
				"X-Custom":     {"CustomValue"},
			},
		},
		{
			name: "Add multiple Set-Cookie headers on response",
			rule: types.Rule{
				Header:        "Set-Cookie",
				Value:         "session=xyz789; Path=/; Secure",
				SetOnResponse: true,
			},
			responseHeaders: map[string][]string{
				"Set-Cookie": {"user=jane; Path=/"},
			},
			expectedHeaders: map[string][]string{
				"Set-Cookie": {"user=jane; Path=/", "session=xyz789; Path=/; Secure"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()

			for hName, hVals := range test.responseHeaders {
				for _, hVal := range hVals {
					recorder.Header().Add(hName, hVal)
				}
			}

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/foo", nil)
			require.NoError(t, err)

			addHandler, err := add.New(test.rule)
			require.NoError(t, err)

			addHandler.Handle(recorder, req)

			for hName, expectedVals := range test.expectedHeaders {
				actualVals := recorder.Header().Values(hName)
				assert.Equal(t, len(expectedVals), len(actualVals))

				for i, expectedVal := range expectedVals {
					assert.Equal(t, expectedVal, actualVals[i])
				}
			}
		})
	}
}

func TestValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		rule    types.Rule
		wantErr bool
	}{
		{
			name:    "missing header",
			rule:    types.Rule{},
			wantErr: true,
		},
		{
			name: "missing value is OK",
			rule: types.Rule{
				Header: "X-Test",
			},
			wantErr: false,
		},
		{
			name: "valid rule",
			rule: types.Rule{
				Header: "X-Test",
				Value:  "TestValue",
			},
			wantErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			addHandler, err := add.New(test.rule)
			require.NoError(t, err)

			err = addHandler.Validate()

			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
