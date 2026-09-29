// Paquete ast define los nodos del ÁRBOL DE SINTAXIS ABSTRACTA (AST).
//
// ¿Qué es el AST?
// Es la estructura de datos que construye el parser para representar el
// programa. Cada nodo es una construcción del lenguaje (una sentencia let,
// una suma, un if...) y sus hijos son las partes que la componen.
//
//	Código:  let x = 1 + 2;
//
//	Program
//	└── LetStatement            (Name = x)
//	    └── InfixExpression     (Operator = "+")
//	        ├── IntegerLiteral  1
//	        └── IntegerLiteral  2
//
// Se llama "abstracto" porque NO guarda todos los detalles del texto: los
// paréntesis, los punto y coma y los espacios desaparecen, y solo queda la
// estructura. Por ejemplo "(1 + 2)" y "1 + 2" producen el mismo árbol.
//
// Sentencias vs. expresiones
// Los nodos se dividen en dos familias:
//   - Statement (sentencia): una instrucción que NO produce un valor,
//     por ejemplo "let x = 5;" o "return x;".
//   - Expression (expresión): algo que SÍ produce un valor, por ejemplo
//     "5", "x", "a + b", "suma(1, 2)". En este lenguaje también son
//     expresiones el if, while, for y las funciones.
//
// Cada familia es una interfaz de Go con un método "marcador"
// (statementNode / expressionNode) que no hace nada: solo sirve para que el
// compilador de Go impida, por ejemplo, poner una sentencia donde se espera
// una expresión.
package ast

import (
	"bytes"
	"polaobesahermosa/token"
	"strings"
)

// Node es la interfaz que implementan TODOS los nodos del árbol.
type Node interface {
	// TokenLiteral devuelve el literal del token con el que empieza el
	// nodo (por ejemplo "let" para un LetStatement). Útil para depurar.
	TokenLiteral() string
	// String devuelve el nodo convertido de nuevo a código. Los operadores
	// se imprimen entre paréntesis para que se vea cómo se agruparon:
	// "1 + 2 * 3" → "(1 + (2 * 3))". Se usa para depurar y en las pruebas.
	String() string
}

// Statement es un nodo que representa una sentencia (no produce valor).
type Statement interface {
	Node // "herencia" por composición: todo Statement también es un Node
	statementNode()
}

// Expression es un nodo que representa una expresión (produce un valor).
type Expression interface {
	Node // "herencia" por composición: toda Expression también es un Node
	expressionNode()
}

// ---------------------------------------------------------------------------
// Programa
// ---------------------------------------------------------------------------

// Program es la RAÍZ del AST. Un programa es simplemente una lista de
// sentencias que se ejecutan en orden. El parser siempre devuelve un
// *Program (aunque esté vacío).
type Program struct {
	Statements []Statement
}

// TokenLiteral devuelve el literal del primer token del programa, o ""
// si el programa está vacío.
func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// String concatena la representación de todas las sentencias.
// Se usa bytes.Buffer para ir armando el texto de forma eficiente.
func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// ---------------------------------------------------------------------------
// Sentencias
// ---------------------------------------------------------------------------

// LetStatement representa la declaración de una variable:
//
//	let <Name> = <Value>;
//	ej: let edad = 20;
type LetStatement struct {
	Token token.Token // el token LET
	Name  *Identifier // nombre de la variable (edad)
	Value Expression  // expresión cuyo valor se guarda (20)
}

func (ls *LetStatement) statementNode() {}
func (ls *LetStatement) TokenLiteral() string {
	return ls.Token.Literal
}
func (ls *LetStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ") // "let "
	out.WriteString(ls.Name.String())        // "edad"
	out.WriteString(" = ")

	// Value puede ser nil si hubo un error de sintaxis.
	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}

// ReturnStatement representa el retorno de un valor desde una función:
//
//	return <ReturnValue>;
//	ej: return a + b;
type ReturnStatement struct {
	Token       token.Token // el token 'return'
	ReturnValue Expression  // valor que se devuelve (a + b)
}

func (rs *ReturnStatement) statementNode() {}
func (rs *ReturnStatement) TokenLiteral() string {
	return rs.Token.Literal
}
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	out.WriteString(";")

	return out.String()
}

