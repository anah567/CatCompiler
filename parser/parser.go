// Paquete parser implementa el ANALIZADOR SINTÁCTICO del compilador.
//
// ¿Qué hace?
// Recibe la secuencia de tokens que produce el lexer y construye con ellos
// un Árbol de Sintaxis Abstracta (AST). El AST representa la ESTRUCTURA
// del programa: qué operación se aplica sobre qué operandos, qué sentencias
// hay dentro de qué bloques, etc.
//
//	Código:  let x = 1 + 2 * 3;
//	Tokens:  LET IDENT(x) = INT(1) + INT(2) * INT(3) ;
//	AST:     LetStatement x
//	         └── Valor: (+)
//	             ├── 1
//	             └── (*)
//	                 ├── 2
//	                 └── 3
//
// ¿Cómo lo hace? — Parser de Pratt
// Se usa la técnica de Vaughan Pratt ("Top Down Operator Precedence"), que
// es un parser descendente recursivo en el que:
//
//  1. Cada TIPO DE TOKEN tiene asociadas hasta dos funciones de parseo:
//     - una función PREFIJA (prefixParseFn): se usa cuando el token aparece
//     al INICIO de una expresión. Ej: el '-' en "-5", un número "5",
//     un identificador "x", un '(' en "(a + b)".
//     - una función INFIJA (infixParseFn): se usa cuando el token aparece
//     ENTRE dos expresiones. Ej: el '+' en "a + b", el '(' en "f(x)".
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
// Regla general: una función de parseo de expresiones empieza con curToken
// en el primer token de la expresión y termina con curToken en el ÚLTIMO
// token de esa expresión (no en el siguiente).
package parser

import (
	"fmt"
	"polaobesahermosa/ast"
	"polaobesahermosa/lexer"
	"polaobesahermosa/token"
	"strconv"
)

// ---------------------------------------------------------------------------
// Precedencias
// ---------------------------------------------------------------------------

// Niveles de precedencia, de MENOR a MAYOR. iota asigna números
// consecutivos: LOWEST = 1, ASSIGN = 2, EQUALS = 3, ... INDEX = 10.
// (El '_' descarta el valor 0.)
//
// Un operador con mayor precedencia se agrupa primero. Por ejemplo, como
// PRODUCT > SUM, en "1 + 2 * 3" la multiplicación se resuelve antes.
const (
	_           int = iota
	LOWEST          // precedencia mínima: se usa al empezar a parsear una expresión
	ASSIGN          // x = 5          (la más baja de los operadores)
	EQUALS          // ==  !=
	LESSGREATER     // >  <  >=  <=
	SUM             // +  -
	PRODUCT         // *  /  %
	PREFIX          // -X  !X         (operadores prefijos)
	EXPONENT        // **             (más fuerte que el prefijo: -2**2 = -(2**2))
	CALL            // miFuncion(X)
	INDEX           // arreglo[indice] (la más alta de todas)
)

