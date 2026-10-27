// Paquete parser implementa el ANALIZADOR SINTÁCTICO del compilador.
//
// ¿Qué hace?
// Recibe la secuencia de tokens que produce el lexer y construye con ellos
// un Árbol de Sintaxis Abstracta (AST). El AST representa la ESTRUCTURA
// del programa: qué operación se aplica sobre qué operandos, qué sentencias
// hay dentro de qué bloques, etc.
//
//	Código:  int x = 1 + 2 * 3;
//	Tokens:  INT_TYPE IDENT(x) = INT(1) + INT(2) * INT(3) ;
//	AST:     VarDeclaration int x
//	         └── Valor: (+)
//	             ├── 1
//	             └── (*)
//	                 ├── 2
//	                 └── 3
//
// ¿Cómo lo hace?
// Las SENTENCIAS (declaraciones, if, while, for, return, cout, ...) se
// reconocen por su primer token con un parser descendente recursivo
// normal: hay una función parseXxx por cada regla de la gramática
// (ver docs/gramatica_cpp.ebnf).
//
// Las EXPRESIONES se parsean con la técnica de Vaughan Pratt ("Top Down
// Operator Precedence"):
//
//  1. Cada TIPO DE TOKEN tiene asociadas hasta dos funciones de parseo:
//     - una función PREFIJA (prefixParseFn): se usa cuando el token aparece
//     al INICIO de una expresión. Ej: el '-' en "-5", un número "5",
//     un identificador "x", un '(' en "(a + b)", el '++' en "++i".
//     - una función INFIJA (infixParseFn): se usa cuando el token aparece
//     DESPUÉS de una expresión. Ej: el '+' en "a + b", el '(' en "f(x)",
//     el '++' en "i++".
//
//  2. Cada operador infijo tiene una PRECEDENCIA (qué tan fuerte "atrapa"
//     a sus operandos). La función parseExpression usa esas precedencias
//     para decidir cómo agrupar. Así "1 + 2 * 3" se agrupa como
//     "1 + (2 * 3)" sin necesidad de escribir una regla de gramática por
//     cada nivel de precedencia.
//
// Convención importante: curToken y peekToken
// El parser siempre mira DOS tokens: el actual (curToken) y el siguiente
// (peekToken). En los comentarios de cada función se indica en qué token
// debe estar el parser al ENTRAR y en cuál queda al SALIR. Respetar esa
// convención es la clave para que todas las funciones encajen entre sí.
// Regla general: una función de parseo empieza con curToken en el primer
// token de su construcción y termina con curToken en el ÚLTIMO token de
// esa construcción (por ejemplo el ';' o la '}'), no en el siguiente.
//
// Errores
// Los errores se guardan en una lista (no se detiene en el primero) con el
// formato:
//
//	Error sintáctico [línea 3, columna 5]: se esperaba ';' pero se encontró '}'
//	Error léxico [línea 2, columna 9]: carácter no válido '@'
//
// Después de un error en una sentencia, el parser salta tokens hasta un
// punto seguro (ver synchronize) para no producir errores en cascada.
package parser

