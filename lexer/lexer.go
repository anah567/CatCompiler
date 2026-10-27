// Paquete lexer implementa el ANALIZADOR LÉXICO del compilador.
//
// ¿Qué hace?
// Recibe el código fuente como un texto (string) y lo divide en TOKENS:
// las "palabras" del lenguaje. Ignora los espacios, los comentarios y las
// líneas de preprocesador (#include), y agrupa los caracteres que van
// juntos (por ejemplo, las letras de un identificador o los dígitos de un
// número).
//
//	Entrada:  int x = 10 + y;
//	Tokens:   INT_TYPE  IDENT(x)  =  INT(10)  +  IDENT(y)  ;  EOF
//
// El lexer NO revisa si el orden de los tokens tiene sentido ("int int 5"
// se tokeniza sin problema); eso es trabajo del parser.
//
// ¿Cómo lo hace?
// Recorre la entrada un carácter a la vez. Mira el carácter actual (y a
// veces el siguiente) para decidir qué token empieza ahí. Cada llamada a
// NextToken devuelve UN token y deja el lexer listo para el siguiente.
// Además lleva la cuenta de la LÍNEA y la COLUMNA de cada carácter, y las
// guarda en cada token para poder reportar errores con su posición.
//
// Errores léxicos
// Cuando encuentra algo que no puede formar un token válido (un carácter
// como '@', una cadena sin cerrar, ...) devuelve un token ILLEGAL cuyo
// Literal es el MENSAJE del error. El lexer no se detiene: la siguiente
// llamada a NextToken sigue con el resto de la entrada.
//
// Limitación: trabaja con bytes, no con runas Unicode, así que solo
// reconoce identificadores con letras ASCII (sin tildes ni ñ). Dentro de
// las cadenas y los comentarios sí se puede escribir cualquier carácter.
package lexer

import (
	"polaobesahermosa/token"
	"strings"
)

// Lexer guarda el estado del recorrido por la entrada.
//
// Ejemplo con input = "int", después de leer la 'n':
//
//	input:        i  n  t
//	índice:       0  1  2
//	position     =   1        (apunta a 'n', el carácter actual)
//	readPosition =      2     (apunta a 't', el siguiente por leer)
//	ch           = 'n'
//	line = 1, column = 2      (posición de 'n' en el texto)
type Lexer struct {
	input        string
	position     int  // posición actual en la entrada (apunta al carácter actual, ch)
	readPosition int  // siguiente posición a leer (siempre position + 1)
	ch           byte // carácter que se está examinando (0 = fin de la entrada)
	line         int  // línea del carácter actual (empieza en 1)
	column       int  // columna del carácter actual (empieza en 1)
}

// New crea un lexer para la entrada dada. Llama a readChar una vez para
// que ch quede con el primer carácter (línea 1, columna 1) y el lexer esté
// listo para usarse.
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