// precedences asocia cada token que puede actuar como OPERADOR INFIJO con
// su nivel de precedencia. Los tokens que no están en este mapa (por
// ejemplo ';', ')', '{') tienen precedencia LOWEST, lo que hace que
// parseExpression se detenga al encontrarlos.
//
// Nótese que '(' y '[' aparecen aquí: cuando van DESPUÉS de una expresión
// se comportan como operadores infijos (llamada "f(x)" e índice "a[0]").
var precedences = map[token.TokenType]int{
	token.ASSIGN:   ASSIGN,
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.LTE:      LESSGREATER,
	token.GTE:      LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
	token.MOD:      PRODUCT,
	token.POW:      EXPONENT,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
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
	errors []string     // errores de sintaxis encontrados (no se detiene en el primero)

	curToken  token.Token // token que se está analizando en este momento
	peekToken token.Token // token siguiente; permite "mirar adelante" para decidir

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
	p.registerPrefix(token.IDENT, p.parseIdentifier)         // x
	p.registerPrefix(token.INT, p.parseIntegerLiteral)       // 5
	p.registerPrefix(token.BANG, p.parsePrefixExpression)    // !x
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)   // -x
	p.registerPrefix(token.TRUE, p.parseBoolean)             // true
	p.registerPrefix(token.FALSE, p.parseBoolean)            // false
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression) // (a + b)
	p.registerPrefix(token.IF, p.parseIfExpression)          // if (...) {...}
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral) // function(a) {...}
	p.registerPrefix(token.STRING, p.parseStringLiteral)     // "hola"
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)    // [1, 2, 3]
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)       // {"a": 1}
	p.registerPrefix(token.FOR, p.parseForLiteral)           // for (...;...;...) {...}
	p.registerPrefix(token.WHILE, p.parseWhileLiteral)       // while (...) {...}

	// --- Funciones INFIJAS: tokens que aparecen ENTRE dos expresiones ---
	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)     // a + b
	p.registerInfix(token.MINUS, p.parseInfixExpression)    // a - b
	p.registerInfix(token.SLASH, p.parseInfixExpression)    // a / b
	p.registerInfix(token.ASTERISK, p.parseInfixExpression) // a * b
	p.registerInfix(token.MOD, p.parseInfixExpression)      // a % b
	p.registerInfix(token.EQ, p.parseInfixExpression)       // a == b
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)   // a != b
	p.registerInfix(token.LT, p.parseInfixExpression)       // a < b
	p.registerInfix(token.GT, p.parseInfixExpression)       // a > b
	p.registerInfix(token.LTE, p.parseInfixExpression)      // a <= b
	p.registerInfix(token.GTE, p.parseInfixExpression)      // a >= b
	p.registerInfix(token.POW, p.parseInfixExpression)      // a ** b
	p.registerInfix(token.ASSIGN, p.parseAssignExpression)  // a = b
	p.registerInfix(token.LPAREN, p.parseCallExpression)    // f(a, b)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression) // a[i]

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

// Errors devuelve los errores de sintaxis encontrados durante el parseo.
// Quien use el parser (por ejemplo el REPL) debe revisar esta lista después
// de llamar a ParseProgram: si no está vacía, el AST puede estar incompleto.
func (p *Parser) Errors() []string {
	return p.errors
}

// peekError registra un error cuando el siguiente token no es el esperado.
// Ej: en "let = 5" se esperaba IDENT después de 'let' pero llegó '='.
func (p *Parser) peekError(t token.TokenType) {
	p.errors = append(p.errors,
		fmt.Sprintf("se esperaba %s, pero se encontró %s (%q)",
			t, p.peekToken.Type, p.peekToken.Literal))
}

// noPrefixParseFnError registra un error cuando el token actual no puede
// iniciar una expresión, porque no tiene función prefija registrada.
// Ej: una expresión que empieza con ')' o con '*'.
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	p.errors = append(p.errors,
		fmt.Sprintf("no se puede iniciar una expresión con %s (%q)",
			t, p.curToken.Literal))
}

// nextToken avanza una posición: el siguiente pasa a ser el actual y se
// le pide un token nuevo al lexer.
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
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
// Ej: después de 'let' se llama expectPeek(IDENT) para exigir un nombre.
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
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseStatement decide QUÉ TIPO de sentencia parsear mirando el token
// actual:
//   - 'let'    → sentencia de declaración  (let x = 5;)
//   - 'return' → sentencia de retorno      (return x;)
//   - ';'      → sentencia vacía, se ignora
//   - otro     → una expresión usada como sentencia (x + 1; f(2); if ...)
//
// Nota sobre el nil (detalle de Go): parseLetStatement devuelve un
// *ast.LetStatement. Si ese puntero es nil y se devuelve directamente como
// ast.Statement (una interfaz), la interfaz NO es nil: guarda "tipo
// *LetStatement con valor nil". Entonces el "if stmt != nil" de
// ParseProgram lo dejaría pasar. Por eso aquí se revisa explícitamente.
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		if stmt := p.parseLetStatement(); stmt != nil {
			return stmt
		}
		return nil
	case token.RETURN:
		return p.parseReturnStatement()
	case token.SEMICOLON:
		// Sentencia vacía (por ejemplo ";;"): se ignora.
		return nil
	default:
		return p.parseExpressionStatement()
	}
}