import (
	"fmt"
	"polaobesahermosa/ast"
	"polaobesahermosa/lexer"
	"polaobesahermosa/token"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Precedencias
// ---------------------------------------------------------------------------

// Niveles de precedencia, de MENOR a MAYOR, iguales a los de C++.
// iota asigna números consecutivos: LOWEST = 1, ASSIGN = 2, ... POSTFIX = 11.
// (El '_' descarta el valor 0.)
//
// Un operador con mayor precedencia se agrupa primero. Por ejemplo, como
// PRODUCT > SUM, en "1 + 2 * 3" la multiplicación se resuelve antes.
const (
	_           int = iota
	LOWEST          // precedencia mínima: se usa al empezar a parsear una expresión
	ASSIGN          // =  +=  -=  *=  /=  %=   (asociativa a la derecha)
	OR              // ||
	AND             // &&
	EQUALS          // ==  !=
	LESSGREATER     // >  <  >=  <=
	SHIFT           // <<  >>
	SUM             // +  -
	PRODUCT         // *  /  %
	PREFIX          // -x  !x  ++x  --x        (operadores prefijos)
	POSTFIX         // x++  x--  f(x)  a[i]  s.m  (la más alta de todas)
)

// precedences asocia cada token que puede ir DESPUÉS de una expresión
// (operador infijo o postfijo) con su nivel de precedencia. Los tokens que
// no están en este mapa (por ejemplo ';', ')', ',') tienen precedencia
// LOWEST, lo que hace que parseExpression se detenga al encontrarlos.
//
// Nótese que '(' y '[' aparecen aquí: cuando van DESPUÉS de una expresión
// se comportan como operadores (llamada "f(x)" e índice "a[0]").
var precedences = map[token.TokenType]int{
	token.ASSIGN:          ASSIGN,
	token.PLUS_ASSIGN:     ASSIGN,
	token.MINUS_ASSIGN:    ASSIGN,
	token.ASTERISK_ASSIGN: ASSIGN,
	token.SLASH_ASSIGN:    ASSIGN,
	token.MOD_ASSIGN:      ASSIGN,
	token.OR:              OR,
	token.AND:             AND,
	token.EQ:              EQUALS,
	token.NOT_EQ:          EQUALS,
	token.LT:              LESSGREATER,
	token.GT:              LESSGREATER,
	token.LTE:             LESSGREATER,
	token.GTE:             LESSGREATER,
	token.SHIFT_LEFT:      SHIFT,
	token.SHIFT_RIGHT:     SHIFT,
	token.PLUS:            SUM,
	token.MINUS:           SUM,
	token.SLASH:           PRODUCT,
	token.ASTERISK:        PRODUCT,
	token.MOD:             PRODUCT,
	token.INCREMENT:       POSTFIX,
	token.DECREMENT:       POSTFIX,
	token.LPAREN:          POSTFIX,
	token.LBRACKET:        POSTFIX,
	token.DOT:             POSTFIX,
}

// ---------------------------------------------------------------------------
// Tipos y estructura del parser
// ---------------------------------------------------------------------------

// Tipos de las funciones de parseo.
type (
	// prefixParseFn parsea una expresión que EMPIEZA en curToken.
	// No recibe nada porque no hay nada a su izquierda.
	prefixParseFn func() ast.Expression

	// infixParseFn recibe como argumento la expresión que ya se parseó a
	// la IZQUIERDA del operador (curToken) y devuelve la expresión
	// completa "izquierda operador derecha".
	infixParseFn func(ast.Expression) ast.Expression
)

// Parser guarda el estado del análisis sintáctico.
type Parser struct {
	l      *lexer.Lexer // fuente de tokens
	errors []string     // errores encontrados (no se detiene en el primero)

	curToken  token.Token // token que se está analizando en este momento
	peekToken token.Token // token siguiente; permite "mirar adelante" para decidir

	// blockDepth cuenta cuántos bloques '{' hay abiertos alrededor del
	// punto actual. Vale 0 en el nivel superior del programa. Se usa para
	// prohibir funciones dentro de otras funciones, como en C++.
	blockDepth int

	// Tablas que relacionan cada tipo de token con su función de parseo.
	// Son el corazón del parser de Pratt: en lugar de un gran switch,
	// se busca en estas tablas qué hacer con cada token.
	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

// New crea un parser que leerá los tokens del lexer 'l', y registra en las
// tablas qué función de parseo corresponde a cada tipo de token.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// --- Funciones PREFIJAS: tokens que pueden INICIAR una expresión ---
	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)           // x
	p.registerPrefix(token.INT, p.parseIntegerLiteral)         // 5
	p.registerPrefix(token.DOUBLE, p.parseDoubleLiteral)       // 3.5
	p.registerPrefix(token.STRING, p.parseStringLiteral)       // "hola"
	p.registerPrefix(token.TRUE, p.parseBoolean)               // true
	p.registerPrefix(token.FALSE, p.parseBoolean)              // false
	p.registerPrefix(token.BANG, p.parsePrefixExpression)      // !x
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)     // -x
	p.registerPrefix(token.INCREMENT, p.parsePrefixExpression) // ++x
	p.registerPrefix(token.DECREMENT, p.parsePrefixExpression) // --x
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)   // (a + b)
	p.registerPrefix(token.ENDL, p.parseEndl)                  // endl

	// --- Funciones INFIJAS: tokens que van DESPUÉS de una expresión ---
	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	for _, op := range []token.TokenType{
		token.PLUS, token.MINUS, token.ASTERISK, token.SLASH, token.MOD, // a + b
		token.EQ, token.NOT_EQ, token.LT, token.GT, token.LTE, token.GTE, // a < b
		token.AND, token.OR, // a && b
		token.SHIFT_LEFT, token.SHIFT_RIGHT, // a << b
	} {
		p.registerInfix(op, p.parseInfixExpression)
	}
	for _, op := range []token.TokenType{
		token.ASSIGN, token.PLUS_ASSIGN, token.MINUS_ASSIGN,
		token.ASTERISK_ASSIGN, token.SLASH_ASSIGN, token.MOD_ASSIGN,
	} {
		p.registerInfix(op, p.parseAssignExpression) // a = b   a += b
	}
	p.registerInfix(token.INCREMENT, p.parsePostfixExpression) // i++
	p.registerInfix(token.DECREMENT, p.parsePostfixExpression) // i--
	p.registerInfix(token.LPAREN, p.parseCallExpression)       // f(a, b)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)    // a[i]
	p.registerInfix(token.DOT, p.parseMemberExpression)        // s.size

	// Se leen dos tokens para que curToken y peekToken queden inicializados.
	// Tras la primera llamada: cur = (vacío), peek = token 1.
	// Tras la segunda:         cur = token 1, peek = token 2.
	p.nextToken()
	p.nextToken()

	return p
}

// ---------------------------------------------------------------------------
// Utilidades para moverse por los tokens y reportar errores
// ---------------------------------------------------------------------------

// registerPrefix asocia una función de parseo prefija a un tipo de token.
func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

// registerInfix asocia una función de parseo infija a un tipo de token.
func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// Errors devuelve los errores léxicos y sintácticos encontrados durante el
// parseo, en el orden en que aparecen. Cada mensaje ya incluye el tipo de
// error, la línea y la columna. Quien use el parser (por ejemplo el REPL)
// debe revisar esta lista después de llamar a ParseProgram: si no está
// vacía, el AST puede estar incompleto y NO debe evaluarse.
func (p *Parser) Errors() []string {
	return p.errors
}

// syntaxError registra un error sintáctico en la posición del token 'tok'.
// Recibe un formato como fmt.Sprintf:
//
//	p.syntaxError(tok, "se esperaba %s", "';'")
//	→ "Error sintáctico [línea 1, columna 7]: se esperaba ';'"
func (p *Parser) syntaxError(tok token.Token, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	p.errors = append(p.errors,
		fmt.Sprintf("Error sintáctico [línea %d, columna %d]: %s", tok.Line, tok.Column, msg))
}

// peekError registra un error cuando el siguiente token no es el esperado.
// Ej: en "int x = 5 }" se esperaba ';' después del 5 pero llegó '}'.
// El error se reporta en la posición del token que SÍ se encontró.
func (p *Parser) peekError(t token.TokenType) {
	p.syntaxError(p.peekToken, "se esperaba %s pero se encontró %s",
		describeType(t), describeToken(p.peekToken))
}

// describeType convierte un tipo de token en texto para los mensajes de
// error: IDENT → "un identificador", SEMICOLON → "';'", WHILE → "'while'".
// Para los símbolos el tipo ya es el propio texto (";", ")"); para las
// palabras reservadas basta pasarlo a minúsculas ("WHILE" → "while").
func describeType(t token.TokenType) string {
	switch t {
	case token.IDENT:
		return "un identificador"
	case token.EOF:
		return "el fin del archivo"
	}
	return "'" + strings.ToLower(string(t)) + "'"
}

// describeToken convierte un token concreto en texto para los mensajes de
// error, usando lo que se escribió en el código: '}', 'x', "hola".
func describeToken(tok token.Token) string {
	switch tok.Type {
	case token.EOF:
		return "el fin del archivo"
	case token.STRING:
		return strconv.Quote(tok.Literal)
	}
	return "'" + tok.Literal + "'"
}

