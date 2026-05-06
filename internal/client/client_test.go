package client

import "testing"

func TestEndpointURLAddsCommandPathToBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		endpoint string
		want     string
	}{
		{name: "connect base", rawURL: "http://node-a:8080", endpoint: "connect", want: "http://node-a:8080/connect"},
		{name: "connect slash", rawURL: "http://node-a:8080/", endpoint: "connect", want: "http://node-a:8080/connect"},
		{name: "connect existing", rawURL: "http://node-a:8080/connect", endpoint: "connect", want: "http://node-a:8080/connect"},
		{name: "disconnect base", rawURL: "http://node-a:8080", endpoint: "disconnect", want: "http://node-a:8080/disconnect"},
		{name: "peers base", rawURL: "http://node-a:8080", endpoint: "peers", want: "http://node-a:8080/peers"},
		{name: "prefix path", rawURL: "http://node-a:8080/api", endpoint: "connect", want: "http://node-a:8080/api/connect"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := endpointURL(tt.rawURL, tt.endpoint); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
