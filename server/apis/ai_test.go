package apis

import (
	"net/http"
	"strings"
	"testing"
)

func TestMapOpenAIError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		assertFn   func(t *testing.T, got string)
	}{
		{
			name:       "missing model request scope gets actionable message",
			statusCode: http.StatusForbidden,
			body:       `{"error":{"message":"You have insufficient permissions for this operation. Missing scopes: model.request."}}`,
			assertFn: func(t *testing.T, got string) {
				if !strings.Contains(got, "model.request") {
					t.Fatalf("expected model.request guidance, got %q", got)
				}
				if !strings.Contains(got, "Reconnect AI") {
					t.Fatalf("expected reconnect guidance, got %q", got)
				}
			},
		},
		{
			name:       "unauthorized errors map to auth guidance",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"message":"Invalid authentication credentials"}}`,
			assertFn: func(t *testing.T, got string) {
				want := "AI authentication failed. Reconnect AI in Settings or update your API key."
				if got != want {
					t.Fatalf("expected %q, got %q", want, got)
				}
			},
		},
		{
			name:       "provider error message remains visible",
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"message":"Model 'gpt-4o' not found"}}`,
			assertFn: func(t *testing.T, got string) {
				want := "AI service error: Model 'gpt-4o' not found"
				if got != want {
					t.Fatalf("expected %q, got %q", want, got)
				}
			},
		},
		{
			name:       "raw non json payload falls back to trimmed text",
			statusCode: http.StatusBadGateway,
			body:       " upstream timeout ",
			assertFn: func(t *testing.T, got string) {
				want := "AI service error: upstream timeout"
				if got != want {
					t.Fatalf("expected %q, got %q", want, got)
				}
			},
		},
		{
			name:       "empty unauthorized payload returns auth default",
			statusCode: http.StatusUnauthorized,
			body:       "   ",
			assertFn: func(t *testing.T, got string) {
				want := "AI authentication failed. Reconnect AI in Settings or update your API key."
				if got != want {
					t.Fatalf("expected %q, got %q", want, got)
				}
			},
		},
		{
			name:       "empty non auth payload returns generic default",
			statusCode: http.StatusInternalServerError,
			body:       "",
			assertFn: func(t *testing.T, got string) {
				want := "AI service error"
				if got != want {
					t.Fatalf("expected %q, got %q", want, got)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapOpenAIError(tc.statusCode, []byte(tc.body))
			tc.assertFn(t, got)
		})
	}
}