// nextToken avanza una posición: el siguiente pasa a ser el actual y se
// le pide un token nuevo al lexer.
//
// Si el lexer entrega un token ILLEGAL (error léxico), aquí se registra
// como "Error léxico" y se descarta, pidiendo otro token. Así el resto del
// parser nunca ve tokens ILLEGAL y puede seguir revisando la sintaxis del
// programa como si el carácter inválido no estuviera.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	for p.peekToken.Type == token.ILLEGAL {
		p.errors = append(p.errors, fmt.Sprintf("Error léxico [línea %d, columna %d]: %s",
			p.peekToken.Line, p.peekToken.Column, p.peekToken.Literal))
		p.peekToken = p.l.NextToken()
	}
}

// curTokenIs indica si el token ACTUAL es del tipo dado.
func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

// peekTokenIs indica si el SIGUIENTE token es del tipo dado (sin avanzar).
func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// expectPeek es la forma de "exigir" un token según la gramática.
//   - Si el siguiente token es del tipo esperado, avanza (queda como
//     curToken) y devuelve true.
//   - Si no, NO avanza, registra un error y devuelve false; la función que
//     la llamó normalmente abandona y devuelve nil.
//
// Ej: al final de "x = 5" se llama expectPeek(SEMICOLON) para exigir ';'.
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

// peekPrecedence devuelve la precedencia del SIGUIENTE token
// (LOWEST si no es un operador).
func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// curPrecedence devuelve la precedencia del token ACTUAL
// (LOWEST si no es un operador).
func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// synchronize se llama después de un error dentro de una sentencia. Salta
// tokens hasta un punto seguro para seguir parseando la sentencia
// siguiente, y así no reportar errores "en cascada" que solo son
// consecuencia del primero. Se detiene:
//   - en un ';' (fin de la sentencia con error), o
//   - en la '}' que cierra un bloque que se abrió DENTRO de la zona
//     saltada (por ejemplo el cuerpo de un if con error en la condición), o
//   - justo ANTES de una '}' que cierra el bloque de afuera, o del EOF,
//     para que el ciclo que llamó (ParseProgram o parseBlockStatement)
//     los procese normalmente.
//
// Igual que una sentencia, deja curToken en su último token; quien llama
// hace nextToken después.
func (p *Parser) synchronize() {
	depth := 0 // llaves abiertas dentro de la zona que se está saltando
	for !p.curTokenIs(token.EOF) {
		switch p.curToken.Type {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			depth--
			if depth <= 0 {
				return
			}
		case token.SEMICOLON:
			if depth == 0 {
				return
			}
		}
		if depth == 0 && (p.peekTokenIs(token.RBRACE) || p.peekTokenIs(token.EOF)) {
			return
		}
		p.nextToken()
	}
}

// ---------------------------------------------------------------------------
// Programa y sentencias
// ---------------------------------------------------------------------------

// ParseProgram es el PUNTO DE ENTRADA del parser. Parsea sentencias una
// tras otra hasta llegar al fin de la entrada (EOF) y devuelve la raíz del
// AST (*ast.Program), que contiene la lista de sentencias.
//
// Cada parseStatement termina con curToken en el ÚLTIMO token de la
// sentencia (por ejemplo el ';'), por eso después se llama a nextToken
// para pasar al primer token de la siguiente sentencia.
// Si la sentencia agregó errores, antes se llama a synchronize.
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		errorsBefore := len(p.errors)
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		if len(p.errors) > errorsBefore {
			p.synchronize()
		}
		p.nextToken()
	}

	return program
}

// parseStatement decide QUÉ TIPO de sentencia parsear mirando el token
// actual:
//   - un tipo o 'const' → declaración de variable o función (int x = 5;)
//   - 'return'          → retorno (return x;)
//   - 'if', 'while', 'for', 'do' → estructuras de control
//   - 'break', 'continue' → saltos dentro de un ciclo
//   - 'cout'            → impresión (cout << x << endl;)
//   - '{'               → bloque suelto (abre un ámbito nuevo)
//   - 'using'           → using namespace std; (se acepta y se ignora)
//   - ';'               → sentencia vacía, se ignora
//   - otro              → una expresión usada como sentencia (x = 1; f(2);)
//
// Nota sobre el nil (detalle de Go): todas las funciones parseXxx de
// sentencias devuelven el tipo interfaz ast.Statement y usan "return nil"
// cuando hay un error. Si devolvieran un puntero concreto nil (por
// ejemplo (*ast.ReturnStatement)(nil)) y se guardara en una interfaz, la
// interfaz NO sería nil y el "if stmt != nil" de ParseProgram lo dejaría
// pasar.
func (p *Parser) parseStatement() ast.Statement {
	if token.IsType(p.curToken.Type) || p.curTokenIs(token.CONST) {
		return p.parseDeclaration()
	}

	switch p.curToken.Type {
	case token.RETURN:
		return p.parseReturnStatement()
	case token.IF:
		return p.parseIfStatement()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.FOR:
		return p.parseForStatement()
	case token.DO:
		return p.parseDoWhileStatement()
	case token.BREAK:
		return p.parseJumpStatement(&ast.BreakStatement{Token: p.curToken})
	case token.CONTINUE:
		return p.parseJumpStatement(&ast.ContinueStatement{Token: p.curToken})
	case token.COUT:
		return p.parseCoutStatement()
	case token.LBRACE:
		return p.parseBlockStatement()
	case token.USING:
		return p.parseUsingDirective()
	case token.SEMICOLON:
		// Sentencia vacía (por ejemplo ";;"): se ignora.
		return nil
	default:
		return p.parseExpressionStatement()
	}
}

// parseUsingDirective parsea:   using namespace std ;
// El intérprete no tiene espacios de nombres, así que la directiva solo se
// verifica y NO genera ningún nodo (devuelve nil).
//
//	Al entrar: curToken = 'using'
//	Al salir:  curToken = ';'
func (p *Parser) parseUsingDirective() ast.Statement {
	if !p.expectPeek(token.NAMESPACE) {
		return nil
	}
	if !p.expectPeek(token.STD) {
		return nil
	}
	p.expectPeek(token.SEMICOLON)
	return nil
}