// readChar avanza una posición en la entrada: carga en ch el siguiente
// carácter y mueve ambos índices. Al pasar del final, ch queda en 0
// (el carácter NUL de ASCII), que el lexer interpreta como fin de archivo.
//
// También actualiza línea y columna: si el carácter que se DEJA atrás es
// un salto de línea, el nuevo carácter está al inicio de la línea
// siguiente (columna 1); si no, simplemente se avanza una columna.
//
// Si ya se pasó del final, no hace nada: así llamar readChar de más al
// llegar a EOF no mueve la columna del token EOF.
func (l *Lexer) readChar() {
	if l.readPosition > len(l.input) {
		return
	}
	if l.ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

// NextToken devuelve el siguiente token de la entrada. Se llama
// repetidamente hasta obtener un token de tipo EOF.
//
// Funcionamiento:
//  1. Se saltan los espacios en blanco.
//  2. Se guarda la línea y columna donde EMPIEZA el token.
//  3. Según el carácter actual (ch) se decide qué token es.
//     - Para operadores que pueden tener uno o dos caracteres ('=' / '==',
//     '+' / '++' / '+=', ...) se usa peekChar para mirar el siguiente
//     carácter SIN consumirlo, y así decidir.
//     - Los comentarios y las líneas '#' no producen token: se saltan y se
//     vuelve a llamar a NextToken para devolver el token que sigue.
//  4. Al final se avanza un carácter (readChar) para que la próxima
//     llamada empiece después del token actual.
//     (Los identificadores y números son la excepción: sus funciones de
//     lectura ya avanzan hasta el carácter siguiente, por eso retornan
//     antes de llegar al readChar del final.)
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	// Posición del primer carácter del token.
	line, column := l.line, l.column

	switch l.ch {
	case '=':
		// "==" (comparación) o "=" (asignación)
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.EQ)
		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case '+':
		// "++", "+=" o "+"
		if l.peekChar() == '+' {
			tok = l.makeTwoCharToken(token.INCREMENT)
		} else if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.PLUS_ASSIGN)
		} else {
			tok = newToken(token.PLUS, l.ch)
		}
	case '-':
		// "--", "-=" o "-"
		if l.peekChar() == '-' {
			tok = l.makeTwoCharToken(token.DECREMENT)
		} else if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.MINUS_ASSIGN)
		} else {
			tok = newToken(token.MINUS, l.ch)
		}
	case '*':
		// "*=" o "*"
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.ASTERISK_ASSIGN)
		} else {
			tok = newToken(token.ASTERISK, l.ch)
		}
	case '/':
		// "//" comentario de línea, "/*" comentario de bloque, "/=" o "/"
		if l.peekChar() == '/' {
			l.skipLine()
			return l.NextToken()
		} else if l.peekChar() == '*' {
			if !l.skipBlockComment() {
				return illegal("comentario /* sin cerrar", line, column)
			}
			return l.NextToken()
		} else if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.SLASH_ASSIGN)
		} else {
			tok = newToken(token.SLASH, l.ch)
		}
	case '%':
		// "%=" o "%"
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.MOD_ASSIGN)
		} else {
			tok = newToken(token.MOD, l.ch)
		}
	case '!':
		// "!=" (distinto) o "!" (negación)
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.NOT_EQ)
		} else {
			tok = newToken(token.BANG, l.ch)
		}
	case '&':
		// Solo existe "&&". Un '&' suelto (referencias, operador de bits)
		// no está en el subconjunto, así que es un error léxico.
		if l.peekChar() == '&' {
			tok = l.makeTwoCharToken(token.AND)
		} else {
			tok = illegal("carácter no válido '&' (¿quisiste escribir '&&'?)", line, column)
		}
	case '|':
		// Solo existe "||".
		if l.peekChar() == '|' {
			tok = l.makeTwoCharToken(token.OR)
		} else {
			tok = illegal("carácter no válido '|' (¿quisiste escribir '||'?)", line, column)
		}
	case '<':
		// "<=", "<<" o "<"
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.LTE)
		} else if l.peekChar() == '<' {
			tok = l.makeTwoCharToken(token.SHIFT_LEFT)
		} else {
			tok = newToken(token.LT, l.ch)
		}
	case '>':
		// ">=", ">>" o ">"
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.GTE)
		} else if l.peekChar() == '>' {
			tok = l.makeTwoCharToken(token.SHIFT_RIGHT)
		} else {
			tok = newToken(token.GT, l.ch)
		}
	case '#':
		// Línea de preprocesador (#include <iostream>): el intérprete no
		// tiene preprocesador, así que se descarta la línea completa.
		l.skipLine()
		return l.NextToken()
	// Delimitadores de un solo carácter.
	case '.':
		tok = newToken(token.DOT, l.ch)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch)
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case '{':
		tok = newToken(token.LBRACE, l.ch)
	case '}':
		tok = newToken(token.RBRACE, l.ch)
	case '[':
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		tok = newToken(token.RBRACKET, l.ch)
	case '"':
		// Cadena de texto. El literal del token es el contenido SIN las
		// comillas y con los escapes ya convertidos:
		// "a\n" → Token{STRING, a + salto de línea}.
		// readString deja ch sobre la comilla de cierre; el readChar del
		// final de esta función la salta.
		str, errMsg := l.readString()
		if errMsg != "" {
			tok = illegal(errMsg, line, column)
		} else {
			tok = token.Token{Type: token.STRING, Literal: str}
		}
	case 0:
		// Fin de la entrada.
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			// Identificador o palabra reservada: se lee la palabra completa
			// y luego se consulta si es reservada ("int" → INT_TYPE) o un
			// nombre cualquiera ("x" → IDENT).
			// Se retorna aquí porque readIdentifier ya avanzó el lexer.
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			tok.Line, tok.Column = line, column
			return tok
		} else if isDigit(l.ch) {
			// Número: entero (42) o double (3.14).
			// Mismo caso: readNumber ya avanzó, por eso se retorna aquí.
			tok = l.readNumber()
			tok.Line, tok.Column = line, column
			return tok
		} else {
			// Cualquier otro carácter (por ejemplo '@' o '$') es ilegal.
			tok = illegal("carácter no válido '"+string(l.ch)+"'", line, column)
		}
	}

	// Se avanza al carácter siguiente al token que se acaba de reconocer.
	l.readChar()
	tok.Line, tok.Column = line, column
	return tok
}

// makeTwoCharToken construye un token de dos caracteres (como "==" o "++")
// uniendo el carácter actual con el siguiente. Avanza el lexer una vez,
// dejando ch sobre el SEGUNDO carácter; el readChar del final de NextToken
// se encarga de pasar al siguiente.
func (l *Lexer) makeTwoCharToken(tokenType token.TokenType) token.Token {
	ch := l.ch   // se guarda el primer carácter antes de avanzar
	l.readChar() // ahora l.ch es el segundo carácter
	return token.Token{Type: tokenType, Literal: string(ch) + string(l.ch)}
}

