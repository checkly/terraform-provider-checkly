package provider

import (
	"errors"
	"testing"
)

func TestIsNotFoundError(t *testing.T) {
	cases := map[string]struct {
		err  error
		want bool
	}{
		"nil": {nil, false},
		"404 response": {
			errors.New(`unexpected response status 404: "{\"statusCode\":404}"`), true,
		},
		"404 response in the alternative format": {
			errors.New(`unexpected response status: 404, res: "{\"statusCode\":404}"`), true,
		},
		// The 400 body quotes component names, which may contain "404".
		"400 response quoting 404": {
			errors.New(`unexpected response status 400: "\"API 404\" is not a GROUP component."`), false,
		},
		"400 response quoting the 404 prefix": {
			errors.New(`unexpected response status 400: "\"unexpected response status 404: x\" is not a GROUP component"`), false,
		},
		"500 response for an ID containing 404": {
			errors.New(`unexpected response status 500: "internal error for check 1404"`), false,
		},
		"transport error for a URL containing 404": {
			errors.New(`HTTP request failed with: Delete "https://api.checklyhq.com/v1/variables/PAGE_404_URL": context deadline exceeded`), false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := isNotFoundError(c.err); got != c.want {
				t.Errorf("isNotFoundError(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
