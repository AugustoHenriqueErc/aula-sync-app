package main

import "testing"

func TestServerPort(t *testing.T) {
	for _, tt := range []struct{ value, want string }{
		{"", "3000"}, {"3001", "3001"}, {"1", "1"}, {"65535", "65535"},
	} {
		t.Run("válida_"+tt.value, func(t *testing.T) {
			got, err := serverPort(tt.value)
			if err != nil || got != tt.want {
				t.Fatalf("serverPort(%q) = %q, %v; esperado %q", tt.value, got, err, tt.want)
			}
		})
	}
	for _, value := range []string{"0", "-1", "65536", "abc", "3000.5", " "} {
		t.Run("inválida_"+value, func(t *testing.T) {
			if _, err := serverPort(value); err == nil {
				t.Fatalf("PORT=%q deveria ser rejeitada", value)
			}
		})
	}
}