// parseReturnStatement parsea:   return <expresión> ;   o   return ;
//
//	Al entrar: curToken = 'return'
//	Al salir:  curToken = ';'
func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	// "return;" sin valor (funciones void): ReturnValue queda nil.
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
		return stmt
	}

	p.nextToken() // se salta el 'return' → curToken = inicio del valor
	stmt.ReturnValue = p.parseExpression(LOWEST)
	if stmt.ReturnValue == nil {
		return nil
	}

	// Como en C++, el ';' es obligatorio.
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return stmt
}

// parseExpressionStatement parsea una expresión que se usa sola como
// sentencia, por ejemplo "x = 10;", "suma(1, 2);" o "i++;".
//
//	Al entrar: curToken = primer token de la expresión
//	Al salir:  curToken = ';'
func (p *Parser) parseExpressionStatement() ast.Statement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)
	if stmt.Expression == nil {
		return nil
	}

	// Como en C++, el ';' es obligatorio.
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return stmt
}

// parseBlockStatement parsea las sentencias que hay entre '{' y '}'.
// Se usa para los cuerpos de funciones y estructuras de control, y para
// los bloques sueltos.
//
//	Al entrar: curToken = '{'
//	Al salir:  curToken = '}'
//
// Si se llega a EOF sin encontrar '}', se registra un error.
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}
	p.blockDepth++

	p.nextToken() // se salta el '{'

	// Mismo esquema que ParseProgram, pero se detiene en '}'.
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		errorsBefore := len(p.errors)
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		if len(p.errors) > errorsBefore {
			p.synchronize()
		}
		p.nextToken()
	}

	if !p.curTokenIs(token.RBRACE) {
		p.syntaxError(p.curToken, "falta '}' para cerrar el bloque abierto en la línea %d",
			block.Token.Line)
	}

	p.blockDepth--
	return block
}

// ---------------------------------------------------------------------------
// Estructuras de control y cout
// ---------------------------------------------------------------------------

// parseBody parsea el CUERPO de un if, else, while, for o do. Como en C++,
// puede ser un bloque entre llaves o UNA sola sentencia:
//
//	if (x) { y = 1; }      cuerpo = el bloque
//	if (x) y = 1;          cuerpo = la sentencia, envuelta en un bloque
//
// Envolver la sentencia sola en un BlockStatement hace que el evaluador
// siempre reciba un bloque, escriba o no las llaves el programador.
//
//	Al entrar: curToken = token ANTERIOR al cuerpo (')' , 'else' o 'do')
//	Al salir:  curToken = último token del cuerpo ('}' o ';')
func (p *Parser) parseBody() *ast.BlockStatement {
	p.nextToken() // curToken = primer token del cuerpo

	if p.curTokenIs(token.LBRACE) {
		return p.parseBlockStatement()
	}

	block := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}
	p.blockDepth++ // la sentencia sola también está "dentro" de un bloque
	stmt := p.parseStatement()
	p.blockDepth--
	if stmt != nil {
		block.Statements = append(block.Statements, stmt)
	}
	return block
}