// ExpressionStatement "envuelve" una expresión para poder usarla como
// sentencia. Es necesario porque un programa es una lista de SENTENCIAS,
// pero es válido escribir una expresión suelta en una línea:
//
//	x + 10;
//	suma(1, 2);
//	if (x > 1) { ... }
type ExpressionStatement struct {
	Token      token.Token // el primer token de la expresión
	Expression Expression  // la expresión envuelta
}

func (es *ExpressionStatement) statementNode() {}
func (es *ExpressionStatement) TokenLiteral() string {
	return es.Token.Literal
}
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement es una lista de sentencias entre llaves. Se usa como
// cuerpo de if, while, for y funciones:
//
//	{ let y = x * 2; return y; }
type BlockStatement struct {
	Token      token.Token // el token '{'
	Statements []Statement // sentencias dentro del bloque, en orden
}

func (bs *BlockStatement) statementNode() {}
func (bs *BlockStatement) TokenLiteral() string {
	return bs.Token.Literal
}
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	out.WriteString("{ ")
	for _, s := range bs.Statements {
		out.WriteString(s.String())
		out.WriteString(" ")
	}
	out.WriteString("}")

	return out.String()
}

// ---------------------------------------------------------------------------
// Expresiones simples (literales e identificadores)
// Son las "hojas" del árbol: no tienen hijos.
// ---------------------------------------------------------------------------

// Identifier es el nombre de una variable o función:   x   suma
// Es una expresión porque al usarlo produce un valor (el de la variable).
type Identifier struct {
	Token token.Token // el token IDENT
	Value string      // el nombre ("x")
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) TokenLiteral() string {
	return i.Token.Literal
}
func (i *Identifier) String() string {
	return i.Value
}

// IntegerLiteral es un número entero escrito en el código:   42
type IntegerLiteral struct {
	Token token.Token // el token INT (Literal = "42", como texto)
	Value int64       // el valor ya convertido a número (42)
}

func (il *IntegerLiteral) expressionNode() {}
func (il *IntegerLiteral) TokenLiteral() string {
	return il.Token.Literal
}
func (il *IntegerLiteral) String() string {
	return il.Token.Literal
}

// Boolean es un valor lógico:   true   false
type Boolean struct {
	Token token.Token // el token TRUE o FALSE
	Value bool
}

func (b *Boolean) expressionNode() {}
func (b *Boolean) TokenLiteral() string {
	return b.Token.Literal
}
func (b *Boolean) String() string {
	return b.Token.Literal
}

// StringLiteral es una cadena de texto:   "hola"
type StringLiteral struct {
	Token token.Token // el token STRING
	Value string      // el contenido sin comillas (hola)
}

func (sl *StringLiteral) expressionNode() {}
func (sl *StringLiteral) TokenLiteral() string {
	return sl.Token.Literal
}

// String vuelve a poner las comillas para que se vea que es una cadena.
func (sl *StringLiteral) String() string {
	return "\"" + sl.Value + "\""
}

// ---------------------------------------------------------------------------
// Operadores
// ---------------------------------------------------------------------------

// PrefixExpression es un operador que va ANTES de su operando:
//
//	<Operator><Right>
//	ej: -5   !true   -x
type PrefixExpression struct {
	Token    token.Token // el token del operador ('-' o '!')
	Operator string      // "-" o "!"
	Right    Expression  // el operando
}

func (pe *PrefixExpression) expressionNode() {}
func (pe *PrefixExpression) TokenLiteral() string {
	return pe.Token.Literal
}
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	// Los paréntesis muestran cómo se agrupó: "-a * b" → "((-a) * b)".
	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

// InfixExpression es un operador binario, que va ENTRE sus dos operandos:
//
//	<Left> <Operator> <Right>
//	ej: 5 + 3   a * b   x == y   n ** 2
//
// Como Left y Right son a su vez expresiones, pueden ser otros
// InfixExpression; así se forman árboles como (1 + (2 * 3)).
type InfixExpression struct {
	Token    token.Token // el token del operador ('+', '*', '==', ...)
	Left     Expression  // operando izquierdo
	Operator string      // el operador como texto
	Right    Expression  // operando derecho
}

