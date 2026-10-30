package lexer

import (
	"os"
	"polaobesahermosa/token"
	"testing"
)

// tok es una forma corta de escribir el token esperado en las tablas.
func tok(t token.TokenType, lit string, line, col int) token.Token {
	return token.Token{Type: t, Literal: lit, Line: line, Column: col}
}

// TestNextToken compara, para cada entrada, la lista EXACTA de tokens
// (tipo, literal, línea y columna) que debe producir el lexer, incluido
// el EOF final.
func TestNextToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []token.Token
	}{
		{
			name:  "declaración tipada",
			input: "int edad = 20;",
			want: []token.Token{
				tok(token.INT_TYPE, "int", 1, 1),
				tok(token.IDENT, "edad", 1, 5),
				tok(token.ASSIGN, "=", 1, 10),
				tok(token.INT, "20", 1, 12),
				tok(token.SEMICOLON, ";", 1, 14),
				tok(token.EOF, "", 1, 15),
			},
		},
		{
			name:  "doubles en dos líneas",
			input: "double p = 3.5;\nx = 10.0;",
			want: []token.Token{
				tok(token.DOUBLE_TYPE, "double", 1, 1),
				tok(token.IDENT, "p", 1, 8),
				tok(token.ASSIGN, "=", 1, 10),
				tok(token.DOUBLE, "3.5", 1, 12),
				tok(token.SEMICOLON, ";", 1, 15),
				tok(token.IDENT, "x", 2, 1),
				tok(token.ASSIGN, "=", 2, 3),
				tok(token.DOUBLE, "10.0", 2, 5),
				tok(token.SEMICOLON, ";", 2, 9),
				tok(token.EOF, "", 2, 10),
			},
		},
		{
			name:  "un punto sin dígito después no es decimal",
			input: "3.x",
			want: []token.Token{
				tok(token.INT, "3", 1, 1),
				tok(token.DOT, ".", 1, 2),
				tok(token.IDENT, "x", 1, 3),
				tok(token.EOF, "", 1, 4),
			},
		},
		{
			name:  "cadena con escapes",
			input: `"a\tb\n\"c\"\\"`,
			want: []token.Token{
				tok(token.STRING, "a\tb\n\"c\"\\", 1, 1),
				tok(token.EOF, "", 1, 16),
			},
		},
		{
			name:  "operadores de dos caracteres",
			input: "++ -- && || << >> += -= *= /= %= == != <= >=",
			want: []token.Token{
				tok(token.INCREMENT, "++", 1, 1),
				tok(token.DECREMENT, "--", 1, 4),
				tok(token.AND, "&&", 1, 7),
				tok(token.OR, "||", 1, 10),
				tok(token.SHIFT_LEFT, "<<", 1, 13),
				tok(token.SHIFT_RIGHT, ">>", 1, 16),
				tok(token.PLUS_ASSIGN, "+=", 1, 19),
				tok(token.MINUS_ASSIGN, "-=", 1, 22),
				tok(token.ASTERISK_ASSIGN, "*=", 1, 25),
				tok(token.SLASH_ASSIGN, "/=", 1, 28),
				tok(token.MOD_ASSIGN, "%=", 1, 31),
				tok(token.EQ, "==", 1, 34),
				tok(token.NOT_EQ, "!=", 1, 37),
				tok(token.LTE, "<=", 1, 40),
				tok(token.GTE, ">=", 1, 43),
				tok(token.EOF, "", 1, 45),
			},
		},
		{
			name:  "operadores y delimitadores de un carácter",
			input: "+ - * / % = < > ! . , ; ( ) { } [ ]",
			want: []token.Token{
				tok(token.PLUS, "+", 1, 1),
				tok(token.MINUS, "-", 1, 3),
				tok(token.ASTERISK, "*", 1, 5),
				tok(token.SLASH, "/", 1, 7),
				tok(token.MOD, "%", 1, 9),
				tok(token.ASSIGN, "=", 1, 11),
				tok(token.LT, "<", 1, 13),
				tok(token.GT, ">", 1, 15),
				tok(token.BANG, "!", 1, 17),
				tok(token.DOT, ".", 1, 19),
				tok(token.COMMA, ",", 1, 21),
				tok(token.SEMICOLON, ";", 1, 23),
				tok(token.LPAREN, "(", 1, 25),
				tok(token.RPAREN, ")", 1, 27),
				tok(token.LBRACE, "{", 1, 29),
				tok(token.RBRACE, "}", 1, 31),
				tok(token.LBRACKET, "[", 1, 33),
				tok(token.RBRACKET, "]", 1, 35),
				tok(token.EOF, "", 1, 36),
			},
		},
		{
			// El lexer toma siempre el operador más largo posible:
			// "a+++b" se parte como a ++ + b (igual que en C++).
			name:  "operadores pegados",
			input: "a+++b",
			want: []token.Token{
				tok(token.IDENT, "a", 1, 1),
				tok(token.INCREMENT, "++", 1, 2),
				tok(token.PLUS, "+", 1, 4),
				tok(token.IDENT, "b", 1, 5),
				tok(token.EOF, "", 1, 6),
			},
		},
		{
			name:  "cout con endl",
			input: "cout << x << endl;",
			want: []token.Token{
				tok(token.COUT, "cout", 1, 1),
				tok(token.SHIFT_LEFT, "<<", 1, 6),
				tok(token.IDENT, "x", 1, 9),
				tok(token.SHIFT_LEFT, "<<", 1, 11),
				tok(token.ENDL, "endl", 1, 14),
				tok(token.SEMICOLON, ";", 1, 18),
				tok(token.EOF, "", 1, 19),
			},
		},
		{
			name:  "identificadores con dígitos y guion bajo",
			input: "mi_var2 _x",
			want: []token.Token{
				tok(token.IDENT, "mi_var2", 1, 1),
				tok(token.IDENT, "_x", 1, 9),
				tok(token.EOF, "", 1, 11),
			},
		},
		{
			name:  "comentarios de línea y de bloque",
			input: "x // comentario\n/* bloque\n varias */ y",
			want: []token.Token{
				tok(token.IDENT, "x", 1, 1),
				tok(token.IDENT, "y", 3, 12),
				tok(token.EOF, "", 3, 13),
			},
		},
		{
			name:  "include y using namespace",
			input: "#include <iostream>\nusing namespace std;",
			want: []token.Token{
				tok(token.USING, "using", 2, 1),
				tok(token.NAMESPACE, "namespace", 2, 7),
				tok(token.STD, "std", 2, 17),
				tok(token.SEMICOLON, ";", 2, 20),
				tok(token.EOF, "", 2, 21),
			},
		},
		// --- Entradas con error léxico ---
		{
			name:  "carácter no válido",
			input: "x @ y",
			want: []token.Token{
				tok(token.IDENT, "x", 1, 1),
				tok(token.ILLEGAL, "carácter no válido '@'", 1, 3),
				tok(token.IDENT, "y", 1, 5),
				tok(token.EOF, "", 1, 6),
			},
		},
		{
			name:  "ampersand solo",
			input: "a & b",
			want: []token.Token{
				tok(token.IDENT, "a", 1, 1),
				tok(token.ILLEGAL, "carácter no válido '&' (¿quisiste escribir '&&'?)", 1, 3),
				tok(token.IDENT, "b", 1, 5),
				tok(token.EOF, "", 1, 6),
			},
		},
		{
			name:  "cadena sin cerrar al final",
			input: `"hola`,
			want: []token.Token{
				tok(token.ILLEGAL, "cadena sin cerrar: falta '\"'", 1, 1),
				tok(token.EOF, "", 1, 6),
			},
		},
		{
			// La cadena se corta en el salto de línea; la línea siguiente
			// se sigue tokenizando normalmente.
			name:  "cadena sin cerrar antes del salto de línea",
			input: "\"hola\nx",
			want: []token.Token{
				tok(token.ILLEGAL, "cadena sin cerrar: falta '\"'", 1, 1),
				tok(token.IDENT, "x", 2, 1),
				tok(token.EOF, "", 2, 2),
			},
		},
		{
			name:  "escape no válido",
			input: `"a\qb"`,
			want: []token.Token{
				tok(token.ILLEGAL, `secuencia de escape no válida '\q'`, 1, 1),
				tok(token.EOF, "", 1, 7),
			},
		},
		{
			name:  "comentario de bloque sin cerrar",
			input: "/* abierto",
			want: []token.Token{
				tok(token.ILLEGAL, "comentario /* sin cerrar", 1, 1),
				tok(token.EOF, "", 1, 11),
			},
		},
		{
			name:  "número mal formado",
			input: "3.4.5;",
			want: []token.Token{
				tok(token.ILLEGAL, "número mal formado '3.4.5'", 1, 1),
				tok(token.SEMICOLON, ";", 1, 6),
				tok(token.EOF, "", 1, 7),
			},
		},
		{
			name:  "entrada vacía",
			input: "",
			want: []token.Token{
				tok(token.EOF, "", 1, 1),
			},
		},
		{
			name:  "arreglo constante",
			input: "const int numeros[3] = {1, 2, 3};",
			want: []token.Token{
				tok(token.CONST, "const", 1, 1),
				tok(token.INT_TYPE, "int", 1, 7),
				tok(token.IDENT, "numeros", 1, 11),
				tok(token.LBRACKET, "[", 1, 18),
				tok(token.INT, "3", 1, 19),
				tok(token.RBRACKET, "]", 1, 20),
				tok(token.ASSIGN, "=", 1, 22),
				tok(token.LBRACE, "{", 1, 24),
				tok(token.INT, "1", 1, 25),
				tok(token.COMMA, ",", 1, 26),
				tok(token.INT, "2", 1, 28),
				tok(token.COMMA, ",", 1, 29),
				tok(token.INT, "3", 1, 31),
				tok(token.RBRACE, "}", 1, 32),
				tok(token.SEMICOLON, ";", 1, 33),
				tok(token.EOF, "", 1, 34),
			},
		},
		{
			name:  "saltos de línea Windows",
			input: "int x;\r\nint y;",
			want: []token.Token{
				tok(token.INT_TYPE, "int", 1, 1),
				tok(token.IDENT, "x", 1, 5),
				tok(token.SEMICOLON, ";", 1, 6),
				tok(token.INT_TYPE, "int", 2, 1),
				tok(token.IDENT, "y", 2, 5),
				tok(token.SEMICOLON, ";", 2, 6),
				tok(token.EOF, "", 2, 7),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			for i, want := range tt.want {
				got := l.NextToken()
				if got != want {
					t.Fatalf("token %d: se esperaba %+v, se obtuvo %+v", i, want, got)
				}
			}
		})
	}
}