// parseLetStatement parsea:   let <identificador> = <expresión> ;
//
//	Al entrar: curToken = 'let'
//	Al salir:  curToken = ';' (si lo hay) o el último token de la expresión
//
// Devuelve nil si falta el nombre o el '='.
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	// Después de 'let' debe venir un nombre: let x
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Después del nombre debe venir '=': let x =
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken() // se salta el '=' → curToken = primer token del valor

	// El valor es cualquier expresión. Se usa LOWEST porque no hay ningún
	// operador a la izquierda que compita por los operandos.
	stmt.Value = p.parseExpression(LOWEST)

	// El punto y coma es opcional: si está, se consume.
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseReturnStatement parsea:   return <expresión> ;
//
//	Al entrar: curToken = 'return'
//	Al salir:  curToken = ';' (si lo hay) o el último token de la expresión
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken() // se salta el 'return' → curToken = inicio del valor

	stmt.ReturnValue = p.parseExpression(LOWEST)

	// Punto y coma opcional.
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseExpressionStatement parsea una expresión que se usa sola como
// sentencia, por ejemplo "x + 10;", "suma(1, 2);" o un "if".
//
//	Al entrar: curToken = primer token de la expresión
//	Al salir:  curToken = ';' (si lo hay) o el último token de la expresión
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	// Punto y coma opcional (así en el REPL se puede escribir "5 + 5").
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseBlockStatement parsea las sentencias que hay entre '{' y '}'.
// Se usa para los cuerpos de if, while, for y funciones.
//
//	Al entrar: curToken = '{'
//	Al salir:  curToken = '}'
//
// Si se llega a EOF sin encontrar '}', se registra un error.
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken() // se salta el '{'

	// Mismo esquema que ParseProgram, pero se detiene en '}'.
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	if !p.curTokenIs(token.RBRACE) {
		p.errors = append(p.errors, "bloque sin cerrar: falta '}'")
	}

	return block
}

// ---------------------------------------------------------------------------
// Núcleo del algoritmo de Pratt
// ---------------------------------------------------------------------------

// parseExpression es la función central del parser. Parsea una expresión
// completa respetando la precedencia de los operadores.
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
//     ')', '}', EOF...), se termina y se devuelve lo construido.
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
//	Al entrar: curToken = primer token de la expresión
//	Al salir:  curToken = último token de la expresión
func (p *Parser) parseExpression(precedence int) ast.Expression {
	// Paso 1: lado izquierdo con la función prefija.
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	// Paso 2: mientras el siguiente operador ate más fuerte, se absorbe.
	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
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
	lit := &ast.IntegerLiteral{Token: p.curToken}

	// Base 0: strconv deduce la base por el prefijo (en la práctica,
	// el lexer solo entrega dígitos decimales).
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.errors = append(p.errors,
			fmt.Sprintf("no se pudo convertir %q a entero", p.curToken.Literal))
		return nil
	}

	lit.Value = value
	return lit
}

// parseStringLiteral parsea una cadena:   "hola"
// El lexer ya quitó las comillas, el literal del token es solo el contenido.
func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// parseBoolean parsea true o false. El valor es true solo si el token es TRUE.
func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

// parsePrefixExpression parsea un operador prefijo:   -x   !x
//
//	Al entrar: curToken = '-' o '!'
//	Al salir:  curToken = último token del operando
func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken() // se salta el operador → curToken = inicio del operando

	// El operando se parsea con precedencia PREFIX. Como PREFIX es mayor que
	// PRODUCT y SUM, en "-a * b" el '-' solo toma la 'a': ((-a) * b).
	// En cambio EXPONENT > PREFIX, así que "-2 ** 2" es (-(2 ** 2)).
	expression.Right = p.parseExpression(PREFIX)

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

	// Debe cerrarse con ')'.
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