func (ie *InfixExpression) expressionNode() {}
func (ie *InfixExpression) TokenLiteral() string {
	return ie.Token.Literal
}
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	// Los paréntesis muestran la precedencia con la que se agrupó.
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")

	return out.String()
}

// AssignExpression representa la asignación de un nuevo valor:
//
//	<Left> = <Value>
//	ej: x = x + 1    arr[0] = 5
//
// A diferencia de LetStatement (que DECLARA una variable), esta cambia el
// valor de algo que ya existe. Es una expresión (no una sentencia) para
// poder usarla, por ejemplo, en la parte de actualización de un for.
// El parser solo permite que Left sea un Identifier o un IndexExpression.
type AssignExpression struct {
	Token token.Token // el token '='
	Left  Expression  // destino: variable (x) o posición (arr[0])
	Value Expression  // nuevo valor
}

func (ae *AssignExpression) expressionNode() {}
func (ae *AssignExpression) TokenLiteral() string {
	return ae.Token.Literal
}
func (ae *AssignExpression) String() string {
	return ae.Left.String() + " = " + ae.Value.String()
}

// ---------------------------------------------------------------------------
// Estructuras de control
// ---------------------------------------------------------------------------

// IfExpression representa un condicional:
//
//	if (<Condition>) <Consequence> else <Alternative>
//	ej: if (x > 0) { 1 } else { 0 }
//
// No existe un nodo "Elif": un elif se representa como un Alternative que
// contiene otro IfExpression. Es decir:
//
//	if (a) {1} elif (b) {2} else {3}  ≡  if (a) {1} else { if (b) {2} else {3} }
type IfExpression struct {
	Token       token.Token     // el token 'if' (o 'elif' si es un if anidado)
	Condition   Expression      // condición a evaluar
	Consequence *BlockStatement // bloque que se ejecuta si es verdadera
	Alternative *BlockStatement // bloque del else/elif; nil si no hay
}

func (ie *IfExpression) expressionNode() {}
func (ie *IfExpression) TokenLiteral() string {
	return ie.Token.Literal
}
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if ")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

// WhileExpression representa un ciclo que se repite mientras la condición
// sea verdadera:
//
//	while (<Condition>) <Body>
//	ej: while (x > 0) { x = x - 1 }
type WhileExpression struct {
	Token     token.Token     // el token 'while'
	Condition Expression      // se evalúa antes de cada vuelta
	Body      *BlockStatement // cuerpo del ciclo
}

func (we *WhileExpression) expressionNode() {}
func (we *WhileExpression) TokenLiteral() string {
	return we.Token.Literal
}
func (we *WhileExpression) String() string {
	var out bytes.Buffer

	out.WriteString("while ")
	out.WriteString(we.Condition.String())
	out.WriteString(" ")
	out.WriteString(we.Body.String())

	return out.String()
}

// ForExpression representa un ciclo for al estilo C:
//
//	for (<Init>; <Condition>; <Update>) <Body>
//	ej: for (let i = 0; i < 10; i = i + 1) { ... }
//
// Significado:
//  1. Init se ejecuta una sola vez, antes de empezar.
//  2. Antes de cada vuelta se evalúa Condition; si es falsa, el ciclo termina.
//  3. Se ejecuta Body.
//  4. Se ejecuta Update y se vuelve al paso 2.
//
// Cualquiera de las tres partes del encabezado puede omitirse (queda nil):
// "for (;;) { ... }" es un ciclo infinito.
type ForExpression struct {
	Token     token.Token     // el token 'for'
	Init      Statement       // inicialización: let i = 0  (es sentencia)
	Condition Expression      // condición:      i < 10
	Update    Expression      // actualización:  i = i + 1
	Body      *BlockStatement // cuerpo del ciclo
}

