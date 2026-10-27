package token

import "testing"

// TestIsType revisa qué tokens cuentan como nombre de tipo. El parser lo
// usa para saber si una sentencia es una declaración.
func TestIsType(t *testing.T) {
	tests := []struct {
		t    TokenType
		want bool
	}{
		{INT_TYPE, true},
		{DOUBLE_TYPE, true},
		{BOOL_TYPE, true},
		{STRING_TYPE, true},
		{VOID, true},
		{CONST, false}, // const acompaña a un tipo, pero no es un tipo
		{IDENT, false},
		{INT, false}, // INT es un número (42), no el tipo int
	}
	for _, tt := range tests {
		t.Run(string(tt.t), func(t *testing.T) {
			if got := IsType(tt.t); got != tt.want {
				t.Fatalf("IsType(%s) = %v, se esperaba %v", tt.t, got, tt.want)
			}
		})
	}
}