// parseCondition parsea la condición entre paréntesis de if, while y
// do-while. Como en C++, los paréntesis son obligatorios.
//
//	Al entrar: curToken = 'if' o 'while'
//	Al salir:  curToken = ')'
//
// Devuelve nil si hay un error.
func (p *Parser) parseCondition() ast.Expression {
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // se salta el '(' → curToken = inicio de la condición

	condition := p.parseExpression(LOWEST)
	if condition == nil {
		return nil
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return condition
}

// parseIfStatement parsea un condicional en cualquiera de sus formas:
//
//	if (<cond>) <cuerpo>
//	if (<cond>) <cuerpo> else <cuerpo>
//	if (<cond>) <cuerpo> else if (<cond>) <cuerpo> else <cuerpo>
//
//	Al entrar: curToken = 'if'
//	Al salir:  curToken = último token del último cuerpo
//
// Cómo se representa el "else if": no hay un nodo especial ni código
// especial. Después de 'else' se parsea un cuerpo con parseBody; si ese
// cuerpo empieza con 'if', parseBody lo parsea como una sentencia if
// normal (llamada recursiva a esta función) y lo envuelve en un bloque.
// Es decir:
//
//	if (a) {1} else if (b) {2} else {3}
//	    se convierte en
//	if (a) {1} else { if (b) {2} else {3} }
//
// La recursión permite encadenar cualquier número de "else if".
// Además, un 'else' siempre queda con el if más cercano que no tenga
// else, igual que en C++: en  if (a) if (b) x; else y;  el else es del
// if (b).
func (p *Parser) parseIfStatement() ast.Statement {
	stmt := &ast.IfExpression{Token: p.curToken}

	// Condición: deja curToken en ')'.
	stmt.Condition = p.parseCondition()
	if stmt.Condition == nil {
		return nil
	}

	// Cuerpo "entonces".
	stmt.Consequence = p.parseBody()

	// else opcional. Si no hay, Alternative queda en nil.
	if p.peekTokenIs(token.ELSE) {
		p.nextToken() // curToken = 'else'
		stmt.Alternative = p.parseBody()
	}

	return stmt
}

// parseWhileStatement parsea un ciclo:   while (<cond>) <cuerpo>
//
//	Al entrar: curToken = 'while'
//	Al salir:  curToken = último token del cuerpo
func (p *Parser) parseWhileStatement() ast.Statement {
	stmt := &ast.WhileExpression{Token: p.curToken}

	stmt.Condition = p.parseCondition()
	if stmt.Condition == nil {
		return nil
	}

	stmt.Body = p.parseBody()
	return stmt
}

// parseForStatement parsea un ciclo for de C++:
//
//	for (<inicio>; <condición>; <actualización>) <cuerpo>
//	ej: for (int i = 0; i < 10; i++) { ... }
//
// El <inicio> es una SENTENCIA: una declaración (int i = 0) o una
// expresión (i = 0); se ejecuta una vez. La <condición> es una EXPRESIÓN:
// el ciclo sigue mientras sea verdadera. La <actualización> es una
// EXPRESIÓN que se ejecuta al final de cada vuelta.
//
// Cualquiera de las tres partes puede omitirse, por ejemplo "for (;;) {}"
// (en ese caso el campo correspondiente del nodo queda en nil).
//
//	Al entrar: curToken = 'for'
//	Al salir:  curToken = último token del cuerpo
func (p *Parser) parseForStatement() ast.Statement {
	stmt := &ast.ForExpression{Token: p.curToken}

	// for (
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // se salta el '(' → curToken = inicio de <inicio> o ';'

	// 1. <inicio>. Si curToken ya es ';', se omitió.
	//    parseDeclaration y parseExpressionStatement consumen el ';' final,
	//    así que al volver curToken = ';'.
	if !p.curTokenIs(token.SEMICOLON) {
		if token.IsType(p.curToken.Type) || p.curTokenIs(token.CONST) {
			stmt.Init = p.parseDeclaration()
		} else {
			stmt.Init = p.parseExpressionStatement()
		}
		if stmt.Init == nil {
			return nil
		}
	}
	p.nextToken() // se salta el primer ';' → curToken = inicio de <condición> o ';'

	// 2. <condición>. Si curToken ya es ';', se omitió.
	//    Aquí se usa parseExpression, que NO consume el ';', por eso se
	//    exige con expectPeek.
	if !p.curTokenIs(token.SEMICOLON) {
		stmt.Condition = p.parseExpression(LOWEST)
		if stmt.Condition == nil {
			return nil
		}
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
	}
	p.nextToken() // se salta el segundo ';' → curToken = inicio de <actualización> o ')'

	// 3. <actualización>. Si curToken ya es ')', se omitió.
	if !p.curTokenIs(token.RPAREN) {
		stmt.Update = p.parseExpression(LOWEST)
		if stmt.Update == nil {
			return nil
		}
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}
	// Aquí curToken = ')'.

	// 4. Cuerpo.
	stmt.Body = p.parseBody()
	return stmt
}

// parseDoWhileStatement parsea un ciclo que se ejecuta al menos una vez:
//
//	do <cuerpo> while (<cond>);
//	ej: do { x--; } while (x > 0);
//
//	Al entrar: curToken = 'do'
//	Al salir:  curToken = ';'
func (p *Parser) parseDoWhileStatement() ast.Statement {
	stmt := &ast.DoWhileStatement{Token: p.curToken}

	stmt.Body = p.parseBody()

	if !p.expectPeek(token.WHILE) {
		return nil
	}
	stmt.Condition = p.parseCondition()
	if stmt.Condition == nil {
		return nil
	}

	// A diferencia del while normal, el do-while termina en ';'.
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return stmt
}

// parseJumpStatement termina de parsear  break;  o  continue;  (el nodo
// ya viene creado; solo falta exigir el ';'). El parser NO revisa que
// estén dentro de un ciclo: eso lo detecta el evaluador.
//
//	Al entrar: curToken = 'break' o 'continue'
//	Al salir:  curToken = ';'
func (p *Parser) parseJumpStatement(stmt ast.Statement) ast.Statement {
	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return stmt
}

// parseCoutStatement parsea la impresión en pantalla:
//
//	cout << <expr> << <expr> ... ;
//	ej: cout << "Hola " << nombre << endl;
//
// Cada valor se parsea con precedencia SHIFT: así el parseo de un valor se
// detiene al ver el siguiente "<<" (que tiene esa misma precedencia y no
// es mayor), pero absorbe operadores más fuertes como + o *. Entonces
// "cout << a + b << endl;" imprime a + b. Como en C++, para imprimir una
// comparación hay que usar paréntesis: cout << (a < b);
//
//	Al entrar: curToken = 'cout'
//	Al salir:  curToken = ';'
func (p *Parser) parseCoutStatement() ast.Statement {
	stmt := &ast.CoutStatement{Token: p.curToken}

	// Debe haber al menos un "<<".
	if !p.expectPeek(token.SHIFT_LEFT) {
		return nil
	}

	for {
		p.nextToken() // se salta el '<<' → curToken = inicio del valor
		value := p.parseExpression(SHIFT)
		if value == nil {
			return nil
		}
		stmt.Values = append(stmt.Values, value)

		if !p.peekTokenIs(token.SHIFT_LEFT) {
			break
		}
		p.nextToken() // curToken = '<<'
	}

	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}
	return stmt
}

// ---------------------------------------------------------------------------
// Declaraciones: variables y funciones
// ---------------------------------------------------------------------------

// parseDeclaration parsea todo lo que empieza con un tipo. Como las
// variables y las funciones empiezan igual (int suma ...), se lee el tipo
// y el nombre, y se decide mirando el token siguiente:
//
//	int suma ( ...   → '(' : es una FUNCIÓN  → parseFunctionDeclaration
//	int x = 5;       → otro: es una VARIABLE → parseVarDeclarator
//	int a = 1, b;    → varias variables separadas por ',' → VarDeclarationList
//	const int N = 5; → constante (solo variables)
//
//	Al entrar: curToken = el tipo o 'const'
//	Al salir:  curToken = ';' (variables y prototipos) o '}' (funciones)
func (p *Parser) parseDeclaration() ast.Statement {
	isConst := false
	if p.curTokenIs(token.CONST) {
		isConst = true
		p.nextToken() // se salta 'const' → curToken = el tipo
		if !token.IsType(p.curToken.Type) {
			p.syntaxError(p.curToken, "se esperaba un tipo después de 'const' pero se encontró %s",
				describeToken(p.curToken))
			return nil
		}
	}
	typeTok := p.curToken

	// Después del tipo debe venir un nombre: int x
	if !p.peekTokenIs(token.IDENT) {
		p.syntaxError(p.peekToken, "se esperaba un nombre después del tipo '%s' pero se encontró %s",
			typeTok.Literal, describeToken(p.peekToken))
		return nil
	}
	p.nextToken()
	name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// int suma(  → es una función.
	if p.peekTokenIs(token.LPAREN) {
		if isConst {
			p.syntaxError(typeTok, "'const' solo se puede usar en variables")
			return nil
		}
		return p.parseFunctionDeclaration(typeTok, name)
	}

	// Si no, son una o más variables.
	if typeTok.Type == token.VOID {
		p.syntaxError(typeTok, "la variable '%s' no puede ser de tipo void", name.Value)
		return nil
	}

	first := p.parseVarDeclarator(typeTok, isConst, name)
	if first == nil {
		return nil
	}
	decls := []*ast.VarDeclaration{first}

	// Mientras haya comas, viene otra variable del mismo tipo: , b = 2
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // curToken = ','
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		decl := p.parseVarDeclarator(typeTok, isConst, name)
		if decl == nil {
			return nil
		}
		decls = append(decls, decl)
	}

	if !p.expectPeek(token.SEMICOLON) {
		return nil
	}

	// Con una sola variable no hace falta la lista.
	if len(decls) == 1 {
		return decls[0]
	}
	return &ast.VarDeclarationList{Token: typeTok, Declarations: decls}
}