func (fe *ForExpression) expressionNode() {}
func (fe *ForExpression) TokenLiteral() string {
	return fe.Token.Literal
}
func (fe *ForExpression) String() string {
	var out bytes.Buffer

	out.WriteString("for (")
	if fe.Init != nil {
		// LetStatement.String() termina en ';'; se quita para no
		// imprimir ";;" junto con el separador que se agrega abajo.
		out.WriteString(strings.TrimSuffix(fe.Init.String(), ";"))
	}
	out.WriteString("; ")
	if fe.Condition != nil {
		out.WriteString(fe.Condition.String())
	}
	out.WriteString("; ")
	if fe.Update != nil {
		out.WriteString(fe.Update.String())
	}
	out.WriteString(") ")
	out.WriteString(fe.Body.String())

	return out.String()
}

// ---------------------------------------------------------------------------
// Funciones
// ---------------------------------------------------------------------------

// FunctionLiteral representa la DEFINICIÓN de una función:
//
//	function(<Parameters>) <Body>
//	ej: function(a, b) { return a + b; }
//
// Las funciones son valores (expresiones), por eso no tienen nombre propio:
// se les da nombre guardándolas en una variable con let.
type FunctionLiteral struct {
	Token      token.Token     // el token 'function'
	Parameters []*Identifier   // nombres de los parámetros (a, b)
	Body       *BlockStatement // cuerpo de la función
}

func (fl *FunctionLiteral) expressionNode() {}
func (fl *FunctionLiteral) TokenLiteral() string {
	return fl.Token.Literal
}
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	// Se juntan los nombres de los parámetros separados por ", ".
	params := []string{}
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}

	out.WriteString(fl.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fl.Body.String())

	return out.String()
}

// CallExpression representa una LLAMADA a una función:
//
//	<Function>(<Arguments>)
//	ej: suma(1, 2 * 3)
//	    function(x) { x * 2 }(5)   ← se puede llamar un literal directamente
type CallExpression struct {
	Token     token.Token  // el token '('
	Function  Expression   // lo que se llama: Identifier o FunctionLiteral
	Arguments []Expression // valores que se pasan (1, 2 * 3)
}

func (ce *CallExpression) expressionNode() {}
func (ce *CallExpression) TokenLiteral() string {
	return ce.Token.Literal
}
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

// ---------------------------------------------------------------------------
// Estructuras de datos
// ---------------------------------------------------------------------------

// ArrayLiteral representa un arreglo escrito en el código:
//
//	[<Elements>]
//	ej: [1, 2 + 3, "hola"]
type ArrayLiteral struct {
	Token    token.Token  // el token '['
	Elements []Expression // los elementos, en orden
}

func (al *ArrayLiteral) expressionNode() {}
func (al *ArrayLiteral) TokenLiteral() string {
	return al.Token.Literal
}
func (al *ArrayLiteral) String() string {
	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// IndexExpression representa el acceso a un elemento por su índice/clave:
//
//	<Left>[<Index>]
//	ej: arr[0]   h["nombre"]   matriz[i][j]
type IndexExpression struct {
	Token token.Token // el token '['
	Left  Expression  // lo que se indexa (arr)
	Index Expression  // el índice o la clave (0)
}

func (ie *IndexExpression) expressionNode() {}
func (ie *IndexExpression) TokenLiteral() string {
	return ie.Token.Literal
}
func (ie *IndexExpression) String() string {
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

// HashPair es una pareja clave: valor dentro de un HashLiteral.
type HashPair struct {
	Key   Expression
	Value Expression
}

// HashLiteral representa un hash (diccionario) escrito en el código:
//
//	{<clave>: <valor>, ...}
//	ej: {"nombre": "Pola", "edad": 3}
//
// Las parejas se guardan en una lista (y no en un map de Go) para conservar
// el orden en que se escribieron; un map de Go las recorrería en orden
// aleatorio y String() cambiaría en cada ejecución.
type HashLiteral struct {
	Token token.Token // el token '{'
	Pairs []HashPair  // parejas clave: valor, en orden
}

func (hl *HashLiteral) expressionNode() {}
func (hl *HashLiteral) TokenLiteral() string {
	return hl.Token.Literal
}
func (hl *HashLiteral) String() string {
	pairs := []string{}
	for _, pair := range hl.Pairs {
		pairs = append(pairs, pair.Key.String()+": "+pair.Value.String())
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}