// parseCondition parsea la condición de if / elif / while.
// Se acepta con o sin paréntesis: "if (x > 1) {" o "if x > 1 {".
// (Si hay paréntesis, los maneja parseGroupedExpression como a cualquier
// expresión agrupada.)
//
//	Al entrar: curToken = 'if', 'elif' o 'while'
//	Al salir:  curToken = '{' que abre el bloque
//
// Devuelve nil si después de la condición no viene '{'.
func (p *Parser) parseCondition() ast.Expression {
	p.nextToken() // se salta la palabra reservada
	condition := p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	return condition
}

// parseIfExpression parsea un condicional en cualquiera de sus formas:
//
//	if (<cond>) { ... }
//	if (<cond>) { ... } else { ... }
//	if (<cond>) { ... } elif (<cond>) { ... } else { ... }
//
//	Al entrar: curToken = 'if' (o 'elif', en la llamada recursiva)
//	Al salir:  curToken = '}' del último bloque
//
// Cómo se representa el elif: no hay un nodo especial. Un elif es
// equivalente a un else que contiene otro if, es decir:
//
//	if (a) { 1 } elif (b) { 2 } else { 3 }
//	    es lo mismo que
//	if (a) { 1 } else { if (b) { 2 } else { 3 } }
//
// Por eso, al encontrar 'elif', se llama recursivamente a esta misma
// función (que parsea "elif (b) { 2 } else { 3 }" como un if completo) y
// el resultado se mete dentro de un bloque que se usa como Alternative.
// La recursión permite encadenar cualquier número de elif.
func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	// Condición: deja curToken en '{'.
	expression.Condition = p.parseCondition()
	if expression.Condition == nil {
		return nil
	}

	// Bloque "entonces": deja curToken en '}'.
	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELIF) {
		p.nextToken() // curToken = 'elif'
		elifTok := p.curToken

		// Llamada recursiva: el elif se parsea igual que un if.
		nested := p.parseIfExpression()
		if nested == nil {
			return nil
		}

		// Se envuelve el if anidado en un bloque para usarlo como "sino".
		expression.Alternative = &ast.BlockStatement{
			Token: elifTok,
			Statements: []ast.Statement{
				&ast.ExpressionStatement{Token: elifTok, Expression: nested},
			},
		}
	} else if p.peekTokenIs(token.ELSE) {
		p.nextToken() // curToken = 'else'

		// Después de 'else' debe venir '{'.
		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}
	// Si no hay elif ni else, Alternative queda en nil.

	return expression
}

// parseWhileLiteral parsea un ciclo:   while (<cond>) { ... }
//
//	Al entrar: curToken = 'while'
//	Al salir:  curToken = '}' del cuerpo
func (p *Parser) parseWhileLiteral() ast.Expression {
	expression := &ast.WhileExpression{Token: p.curToken}

	// Condición: deja curToken en '{'.
	expression.Condition = p.parseCondition()
	if expression.Condition == nil {
		return nil
	}

	// Cuerpo: deja curToken en '}'.
	expression.Body = p.parseBlockStatement()

	return expression
}