// parseVarDeclarator parsea lo que va después del nombre de UNA variable
// (sin el ';' final, que lo exige parseDeclaration):
//
//	x                 sin inicializar
//	x = 1 + 2         con valor inicial
//	a[5]              arreglo con tamaño
//	a[3] = {1, 2, 3}  arreglo con tamaño y valores
//	a[] = {1, 2}      arreglo sin tamaño (lo da la cantidad de valores)
//
//	Al entrar: curToken = el nombre de la variable
//	Al salir:  curToken = último token del declarador (nombre, ']', '}' o
//	           último token del valor)
//
// Devuelve un *ast.VarDeclaration concreto (no la interfaz) porque
// parseDeclaration necesita guardarlo en una lista; cuidado: quien lo llama
// revisa el nil antes de convertirlo a ast.Statement.
func (p *Parser) parseVarDeclarator(typeTok token.Token, isConst bool, name *ast.Identifier) *ast.VarDeclaration {
	decl := &ast.VarDeclaration{
		Token: typeTok,
		Type:  typeTok.Literal,
		Const: isConst,
		Name:  name,
	}

	// ¿Es un arreglo?  a[ ... ]
	if p.peekTokenIs(token.LBRACKET) {
		p.nextToken() // curToken = '['
		decl.IsArray = true
		if !p.peekTokenIs(token.RBRACKET) {
			p.nextToken() // curToken = inicio del tamaño
			decl.ArraySize = p.parseExpression(LOWEST)
			if decl.ArraySize == nil {
				return nil
			}
		}
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
	}

	// ¿Tiene valor inicial?  = ...
	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken() // curToken = '='
		if decl.IsArray {
			// Un arreglo se inicializa con una lista entre llaves.
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			decl.Elements = p.parseExpressionList(token.RBRACE)
			if decl.Elements == nil {
				return nil
			}
		} else {
			p.nextToken() // se salta el '=' → curToken = inicio del valor
			decl.Value = p.parseExpression(LOWEST)
			if decl.Value == nil {
				return nil
			}
		}
	}

	// Reglas que C++ también exige.
	if isConst && decl.Value == nil && decl.Elements == nil {
		p.syntaxError(name.Token, "la constante '%s' debe tener un valor inicial", name.Value)
		return nil
	}
	if decl.IsArray && decl.ArraySize == nil && decl.Elements == nil {
		p.syntaxError(name.Token, "el arreglo '%s' necesita un tamaño o una lista de valores", name.Value)
		return nil
	}

	return decl
}

// parseFunctionDeclaration parsea una función o un prototipo, a partir del
// '(' (el tipo de retorno y el nombre ya los leyó parseDeclaration):
//
//	int suma(int a, int b) { return a + b; }    definición
//	int suma(int, int);                         prototipo (Body = nil)
//
//	Al entrar: curToken = el nombre de la función; peekToken = '('
//	Al salir:  curToken = '}' del cuerpo, o ';' si es un prototipo
func (p *Parser) parseFunctionDeclaration(typeTok token.Token, name *ast.Identifier) ast.Statement {
	fn := &ast.FunctionDeclaration{
		Token:      typeTok,
		ReturnType: typeTok.Literal,
		Name:       name,
	}

	// En C++ no se puede definir una función dentro de otra.
	if p.blockDepth > 0 {
		p.syntaxError(name.Token,
			"no se puede declarar la función '%s' dentro de un bloque (solo en el nivel superior)",
			name.Value)
		return nil
	}

	p.nextToken() // curToken = '('
	params, ok := p.parseParams()
	if !ok {
		return nil
	}
	fn.Params = params
	// Aquí curToken = ')'.

	// int f(int);  → prototipo: termina en ';' y no tiene cuerpo.
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
		return fn
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// Una función con cuerpo necesita el nombre de cada parámetro para
	// poder usarlo dentro.
	for i, param := range fn.Params {
		if param.Name == nil {
			p.syntaxError(name.Token, "el parámetro %d de la función '%s' necesita un nombre",
				i+1, name.Value)
			return nil
		}
	}

	fn.Body = p.parseBlockStatement()
	return fn
}

// parseParams parsea la lista de parámetros de una función:
//
//	()                      sin parámetros
//	(int a, double b)       con nombre
//	(int, int)              solo tipos (prototipo)
//	(int v[], int n)        arreglo como parámetro
//
//	Al entrar: curToken = '('
//	Al salir:  curToken = ')'
//
// Devuelve la lista (vacía si no hay parámetros) y true, o nil y false si
// hay un error de sintaxis.
func (p *Parser) parseParams() ([]ast.Param, bool) {
	params := []ast.Param{}

	// Caso especial: función sin parámetros "()".
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return params, true
	}

	for {
		p.nextToken() // curToken = tipo del parámetro
		if !token.IsType(p.curToken.Type) || p.curTokenIs(token.VOID) {
			p.syntaxError(p.curToken, "se esperaba el tipo del parámetro pero se encontró %s",
				describeToken(p.curToken))
			return nil, false
		}
		param := ast.Param{Type: p.curToken.Literal}

		// El nombre es opcional (en los prototipos se puede omitir).
		if p.peekTokenIs(token.IDENT) {
			p.nextToken()
			param.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}

		// int v[]  → el parámetro es un arreglo.
		if p.peekTokenIs(token.LBRACKET) {
			p.nextToken() // curToken = '['
			if !p.expectPeek(token.RBRACKET) {
				return nil, false
			}
			param.IsArray = true
		}

		params = append(params, param)

		// Si no hay coma, se acabó la lista.
		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken() // curToken = ','
	}

	if !p.expectPeek(token.RPAREN) {
		return nil, false
	}
	return params, true
}