// TestKeywords revisa que cada palabra reservada produzca su tipo de token
// y que una palabra cualquiera sea IDENT.
func TestKeywords(t *testing.T) {
	tests := []struct {
		input string
		want  token.TokenType
	}{
		{"int", token.INT_TYPE},
		{"double", token.DOUBLE_TYPE},
		{"bool", token.BOOL_TYPE},
		{"string", token.STRING_TYPE},
		{"void", token.VOID},
		{"true", token.TRUE},
		{"false", token.FALSE},
		{"if", token.IF},
		{"else", token.ELSE},
		{"while", token.WHILE},
		{"do", token.DO},
		{"for", token.FOR},
		{"return", token.RETURN},
		{"break", token.BREAK},
		{"continue", token.CONTINUE},
		{"const", token.CONST},
		{"cout", token.COUT},
		{"endl", token.ENDL},
		{"using", token.USING},
		{"namespace", token.NAMESPACE},
		{"std", token.STD},
		{"integer", token.IDENT}, // empieza como "int" pero no es reservada
		{"let", token.IDENT},     // ya no es palabra reservada
		{"function", token.IDENT},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := New(tt.input).NextToken()
			if got.Type != tt.want || got.Literal != tt.input {
				t.Fatalf("se esperaba {%s %s}, se obtuvo {%s %s}",
					tt.want, tt.input, got.Type, got.Literal)
			}
		})
	}
}

// TestProgramaDeReferencia tokeniza el programa completo de ejemplo y
// revisa que no haya ningún token ILLEGAL y que termine en EOF.
func TestProgramaDeReferencia(t *testing.T) {
	src, err := os.ReadFile("../parser/testdata/programa.cpp")
	if err != nil {
		t.Fatal(err)
	}

	l := New(string(src))
	count := 0
	for tk := l.NextToken(); tk.Type != token.EOF; tk = l.NextToken() {
		if tk.Type == token.ILLEGAL {
			t.Fatalf("token ILLEGAL en línea %d, columna %d: %s",
				tk.Line, tk.Column, tk.Literal)
		}
		count++
		if count > 10000 {
			t.Fatal("el lexer no llega a EOF")
		}
	}
}
