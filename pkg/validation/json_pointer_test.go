package validation

import "testing"

func TestJSONPointer(t *testing.T) {
	for _, test := range []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "root", want: ""},
		{name: "tokens", parts: []string{"dependencies", "0", "module"}, want: "/dependencies/0/module"},
		{name: "escaping", parts: []string{"a/b", "~value"}, want: "/a~1b/~0value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := JSONPointer(test.parts...); got != test.want {
				t.Fatalf("JSONPointer() = %q, want %q", got, test.want)
			}
		})
	}
}