// parseForLiteral parsea un ciclo for al estilo C:
//
//		for (<inicio>; <condición>; <actualización>) { ... }
//		ej: for (let i = 0; i < 10; i = i + 1) { ... }
//
//	  - <inicio> es una SENTENCIA (normalmente un let); se ejecuta una vez.
//	  - <condición> es una EXPRESIÓN; el ciclo sigue mientras sea verdadera.
//	  - <actualización> es una EXPRESIÓN; se ejecuta al final de cada vuelta.
//
// Cualquiera de las tres partes puede omitirse, por ejemplo "for (;;) {}"
// (en ese caso el campo correspondiente del nodo queda en nil).
//
//	Al entrar: curToken = 'for'
//	Al salir:  curToken = '}' del cuerpo
func (p *Parser) parseForLiteral() ast.Expression {
	expression := &ast.ForExpression{Token: p.curToken}

	// for (
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken() // se salta el '(' → curToken = inicio de <inicio> o ';'

	// 1. <inicio>. Si curToken ya es ';', se omitió.
	//    parseStatement consume el ';' final si existe, así que al volver
	//    curToken debe ser ';'. Si no lo es, falta el separador.
	if !p.curTokenIs(token.SEMICOLON) {
		expression.Init = p.parseStatement()
		if expression.Init == nil {
			return nil
		}
		if !p.curTokenIs(token.SEMICOLON) {
			p.peekError(token.SEMICOLON)
			return nil
		}
	}
	p.nextToken() // se salta el primer ';' → curToken = inicio de <condición> o ';'

	// 2. <condición>. Si curToken ya es ';', se omitió.
	//    Aquí se usa parseExpression (no parseStatement), que NO consume el
	//    ';', por eso se exige con expectPeek.
	if !p.curTokenIs(token.SEMICOLON) {
		expression.Condition = p.parseExpression(LOWEST)
		if !p.expectPeek(token.SEMICOLON) {
			return nil
		}
	}
	p.nextToken() // se salta el segundo ';' → curToken = inicio de <actualización> o ')'

	// 3. <actualización>. Si curToken ya es ')', se omitió.
	if !p.curTokenIs(token.RPAREN) {
		expression.Update = p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}
	// Aquí curToken = ')'.

	// 4. Cuerpo: debe empezar con '{'.
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	expression.Body = p.parseBlockStatement()

	return expression
}

// parseFunctionLiteral parsea una función:   function(<parámetros>) { ... }
//
//	Al entrar: curToken = 'function'
//	Al salir:  curToken = '}' del cuerpo
//
// Las funciones son expresiones (valores), por eso pueden asignarse:
// let suma = function(a, b) { return a + b; };
func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	// function(
	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	// a, b)  → deja curToken en ')'
	lit.Parameters = p.parseFunctionParameters()
	if lit.Parameters == nil {
		return nil
	}

	// { ... }
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

// parseFunctionParameters parsea la lista de parámetros:   (x, y, z)
// Los parámetros solo pueden ser identificadores.
//
//	Al entrar: curToken = '('
//	Al salir:  curToken = ')'
//
// Devuelve una lista vacía (no nil) si la función no tiene parámetros, y
// nil si hay un error de sintaxis; así quien llama puede distinguir ambos.
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	// Caso especial: función sin parámetros "()".
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	// Primer parámetro.
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	identifiers = append(identifiers,
		&ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

	// Mientras haya comas, viene otro parámetro.
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // curToken = ','
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		identifiers = append(identifiers,
			&ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}

	// La lista se cierra con ')'.
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

// parseArrayLiteral parsea un arreglo:   [1, 2 + 3, "a"]
//
//	Al entrar: curToken = '['
//	Al salir:  curToken = ']'
func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	// Los elementos son expresiones separadas por comas, hasta ']'.
	array.Elements = p.parseExpressionList(token.RBRACKET)
	if array.Elements == nil {
		return nil
	}

	return array
}