// ---------------------------------------------------------------------------
// Núcleo del algoritmo de Pratt
// ---------------------------------------------------------------------------

// parseExpression es la función central del parser de expresiones. Parsea
// una expresión completa respetando la precedencia de los operadores.
//
// El parámetro 'precedence' es la precedencia del operador que está a la
// IZQUIERDA de esta expresión (o LOWEST si no hay ninguno). La idea es:
// "sigue agregando operadores a la derecha mientras aten con MÁS fuerza que
// el operador de la izquierda".
//
// Pasos:
//  1. Se busca la función PREFIJA del token actual y se usa para parsear el
//     lado izquierdo (un número, un identificador, un '-x', un '(...)', ...).
//  2. Mientras el siguiente token sea un operador con precedencia MAYOR que
//     'precedence', se avanza hasta él y se llama a su función INFIJA,
//     pasándole lo parseado hasta ahora como lado izquierdo. El resultado
//     pasa a ser el nuevo lado izquierdo.
//  3. Cuando el siguiente token tiene precedencia menor o igual (o es ';',
//     ')', ',', EOF...), se termina y se devuelve lo construido.
//
// Ejemplo: 1 + 2 * 3   (llamada con LOWEST)
//
//	prefijo de '1'            → izquierda = 1
//	peek '+' (SUM > LOWEST)   → infijo '+': parsea su derecha con SUM
//	    prefijo de '2'        → izquierda = 2
//	    peek '*' (PRODUCT > SUM) → infijo '*': derecha = 3 → (2 * 3)
//	    peek EOF              → se detiene, devuelve (2 * 3)
//	izquierda = (1 + (2 * 3))
//	peek EOF                  → se detiene
//
// Ejemplo: 1 * 2 + 3
//
//	izquierda = 1; infijo '*' parsea su derecha con PRODUCT:
//	    izquierda = 2; peek '+' (SUM) NO es > PRODUCT → devuelve 2
//	izquierda = (1 * 2); peek '+' (SUM > LOWEST) → infijo '+'
//	resultado: ((1 * 2) + 3)
//
// Si alguna parte falla (devuelve nil), se deja de absorber operadores y
// se devuelve nil hacia arriba: el error ya quedó registrado.
//
//	Al entrar: curToken = primer token de la expresión
//	Al salir:  curToken = último token de la expresión
func (p *Parser) parseExpression(precedence int) ast.Expression {
	// Paso 1: lado izquierdo con la función prefija.
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.syntaxError(p.curToken, "se esperaba una expresión pero se encontró %s",
			describeToken(p.curToken))
		return nil
	}
	leftExp := prefix()

	// Paso 2: mientras el siguiente operador ate más fuerte, se absorbe.
	for leftExp != nil && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken() // curToken = el operador

		leftExp = infix(leftExp)
	}

	// Paso 3: no hay más operadores que absorber.
	return leftExp
}

// ---------------------------------------------------------------------------
// Funciones de parseo PREFIJAS
// (curToken = primer token de la expresión al entrar)
// ---------------------------------------------------------------------------

// parseIdentifier parsea un identificador:   x
// Es un solo token, así que no avanza.
func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parseIntegerLiteral parsea un número entero:   42
// El lexer entrega el número como texto ("42"); aquí se convierte a int64.
// Si no cabe en 64 bits (por ejemplo 99999999999999999999) se reporta error.
func (p *Parser) parseIntegerLiteral() ast.Expression {
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.syntaxError(p.curToken, "el número %s es demasiado grande", p.curToken.Literal)
		return nil
	}
	return &ast.IntegerLiteral{Token: p.curToken, Value: value}
}

// parseDoubleLiteral parsea un número con decimales:   3.14
// El lexer ya garantizó el formato (dígitos, un punto, dígitos).
func (p *Parser) parseDoubleLiteral() ast.Expression {
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.syntaxError(p.curToken, "no se pudo convertir %s a double", p.curToken.Literal)
		return nil
	}
	return &ast.DoubleLiteral{Token: p.curToken, Value: value}
}

// parseStringLiteral parsea una cadena:   "hola"
// El lexer ya quitó las comillas y convirtió los escapes.
func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// parseBoolean parsea true o false. El valor es true solo si el token es TRUE.
func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

// parseEndl parsea la palabra endl (fin de línea), que se usa en cout.
func (p *Parser) parseEndl() ast.Expression {
	return &ast.EndlLiteral{Token: p.curToken}
}

// parsePrefixExpression parsea un operador prefijo:   -x   !x   ++x   --x
//
//	Al entrar: curToken = el operador
//	Al salir:  curToken = último token del operando
func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken() // se salta el operador → curToken = inicio del operando

	// El operando se parsea con precedencia PREFIX. Como PREFIX es mayor que
	// PRODUCT y SUM, en "-a * b" el '-' solo toma la 'a': ((-a) * b).
	// En cambio POSTFIX > PREFIX, así que "-a[0]" es (-(a[0])).
	expression.Right = p.parseExpression(PREFIX)
	if expression.Right == nil {
		return nil
	}

	// ++ y -- modifican su operando, así que debe ser una variable.
	if (expression.Operator == "++" || expression.Operator == "--") &&
		!isAssignable(expression.Right) {
		p.syntaxError(expression.Token,
			"el operador %s necesita una variable o un elemento de arreglo, no %s",
			expression.Operator, expression.Right.String())
		return nil
	}

	return expression
}

// parseGroupedExpression parsea una expresión entre paréntesis:   (a + b)
//
//	Al entrar: curToken = '('
//	Al salir:  curToken = ')'
//
// Los paréntesis NO generan un nodo propio en el AST: su único efecto es
// que el contenido se parsea desde LOWEST, "reiniciando" la precedencia.
// Así "(1 + 2) * 3" queda ((1 + 2) * 3).
func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken() // se salta el '('

	exp := p.parseExpression(LOWEST)
	if exp == nil {
		return nil
	}

	// Debe cerrarse con ')'.
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

