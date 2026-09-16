package integration

import (
	"strings"
	"testing"
)

func TestBootRefusesIncompleteConfig(t *testing.T) {
	cases := map[string]map[string]string{
		"blank api key":        {"WORKOS_API_KEY": ""},
		"blank web app url":    {"WEB_APP_URL": ""},
		"blank cookie key":     {"SESSION_COOKIE_KEY": ""},
		"malformed cookie key": {"SESSION_COOKIE_KEY": "k1:short"},
		"key without id":       {"SESSION_COOKIE_KEY": "A9Keki18RmRaac8WO5FQeo9czkzVdL2hFHzp/aDm4IQ="},
	}

	for name, overrides := range cases {
		t.Run(name, func(t *testing.T) {
			app, _, _ := buildApp(t, overrides)

			err := app.Err()
			if err == nil {
				t.Fatal("app built with incomplete configuration")
			}
			var want string
			for key := range overrides {
				want = key
			}
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not name %s", err.Error(), want)
			}
		})
	}
}
