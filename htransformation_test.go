package htransformation_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	plug "github.com/tomMoulard/htransformation"
	"github.com/tomMoulard/htransformation/pkg/tests/assert"
	"github.com/tomMoulard/htransformation/pkg/tests/require"
	"github.com/tomMoulard/htransformation/pkg/types"
)

func TestValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		config  *plug.Config
		wantErr bool
	}{
		{
			name:    "no rules",
			config:  &plug.Config{},
			wantErr: false,
		},
		{
			name: "no rules type",
			config: &plug.Config{
				Rules: []types.Rule{
					{
						Name: "no rule",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid rules type",
			config: &plug.Config{
				Rules: []types.Rule{
					{
						Name: "invalid rule",
						Type: "THIS IS NOT A VALID RULE TYPE",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing type",
			config: &plug.Config{
				Rules: []types.Rule{
					{
						Name:   "rule with no type",
						Header: "not-empty",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid rule",
			config: &plug.Config{
				Rules: []types.Rule{
					{
						Name:   "add rule",
						Header: "not-empty",
						Value:  "not-empty",
						Type:   types.Add,
					},
					{
						Name:   "delete rule",
						Header: "not-empty",
						Type:   types.Delete,
					},
					{
						Name:   "join Rule",
						Header: "not-empty",
						Values: []string{"not-empty"},
						Sep:    "not-empty",
						Type:   types.Join,
					},
					{
						Name:         "rewrite rule",
						Header:       "not-empty",
						ValueReplace: "not-empty",
						Value:        "not-empty",
						Type:         types.RewriteValueRule,
					},
					{
						Name:   "set rule",
						Header: "not-empty",
						Value:  "not-empty",
						Type:   types.Set,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := plug.New(t.Context(), nil, test.config, "test")
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHeaderRules(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		rule             types.Rule
		wantErr          bool
		additionalHeader map[string]string
	}{
		{
			name: "set rule",
			rule: types.Rule{
				Name:   "set rule",
				Header: "not-empty",
				Value:  "not-empty",
				Type:   types.Set,
			},
			wantErr: false,
		},
		{
			name: "rename rule",
			rule: types.Rule{
				Name:   "rename rule",
				Header: "not-empty",
				Value:  "not-empty",
				Type:   types.Rename,
			},
			additionalHeader: map[string]string{
				"Referer": "http://foo.bar",
			},
			wantErr: false,
		},
		{
			name: "rewrite rule",
			rule: types.Rule{
				Name:         "rewrite rule",
				Header:       "not-empty",
				Value:        "not-empty",
				ValueReplace: "not-empty",
				Type:         types.RewriteValueRule,
			},
			additionalHeader: map[string]string{
				"Referer": "http://foo.bar",
			},
			wantErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := plug.CreateConfig()
			cfg.Rules = []types.Rule{test.rule}

			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

			handler, err := plug.New(t.Context(), next, cfg, "demo-plugin")
			require.NoError(t, err)

			recorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost", nil)
			require.NoError(t, err)

			for key, value := range test.additionalHeader {
				req.Header.Set(key, value)
			}

			handler.ServeHTTP(recorder, req)
			result := recorder.Result()
			statusCode := result.StatusCode
			require.NoError(t, result.Body.Close())

			if test.wantErr {
				assert.Equal(t, http.StatusInternalServerError, statusCode)
			} else {
				assert.Equal(t, http.StatusOK, statusCode)
			}
		})
	}
}

func TestAddMultipleHeaders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		rules           []types.Rule
		initialHeaders  map[string][]string
		expectedHeaders map[string][]string
	}{
		{
			name: "add multiple Set-Cookie headers",
			rules: []types.Rule{
				{
					Name:          "add first cookie",
					Header:        "Set-Cookie",
					Value:         "session=abc123; Path=/; HttpOnly",
					Type:          types.Add,
					SetOnResponse: true,
				},
				{
					Name:          "add second cookie",
					Header:        "Set-Cookie",
					Value:         "user=john; Path=/; Secure",
					Type:          types.Add,
					SetOnResponse: true,
				},
			},
			initialHeaders: map[string][]string{
				"Set-Cookie": {"tracking=xyz; Path=/"},
			},
			expectedHeaders: map[string][]string{
				"Set-Cookie": {
					"tracking=xyz; Path=/",
					"session=abc123; Path=/; HttpOnly",
					"user=john; Path=/; Secure",
				},
			},
		},
		{
			name: "add multiple custom headers on request",
			rules: []types.Rule{
				{
					Name:   "add first header",
					Header: "X-Custom",
					Value:  "Value1",
					Type:   types.Add,
				},
				{
					Name:   "add second header",
					Header: "X-Custom",
					Value:  "Value2",
					Type:   types.Add,
				},
			},
			initialHeaders: map[string][]string{},
			expectedHeaders: map[string][]string{
				"X-Custom": {"Value1", "Value2"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := plug.CreateConfig()
			cfg.Rules = test.rules

			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Add initial response headers
				for headerName, headerValues := range test.initialHeaders {
					for _, headerValue := range headerValues {
						rw.Header().Add(headerName, headerValue)
					}
				}
				rw.WriteHeader(http.StatusOK)
			})

			handler, err := plug.New(t.Context(), next, cfg, "demo-plugin")
			require.NoError(t, err)

			recorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost", nil)
			require.NoError(t, err)

			handler.ServeHTTP(recorder, req)
			resp := recorder.Result()
			statusCode := resp.StatusCode
			require.NoError(t, resp.Body.Close())

			assert.Equal(t, http.StatusOK, statusCode)

			for headerName, expectedValues := range test.expectedHeaders {
				actualValues := resp.Header.Values(headerName)
				assert.Equal(t, len(expectedValues), len(actualValues))
				for i, expectedValue := range expectedValues {
					assert.Equal(t, expectedValue, actualValues[i])
				}
			}
		})
	}
}

func TestSetOnResponse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		headerName       string
		headerValue      string
		rule             types.Rule
		expectedNewValue string
	}{
		{
			name:        "set rule",
			headerName:  "Header-A",
			headerValue: "valueA",
			rule: types.Rule{
				Name:          "set rule",
				Header:        "Header-A",
				Value:         "newValue",
				Type:          types.Set,
				SetOnResponse: true,
			},
			expectedNewValue: "newValue",
		},
		{
			name:        "rewrite rule",
			headerName:  "Header-A",
			headerValue: "valueAA",
			rule: types.Rule{
				Name:          "rewrite rule",
				Header:        "Header-A",
				Value:         `value([\w\W]+)`,
				ValueReplace:  "newValue-$1",
				Type:          types.RewriteValueRule,
				SetOnResponse: true,
			},
			expectedNewValue: "newValue-AA",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cfg := plug.CreateConfig()
			cfg.Rules = []types.Rule{test.rule}

			next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.Header().Add(test.headerName, test.headerValue)
				rw.WriteHeader(http.StatusOK)
			})

			handler, err := plug.New(t.Context(), next, cfg, "demo-plugin")
			require.NoError(t, err)

			recorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost", nil)
			require.NoError(t, err)

			handler.ServeHTTP(recorder, req)
			resp := recorder.Result()
			statusCode := resp.StatusCode
			require.NoError(t, resp.Body.Close())

			assert.Equal(t, http.StatusOK, statusCode)

			assert.Equal(t, test.expectedNewValue, resp.Header.Get(test.rule.Header))
		})
	}
}
