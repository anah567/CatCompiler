// Paquete token define los TOKENS del lenguaje: las unidades mínimas con
// significado (como las "palabras" de una oración).
//
// Es el vocabulario compartido entre las fases del compilador:
//
//	código fuente ──lexer──▶ tokens ──parser──▶ AST
//
// Por ejemplo, el código  let x = 5;  produce estos tokens:
//
//	{LET let} {IDENT x} {= =} {INT 5} {; ;}
package token

// TokenType es el TIPO de un token (qué clase de palabra es).
// Se usa string en lugar de int para que sea fácil de imprimir y depurar:
// al mostrar un token se ve "IDENT" o "+" en lugar de un número.
type TokenType string

// Token es la unidad mínima que genera el lexer.
type Token struct {
	Type    TokenType // qué clase de token es (IDENT, INT, PLUS, ...)
	Literal string    // texto exacto que se leyó del código ("x", "5", "+")
}

const (
	// Tokens especiales
	ILLEGAL = "ILLEGAL" // carácter que el lexer no reconoce
	EOF     = "EOF"     // fin de la entrada

	// Identificadores y literales
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	INT    = "INT"    // 1343456
	STRING = "STRING" // "foobar"

	// Operadores
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	POW      = "**" // potencia
	MOD      = "%"  // módulo

	// Operadores de comparación
	LT     = "<"
	GT     = ">"
	EQ     = "=="
	NOT_EQ = "!="
	LTE    = "<="
	GTE    = ">="

	// Delimitadores
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":" // separa clave y valor en los hashes {"a": 1}

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Palabras reservadas
	FUNCTION = "FUNCTION"
	LET      = "LET"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	ELIF     = "ELIF"
	FOR      = "FOR"
	WHILE    = "WHILE"
)

// keywords relaciona cada palabra reservada del lenguaje con su tipo de token.
// Las palabras reservadas se escriben igual que un identificador (solo
// letras), por eso el lexer primero lee la palabra completa y luego la busca
// en este mapa para saber si es reservada. Para agregar una palabra
// reservada nueva basta con definir su constante arriba y añadirla aquí.
var keywords = map[string]TokenType{
	"function": FUNCTION,
	"let":      LET,
	"true":     TRUE,
	"false":    FALSE,
	"for":      FOR,
	"while":    WHILE,
	"if":       IF,
	"else":     ELSE,
	"elif":     ELIF,
	"return":   RETURN,
}

// LookupIdent revisa si un identificador es una palabra reservada.
// Si lo es devuelve su tipo de token; si no, devuelve IDENT.
//
//	LookupIdent("let")  → LET
//	LookupIdent("edad") → IDENT
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
