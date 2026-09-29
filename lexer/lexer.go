// Paquete lexer implementa el ANALIZADOR LÉXICO del compilador.
//
// ¿Qué hace?
// Recibe el código fuente como un texto (string) y lo divide en TOKENS:
// las "palabras" del lenguaje. Ignora los espacios y agrupa los caracteres
// que van juntos (por ejemplo, las letras de un identificador o los dígitos
// de un número).
//
//	Entrada:  let x = 10 + y;
//	Tokens:   LET  IDENT(x)  =  INT(10)  +  IDENT(y)  ;  EOF
//
// El lexer NO revisa si el orden de los tokens tiene sentido ("let let 5"
// se tokeniza sin problema); eso es trabajo del parser.
//
// ¿Cómo lo hace?
// Recorre la entrada un carácter a la vez. Mira el carácter actual (y a
// veces el siguiente) para decidir qué token empieza ahí. Cada llamada a
// NextToken devuelve UN token y deja el lexer listo para el siguiente.
//
// Limitación: trabaja con bytes, no con runas Unicode, así que solo
// reconoce identificadores con letras ASCII (sin tildes ni ñ).
package lexer

import "polaobesahermosa/token"

// Lexer guarda el estado del recorrido por la entrada.
//
// Ejemplo con input = "let", después de leer la 'e':
//
//	input:        l  e  t
//	índice:       0  1  2
//	position     =   1        (apunta a 'e', el carácter actual)
//	readPosition =      2     (apunta a 't', el siguiente por leer)
//	ch           = 'e'
type Lexer struct {
	input        string
	position     int  // posición actual en la entrada (apunta al carácter actual, ch)
	readPosition int  // siguiente posición a leer (siempre position + 1)
	ch           byte // carácter que se está examinando (0 = fin de la entrada)
}

// New crea un lexer para la entrada dada. Llama a readChar una vez para
// que ch quede con el primer carácter y el lexer esté listo para usarse.
func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// readChar avanza una posición en la entrada: carga en ch el siguiente
// carácter y mueve ambos índices. Al pasar del final, ch queda en 0
// (el carácter NUL de ASCII), que el lexer interpreta como fin de archivo.
func (l *Lexer) readChar() {
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
//  2. Según el carácter actual (ch) se decide qué token es.
//     - Para operadores que pueden tener uno o dos caracteres ('=' / '==',
//     '<' / '<=', '*' / '**', ...) se usa peekChar para mirar el
//     siguiente carácter SIN consumirlo, y así decidir.
//  3. Al final se avanza un carácter (readChar) para que la próxima
//     llamada empiece después del token actual.
//     (Los identificadores y números son la excepción: sus funciones de
//     lectura ya avanzan hasta el carácter siguiente, por eso retornan
//     antes de llegar al readChar del final.)
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	switch l.ch {
	case '=':
		// "==" (comparación) o "=" (asignación)
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.EQ)
		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case '+':
		tok = newToken(token.PLUS, l.ch)
	case '-':
		tok = newToken(token.MINUS, l.ch)
	case '!':
		// "!=" (distinto) o "!" (negación)
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.NOT_EQ)
		} else {
			tok = newToken(token.BANG, l.ch)
		}
	case '/':
		tok = newToken(token.SLASH, l.ch)
	case '*':
		// "**" (potencia) o "*" (multiplicación)
		if l.peekChar() == '*' {
			tok = l.makeTwoCharToken(token.POW)
		} else {
			tok = newToken(token.ASTERISK, l.ch)
		}
	case '%':
		tok = newToken(token.MOD, l.ch)
	case '<':
		// "<=" o "<"
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.LTE)
		} else {
			tok = newToken(token.LT, l.ch)
		}
	case '>':
		// ">=" o ">"
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(token.GTE)
		} else {
			tok = newToken(token.GT, l.ch)
		}
	// Delimitadores de un solo carácter.
	case ';':
		tok = newToken(token.SEMICOLON, l.ch)
	case ':':
		tok = newToken(token.COLON, l.ch)
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
		// comillas: "hola" → Token{STRING, hola}.
		// readString deja ch sobre la comilla de cierre; el readChar del
		// final de esta función la salta.
		str, ok := l.readString()
		if ok {
			tok = token.Token{Type: token.STRING, Literal: str}
		} else {
			// Se llegó al final de la entrada sin cerrar la comilla:
			// se reporta como token ILEGAL para que el parser lo rechace.
			tok = token.Token{Type: token.ILLEGAL, Literal: "\"" + str}
		}
	case 0:
		// Fin de la entrada.
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			// Identificador o palabra reservada: se leen todas las letras
			// seguidas y luego se consulta si la palabra es reservada
			// ("let" → LET) o un nombre cualquiera ("x" → IDENT).
			// Se retorna aquí porque readIdentifier ya avanzó el lexer.
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			// Número entero: se leen todos los dígitos seguidos.
			// Mismo caso: readNumber ya avanzó, por eso se retorna aquí.
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			return tok
		} else {
			// Cualquier otro carácter (por ejemplo '@' o '#') es ilegal.
			tok = newToken(token.ILLEGAL, l.ch)
		}
	}

	// Se avanza al carácter siguiente al token que se acaba de reconocer.
	l.readChar()
	return tok
}

// makeTwoCharToken construye un token de dos caracteres (como "==" o "<=")
// uniendo el carácter actual con el siguiente. Avanza el lexer una vez,
// dejando ch sobre el SEGUNDO carácter; el readChar del final de NextToken
// se encarga de pasar al siguiente.
func (l *Lexer) makeTwoCharToken(tokenType token.TokenType) token.Token {
	ch := l.ch   // se guarda el primer carácter antes de avanzar
	l.readChar() // ahora l.ch es el segundo carácter
	return token.Token{Type: tokenType, Literal: string(ch) + string(l.ch)}
}

// readIdentifier lee una secuencia de letras (y '_') y la devuelve.
// Al terminar, ch es el primer carácter que NO es letra.
//
// Se recuerda dónde empezó la palabra y se avanza mientras haya letras;
// la palabra es el pedazo de la entrada entre el inicio y la posición final.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber lee una secuencia de dígitos y la devuelve (como texto; el
// parser se encarga de convertirla a número). Al terminar, ch es el primer
// carácter que NO es dígito.
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readString lee el contenido de una cadena entre comillas dobles.
//
//	Al entrar: ch = '"' de apertura
//	Al salir:  ch = '"' de cierre (o 0 si no se cerró)
//
// Devuelve el texto entre las comillas y true, o lo que alcanzó a leer y
// false si la entrada terminó antes de encontrar la comilla de cierre.
func (l *Lexer) readString() (string, bool) {
	position := l.position + 1 // el contenido empieza después de la comilla
	for {
		l.readChar()
		if l.ch == '"' {
			return l.input[position:l.position], true
		}
		if l.ch == 0 {
			return l.input[position:l.position], false
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

// isLetter indica si el carácter puede formar parte de un identificador:
// letras ASCII mayúsculas o minúsculas y el guion bajo (mi_variable).
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit indica si el carácter es un dígito decimal (0-9).
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
