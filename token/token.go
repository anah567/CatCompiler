// Paquete token define los TOKENS del lenguaje: las unidades mínimas con
// significado (como las "palabras" de una oración).
//
// Es el vocabulario compartido entre las fases del compilador:
//
//	código fuente ──lexer──▶ tokens ──parser──▶ AST
//
// El lenguaje es un SUBCONJUNTO DE C++. Por ejemplo, el código
// int x = 5;  produce estos tokens (tipo, literal, línea, columna):
//
//	{INT_TYPE int 1 1} {IDENT x 1 5} {= = 1 7} {INT 5 1 9} {; ; 1 10}
package token

// TokenType es el TIPO de un token (qué clase de palabra es).
// Se usa string en lugar de int para que sea fácil de imprimir y depurar:
// al mostrar un token se ve "IDENT" o "+" en lugar de un número.
type TokenType string

// Token es la unidad mínima que genera el lexer.
//
// Line y Column indican DÓNDE empieza el token en el código fuente
// (ambas empiezan en 1). Sirven para que el parser y el evaluador puedan
// decir "error en la línea 3, columna 7".
type Token struct {
	Type    TokenType // qué clase de token es (IDENT, INT, PLUS, ...)
	Literal string    // texto exacto que se leyó del código ("x", "5", "+")
	Line    int       // línea donde empieza el token (1, 2, 3, ...)
	Column  int       // columna donde empieza el token (1, 2, 3, ...)
}

const (
	// Tokens especiales
	ILLEGAL = "ILLEGAL" // error léxico; el Literal trae el mensaje del error
	EOF     = "EOF"     // fin de la entrada

	// Identificadores y literales
	IDENT  = "IDENT"  // suma, edad, x, ...
	INT    = "INT"    // 1343456
	DOUBLE = "DOUBLE" // 3.14
	STRING = "STRING" // "hola\n" (el literal ya trae los escapes convertidos)

	// Operadores aritméticos
	PLUS     = "+"
	MINUS    = "-"
	ASTERISK = "*"
	SLASH    = "/"
	MOD      = "%" // módulo (residuo)

	// Operadores de incremento y decremento (prefijos o postfijos)
	INCREMENT = "++" // i++  ++i
	DECREMENT = "--" // i--  --i

	// Asignación simple y compuesta
	ASSIGN          = "="
	PLUS_ASSIGN     = "+="
	MINUS_ASSIGN    = "-="
	ASTERISK_ASSIGN = "*="
	SLASH_ASSIGN    = "/="
	MOD_ASSIGN      = "%="

	// Operadores lógicos
	BANG = "!"  // negación
	AND  = "&&" // y lógico
	OR   = "||" // o lógico

	// Operadores de comparación
	LT     = "<"
	GT     = ">"
	EQ     = "=="
	NOT_EQ = "!="
	LTE    = "<="
	GTE    = ">="

	// Desplazamiento. En este lenguaje "<<" se usa sobre todo con cout:
	// cout << x;
	SHIFT_LEFT  = "<<"
	SHIFT_RIGHT = ">>"

	// Acceso a miembro: s.size()
	DOT = "."

	// Delimitadores
	COMMA     = ","
	SEMICOLON = ";"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Palabras reservadas: tipos de datos
	INT_TYPE    = "INT_TYPE"    // int
	DOUBLE_TYPE = "DOUBLE_TYPE" // double
	BOOL_TYPE   = "BOOL_TYPE"   // bool
	STRING_TYPE = "STRING_TYPE" // string
	VOID        = "VOID"        // void

	// Palabras reservadas: valores y control de flujo
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	WHILE    = "WHILE"
	DO       = "DO"
	FOR      = "FOR"
	RETURN   = "RETURN"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"
	CONST    = "CONST"

	// Palabras reservadas: salida estándar y espacio de nombres
	COUT      = "COUT"      // cout
	ENDL      = "ENDL"      // endl
	USING     = "USING"     // using
	NAMESPACE = "NAMESPACE" // namespace
	STD       = "STD"       // std
)

// keywords relaciona cada palabra reservada del lenguaje con su tipo de token.
// Las palabras reservadas se escriben igual que un identificador (solo
// letras), por eso el lexer primero lee la palabra completa y luego la busca
// en este mapa para saber si es reservada. Para agregar una palabra
// reservada nueva basta con definir su constante arriba y añadirla aquí.
//
// Nota: en C++ real "string", "cout", "endl" y "std" no son palabras
// reservadas sino nombres de la biblioteca estándar. Aquí se tratan como
// reservadas para simplificar el parser (ver docs/DECISIONES_P1.md).
var keywords = map[string]TokenType{
	"int":       INT_TYPE,
	"double":    DOUBLE_TYPE,
	"bool":      BOOL_TYPE,
	"string":    STRING_TYPE,
	"void":      VOID,
	"true":      TRUE,
	"false":     FALSE,
	"if":        IF,
	"else":      ELSE,
	"while":     WHILE,
	"do":        DO,
	"for":       FOR,
	"return":    RETURN,
	"break":     BREAK,
	"continue":  CONTINUE,
	"const":     CONST,
	"cout":      COUT,
	"endl":      ENDL,
	"using":     USING,
	"namespace": NAMESPACE,
	"std":       STD,
}

// LookupIdent revisa si un identificador es una palabra reservada.
// Si lo es devuelve su tipo de token; si no, devuelve IDENT.
//
//	LookupIdent("int")  → INT_TYPE
//	LookupIdent("edad") → IDENT
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// IsType indica si el token es el nombre de un tipo de dato
// (int, double, bool, string o void). El parser lo usa para saber si una
// sentencia es una declaración: "int x" empieza con un tipo, "x = 1" no.
func IsType(t TokenType) bool {
	switch t {
	case INT_TYPE, DOUBLE_TYPE, BOOL_TYPE, STRING_TYPE, VOID:
		return true
	}
	return false
}