// parseHashLiteral parsea un hash (diccionario):   {"clave": valor, x: 2}
// Tanto las claves como los valores pueden ser cualquier expresión.
//
//	Al entrar: curToken = '{'
//	Al salir:  curToken = '}'
//
// Nota: como '{' al inicio de una expresión se interpreta como hash, un
// bloque suelto "{ ... }" fuera de if/while/for/function se trata como hash.
func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = []ast.HashPair{}

	// Se repite mientras no se llegue a la '}' de cierre.
	// (Si el hash es vacío "{}", el ciclo no se ejecuta ni una vez.)
	for !p.peekTokenIs(token.RBRACE) {
		// Clave
		p.nextToken() // curToken = inicio de la clave
		key := p.parseExpression(LOWEST)

		// Separador ':'
		if !p.expectPeek(token.COLON) {
			return nil
		}

		// Valor
		p.nextToken() // se salta el ':' → curToken = inicio del valor
		value := p.parseExpression(LOWEST)

		hash.Pairs = append(hash.Pairs, ast.HashPair{Key: key, Value: value})

		// Después de cada pareja debe venir ',' (hay más parejas) o
		// '}' (fin del hash). Si viene ',' expectPeek la consume.
		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	// Se consume la '}' de cierre.
	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

// ---------------------------------------------------------------------------
// Funciones de parseo INFIJAS
// (reciben el lado izquierdo ya parseado; curToken = el operador al entrar)
// ---------------------------------------------------------------------------

// parseInfixExpression parsea un operador binario:
//
//	a + b   a - b   a * b   a / b   a % b   a ** b
//	a == b  a != b  a < b   a > b   a <= b  a >= b
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
	// ((1 + 2) + 3): asociatividad por la IZQUIERDA, que es lo normal.
	precedence := p.curPrecedence()

	// La potencia es la excepción: es asociativa por la DERECHA, como en
	// matemáticas: 2 ** 3 ** 2 = 2 ** (3 ** 2) = 2 ** 9.
	// Para lograrlo se baja un nivel la precedencia al parsear la derecha;
	// así el siguiente '**' sí es "mayor" y se agrupa primero a la derecha.
	if p.curTokenIs(token.POW) {
		precedence--
	}

	p.nextToken() // se salta el operador → curToken = inicio del lado derecho
	expression.Right = p.parseExpression(precedence)

	return expression
}

// parseAssignExpression parsea una asignación:   x = 5   arr[0] = 1
//
//	Al entrar: curToken = '='; 'left' = el destino
//	Al salir:  curToken = último token del valor
//
// Se implementa como operador infijo con la precedencia más baja (ASSIGN),
// así todo lo que está a la derecha se evalúa antes: x = 1 + 2 es
// x = (1 + 2).
//
// Es asociativa por la DERECHA: a = b = 3 es a = (b = 3). Se logra
// parseando el valor con ASSIGN - 1 (el mismo truco que con '**').
func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	expression := &ast.AssignExpression{Token: p.curToken, Left: left}

	// Solo tiene sentido asignar a una variable (x) o a una posición de un
	// arreglo/hash (a[0]). Asignar a "5" o a "a + b" es un error.
	switch left.(type) {
	case *ast.Identifier, *ast.IndexExpression:
		// destino válido
	default:
		p.errors = append(p.errors,
			fmt.Sprintf("no se puede asignar a %s", left.String()))
		return nil
	}

	p.nextToken() // se salta el '=' → curToken = inicio del valor
	expression.Value = p.parseExpression(ASSIGN - 1)

	return expression
}

// parseCallExpression parsea una llamada a función:   suma(1, 2)
// El '(' actúa como operador infijo: a su izquierda está la función que se
// llama (un identificador como "suma" o incluso un literal de función).
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

// parseIndexExpression parsea un acceso por índice:   arr[0]   h["a"]
// El '[' actúa como operador infijo con la precedencia más alta (INDEX),
// así en "a * b[0]" el índice se aplica solo a b: (a * (b[0])).
//
//	Al entrar: curToken = '['; 'left' = el arreglo/hash
//	Al salir:  curToken = ']'
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken() // se salta el '[' → curToken = inicio del índice
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

// parseExpressionList parsea expresiones separadas por comas hasta
// encontrar el token 'end'. Se reutiliza para:
//
//   - argumentos de una llamada:   f(1, 2, 3)   → end = ')'
//
//   - elementos de un arreglo:     [1, 2, 3]    → end = ']'
//
//     Al entrar: curToken = el delimitador de apertura ('(' o '[')
//     Al salir:  curToken = 'end'
//
// Devuelve una lista vacía si no hay elementos, y nil si hay un error.
func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	// Caso especial: lista vacía "()" o "[]".
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	// Primer elemento.
	p.nextToken() // curToken = inicio del primer elemento
	list = append(list, p.parseExpression(LOWEST))

	// Mientras haya comas, viene otro elemento.
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // curToken = ','
		p.nextToken() // curToken = inicio de la siguiente expresión
		list = append(list, p.parseExpression(LOWEST))
	}

	// La lista debe cerrarse con 'end'.
	if !p.expectPeek(end) {
		return nil
	}

	return list
}