// readIdentifier lee una secuencia de letras, dígitos y '_' que empieza con
// letra o '_' (edad, suma2, mi_var) y la devuelve. Al terminar, ch es el
// primer carácter que NO forma parte del identificador.
//
// Se recuerda dónde empezó la palabra y se avanza mientras haya letras o
// dígitos; la palabra es el pedazo de la entrada entre el inicio y la
// posición final.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber lee un número y devuelve su token (sin posición: la pone
// NextToken). Al terminar, ch es el primer carácter que NO es parte del
// número.
//
//	42      → INT "42"
//	3.14    → DOUBLE "3.14"
//	3.4.5   → ILLEGAL "número mal formado '3.4.5'"
//
// Un punto solo forma parte del número si después viene un dígito; así
// "3.x" no se toma como número decimal.
func (l *Lexer) readNumber() token.Token {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}

	// ¿Tiene parte decimal?
	if l.ch != '.' || !isDigit(l.peekChar()) {
		return token.Token{Type: token.INT, Literal: l.input[position:l.position]}
	}
	l.readChar() // se salta el '.'
	for isDigit(l.ch) {
		l.readChar()
	}

	// Un segundo punto seguido de dígitos (3.4.5) es un número mal formado.
	// Se consume todo el pedazo para que el error salga una sola vez.
	if l.ch == '.' && isDigit(l.peekChar()) {
		for l.ch == '.' || isDigit(l.ch) {
			l.readChar()
		}
		return token.Token{Type: token.ILLEGAL,
			Literal: "número mal formado '" + l.input[position:l.position] + "'"}
	}

	return token.Token{Type: token.DOUBLE, Literal: l.input[position:l.position]}
}

// readString lee el contenido de una cadena entre comillas dobles,
// convirtiendo las secuencias de escape:
//
//	\n → salto de línea   \t → tabulación   \" → comilla   \\ → barra
//
//	Al entrar: ch = '"' de apertura
//	Al salir:  ch = '"' de cierre (o el '\n' / 0 donde se cortó)
//
// Devuelve el texto ya convertido y un mensaje de error vacío, o un
// mensaje si la cadena no se cerró en la misma línea o tiene un escape no
// válido. Igual que en C++, una cadena no puede tener un salto de línea
// "real" adentro (hay que escribir \n).
func (l *Lexer) readString() (string, string) {
	var out strings.Builder
	errMsg := ""

	for {
		l.readChar()
		switch l.ch {
		case '"':
			return out.String(), errMsg
		case 0, '\n':
			return out.String(), "cadena sin cerrar: falta '\"'"
		case '\\':
			l.readChar() // ch = carácter después de la barra
			switch l.ch {
			case 'n':
				out.WriteByte('\n')
			case 't':
				out.WriteByte('\t')
			case '"':
				out.WriteByte('"')
			case '\\':
				out.WriteByte('\\')
			case 0, '\n':
				return out.String(), "cadena sin cerrar: falta '\"'"
			default:
				// Se guarda el error pero se sigue leyendo hasta la comilla
				// de cierre, para que el resto de la cadena no se confunda
				// con código.
				if errMsg == "" {
					errMsg = "secuencia de escape no válida '\\" + string(l.ch) + "'"
				}
			}
		default:
			out.WriteByte(l.ch)
		}
	}
}

// skipWhitespace avanza mientras el carácter actual sea un espacio, una
// tabulación o un salto de línea. En este lenguaje los espacios solo
// separan tokens; no tienen significado propio.
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// skipLine avanza hasta el final de la línea actual (sin consumir el '\n',
// que luego salta skipWhitespace). Se usa para los comentarios "// ..." y
// para las líneas de preprocesador "#include ...".
func (l *Lexer) skipLine() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// skipBlockComment salta un comentario de bloque "/* ... */", que puede
// ocupar varias líneas.
//
//	Al entrar: ch = '/' de la apertura "/*"
//	Al salir:  ch = carácter siguiente al "*/" de cierre
//
// Devuelve false si la entrada terminó sin encontrar "*/".
func (l *Lexer) skipBlockComment() bool {
	l.readChar() // ch = '*'
	l.readChar() // ch = primer carácter dentro del comentario
	for l.ch != 0 {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // ch = '/'
			l.readChar() // ch = carácter después del comentario
			return true
		}
		l.readChar()
	}
	return false
}

// peekChar devuelve el SIGUIENTE carácter sin avanzar el lexer ("mirar
// adelante"). Se usa para distinguir, por ejemplo, '=' de '=='.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// newToken crea un token de un solo carácter.
func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}

// illegal crea un token ILLEGAL con el mensaje del error léxico y la
// posición donde empezó el problema.
func illegal(msg string, line, column int) token.Token {
	return token.Token{Type: token.ILLEGAL, Literal: msg, Line: line, Column: column}
}

// isLetter indica si el carácter puede empezar un identificador:
// letras ASCII mayúsculas o minúsculas y el guion bajo (mi_variable).
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit indica si el carácter es un dígito decimal (0-9).
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