// ---------------------------------------------------------------------------
// Funciones de parseo INFIJAS y POSTFIJAS
// (reciben el lado izquierdo ya parseado; curToken = el operador al entrar)
// ---------------------------------------------------------------------------

// parseInfixExpression parsea un operador binario:
//
//	a + b   a - b   a * b   a / b   a % b
//	a == b  a != b  a < b   a > b   a <= b  a >= b
//	a && b  a || b  a << b  a >> b
//
//	Al entrar: curToken = el operador; 'left' = lo que estaba a la izquierda
//	Al salir:  curToken = último token del lado derecho
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	// El lado derecho se parsea con la precedencia de ESTE operador. Así,
	// en "1 + 2 + 3", el primer '+' al parsear su derecha ve otro '+' con
	// la MISMA precedencia, no lo absorbe (no es mayor), y el resultado es
	// ((1 + 2) + 3): asociatividad por la IZQUIERDA, como en C++.
	precedence := p.curPrecedence()

	p.nextToken() // se salta el operador → curToken = inicio del lado derecho
	expression.Right = p.parseExpression(precedence)
	if expression.Right == nil {
		return nil
	}

	return expression
}

// parseAssignExpression parsea una asignación simple o compuesta:
//
//	x = 5   arr[0] = 1   total += arr[i]   n -= 1
//
//	Al entrar: curToken = el operador ('=', '+=', ...); 'left' = el destino
//	Al salir:  curToken = último token del valor
//
// Se implementa como operador infijo con la precedencia más baja (ASSIGN),
// así todo lo que está a la derecha se evalúa antes: x = 1 + 2 es
// x = (1 + 2).
//
// Es asociativa por la DERECHA: a = b = 3 es a = (b = 3). Se logra
// parseando el valor con ASSIGN - 1: así el siguiente '=' (de precedencia
// ASSIGN) SÍ es "mayor" y queda dentro del valor.
func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	expression := &ast.AssignExpression{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}

	// Solo tiene sentido asignar a una variable (x) o a una posición de un
	// arreglo (a[0]). Asignar a "5" o a "a + b" es un error.
	if !isAssignable(left) {
		p.syntaxError(p.curToken, "no se puede asignar a %s: se esperaba una variable o un elemento de arreglo",
			left.String())
		return nil
	}

	p.nextToken() // se salta el operador → curToken = inicio del valor
	expression.Value = p.parseExpression(ASSIGN - 1)
	if expression.Value == nil {
		return nil
	}

	return expression
}

// parsePostfixExpression parsea un incremento o decremento postfijo:
//
//	i++   arr[0]--
//
//	Al entrar: curToken = '++' o '--'; 'left' = la variable
//	Al salir:  curToken = '++' o '--' (el operador es el último token)
func (p *Parser) parsePostfixExpression(left ast.Expression) ast.Expression {
	if !isAssignable(left) {
		p.syntaxError(p.curToken,
			"el operador %s necesita una variable o un elemento de arreglo, no %s",
			p.curToken.Literal, left.String())
		return nil
	}
	return &ast.PostfixExpression{Token: p.curToken, Left: left, Operator: p.curToken.Literal}
}

// parseCallExpression parsea una llamada a función:   suma(1, 2)
// El '(' actúa como operador infijo: a su izquierda está la función que se
// llama.
//
//	Al entrar: curToken = '('; 'function' = lo que estaba a la izquierda
//	Al salir:  curToken = ')'
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}

	// Argumentos: expresiones separadas por comas hasta ')'.
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	if exp.Arguments == nil {
		return nil
	}

	return exp
}

// parseIndexExpression parsea un acceso por índice:   arr[0]
// El '[' actúa como operador con la precedencia más alta (POSTFIX), así en
// "a * b[0]" el índice se aplica solo a b: (a * (b[0])).
//
//	Al entrar: curToken = '['; 'left' = el arreglo
//	Al salir:  curToken = ']'
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken() // se salta el '[' → curToken = inicio del índice
	exp.Index = p.parseExpression(LOWEST)
	if exp.Index == nil {
		return nil
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

// parseMemberExpression parsea el acceso a un miembro con punto:   s.size
// Como '.' tiene precedencia POSTFIX (igual que la llamada), en s.size()
// primero se arma s.size y luego el '(' lo convierte en una llamada.
//
//	Al entrar: curToken = '.'; 'left' = el objeto
//	Al salir:  curToken = el nombre del miembro
func (p *Parser) parseMemberExpression(left ast.Expression) ast.Expression {
	exp := &ast.MemberExpression{Token: p.curToken, Object: left}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	exp.Member = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}

// parseExpressionList parsea expresiones separadas por comas hasta
// encontrar el token 'end'. Se reutiliza para:
//
//   - argumentos de una llamada:         f(1, 2, 3)    → end = ')'
//
//   - valores iniciales de un arreglo:   {1, 2, 3}     → end = '}'
//
//     Al entrar: curToken = el delimitador de apertura ('(' o '{')
//     Al salir:  curToken = 'end'
//
// Devuelve una lista vacía si no hay elementos, y nil si hay un error.
func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	// Caso especial: lista vacía "()" o "{}".
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	// Primer elemento.
	p.nextToken() // curToken = inicio del primer elemento
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}
	list = append(list, first)

	// Mientras haya comas, viene otro elemento.
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // curToken = ','
		p.nextToken() // curToken = inicio de la siguiente expresión
		next := p.parseExpression(LOWEST)
		if next == nil {
			return nil
		}
		list = append(list, next)
	}

	// La lista debe cerrarse con 'end'.
	if !p.expectPeek(end) {
		return nil
	}

	return list
}

// isAssignable indica si una expresión puede estar a la izquierda de una
// asignación o recibir ++/--: solo una variable (x) o un elemento de
// arreglo (a[i]).
func isAssignable(exp ast.Expression) bool {
	switch exp.(type) {
	case *ast.Identifier, *ast.IndexExpression:
		return true
	}
	return false
}
