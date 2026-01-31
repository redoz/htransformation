package deleter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tomMoulard/htransformation/pkg/handler/deleter"
	"github.com/tomMoulard/htransformation/pkg/tests/assert"
	"github.com/tomMoulard/htransformation/pkg/tests/require"
	"github.com/tomMoulard/htransformation/pkg/types"
)

func TestDeleteHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		rule            types.Rule
		requestHeaders  map[string]string
		expectedHeaders map[string]string
		expectedHost    string
	}{
		{
			name: "Remove not existing header",
			rule: types.Rule{
				Header: "X-Test",
			},
			requestHeaders: map[string]string{
				"Foo": "Bar",
			},
			expectedHeaders: map[string]string{
				"Foo": "Bar",
			},
			expectedHost: "example.com",
		},
		{
			name: "Remove one header",
			rule: types.Rule{
				Header: "X-Test",
			},
			requestHeaders: map[string]string{
				"Foo":    "Bar",
				"X-Test": "Bar",
			},
			expectedHeaders: map[string]string{
				"Foo": "Bar",
			},
			expectedHost: "example.com",
		},
		{
			name: "Remove host header",
			rule: types.Rule{
				Header: "Host",
			},
			expectedHost: "",
		},
		{
			name: "Remove all headers with same name",
			rule: types.Rule{
				Header: "X-Multiple",
			},
			requestHeaders: map[string]string{
				"Foo":        "Bar",
				"X-Multiple": "Value1",
			},
			expectedHeaders: map[string]string{
				"Foo": "Bar",
			},
			expectedHost: "example.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/foo", nil)
			require.NoError(t, err)

			for hName, hVal := range test.requestHeaders {
				req.Header.Add(hName, hVal)
			}

			// For the "Remove all headers with same name" test, add multiple values
			if test.name == "Remove all headers with same name" {
				req.Header.Add("X-Multiple", "Value2")
				req.Header.Add("X-Multiple", "Value3")
			}

			deleteHandler, err := deleter.New(test.rule)
			require.NoError(t, err)

			deleteHandler.Handle(nil, req)

			for hName, hVal := range test.expectedHeaders {
				assert.Equal(t, hVal, req.Header.Get(hName))
			}

			// For the "Remove all headers with same name" test, verify header is completely gone
			if test.name == "Remove all headers with same name" {
				assert.Equal(t, "", req.Header.Get("X-Multiple"))
				assert.Equal(t, 0, len(req.Header.Values("X-Multiple")))
			}

			assert.Equal(t, test.expectedHost, req.Host)
			assert.Equal(t, "example.com", req.URL.Host)
		})
	}
}

func TestDeleteHandlerOnResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		rule           types.Rule
		requestHeaders map[string]string
		want           map[string]string
	}{
		{
			name: "Remove not existing header",
			rule: types.Rule{
				Header:        "X-Test",
				SetOnResponse: true,
			},
			requestHeaders: map[string]string{
				"Foo": "Bar",
			},
			want: map[string]string{
				"Foo": "Bar",
			},
		},
		{
			name: "Remove one header",
			rule: types.Rule{
				Header:        "X-Test",
				SetOnResponse: true,
			},
			requestHeaders: map[string]string{
				"Foo":    "Bar",
				"X-Test": "Bar",
			},
			want: map[string]string{
				"Foo": "Bar",
			},
		},
		{
			name: "Remove all Set-Cookie headers on response",
			rule: types.Rule{
				Header:        "Set-Cookie",
				SetOnResponse: true,
			},
			requestHeaders: map[string]string{
				"Foo": "Bar",
			},
			want: map[string]string{
				"Foo": "Bar",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			rw := httptest.NewRecorder()

			for hName, hVal := range test.requestHeaders {
				rw.Header().Add(hName, hVal)
			}

			// For the "Remove all Set-Cookie headers on response" test, add multiple cookies
			if test.name == "Remove all Set-Cookie headers on response" {
				rw.Header().Add("Set-Cookie", "session=abc; Path=/")
				rw.Header().Add("Set-Cookie", "user=john; Path=/")
				rw.Header().Add("Set-Cookie", "tracking=xyz; Path=/")
			}

			deleteHandler, err := deleter.New(test.rule)
			require.NoError(t, err)

			deleteHandler.Handle(rw, nil)

			for hName, hVal := range test.want {
				assert.Equal(t, hVal, rw.Header().Get(hName))
			}

			// For the "Remove all Set-Cookie headers on response" test, verify all cookies are gone
			if test.name == "Remove all Set-Cookie headers on response" {
				assert.Equal(t, "", rw.Header().Get("Set-Cookie"))
				assert.Equal(t, 0, len(rw.Header().Values("Set-Cookie")))
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
			name:    "no rules",
			wantErr: false,
		},
		{
			name: "valid rule",
			rule: types.Rule{
				Header: "not-empty",
				Type:   types.Delete,
			},
			wantErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			deleteHandler, err := deleter.New(test.rule)
			require.NoError(t, err)

			err = deleteHandler.Validate()
			t.Log(err)

			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
