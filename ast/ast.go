// Paquete ast define los nodos del ÁRBOL DE SINTAXIS ABSTRACTA (AST).
//
// ¿Qué es el AST?
// Es la estructura de datos que construye el parser para representar el
// programa. Cada nodo es una construcción del lenguaje (una declaración,
// una suma, un if...) y sus hijos son las partes que la componen.
//
//	Código:  int x = 1 + 2;
//
//	Program
//	└── VarDeclaration          (Type = "int", Name = x)
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
//     por ejemplo "int x = 5;", "return x;", un if o un while.
//   - Expression (expresión): algo que SÍ produce un valor, por ejemplo
//     "5", "x", "a + b", "suma(1, 2)", "i++".
//
// Cada familia es una interfaz de Go con un método "marcador"
// (statementNode / expressionNode) que no hace nada: solo sirve para que el
// compilador de Go impida, por ejemplo, poner una sentencia donde se espera
// una expresión.
//
// El contrato completo de los nodos (campos y ejemplos) está en
// docs/CONTRATO_AST.md.
package ast

import (
	"bytes"
	"polaobesahermosa/token"
	"strconv"
	"strings"
)

// Node es la interfaz que implementan TODOS los nodos del árbol.
type Node interface {
	// TokenLiteral devuelve el literal del token con el que empieza el
	// nodo (por ejemplo "int" para una declaración). Útil para depurar.
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

// Program es la RAÍZ del AST. Un programa es una lista de sentencias de
// nivel superior: funciones, variables globales y (para que el REPL siga
// funcionando) también sentencias sueltas. El parser siempre devuelve un
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

// String une la representación de todas las sentencias separadas por un
// espacio.
func (p *Program) String() string {
	parts := []string{}
	for _, s := range p.Statements {
		parts = append(parts, s.String())
	}
	return strings.Join(parts, " ")
}

// ---------------------------------------------------------------------------
// Declaraciones
// ---------------------------------------------------------------------------

// VarDeclaration representa la declaración de UNA variable con su tipo:
//
//	int edad = 20;          Type="int", Name=edad, Value=20
//	int x;                  Value=nil (sin inicializar)
//	const int N = 5;        Const=true
//	int a[5];               IsArray=true, ArraySize=5
//	int a[3] = {1, 2, 3};   IsArray=true, ArraySize=3, Elements=[1 2 3]
//	int a[] = {1, 2};       IsArray=true, ArraySize=nil, Elements=[1 2]
//
// Si en una misma línea se declaran varias variables (int a = 1, b = 2;)
// el parser crea un VarDeclarationList con un VarDeclaration por cada una.
type VarDeclaration struct {
	Token     token.Token  // el token del tipo ('int', 'double', ...)
	Type      string       // nombre del tipo: "int", "double", "bool", "string"
	Const     bool         // true si se escribió 'const' antes del tipo
	Name      *Identifier  // nombre de la variable
	IsArray   bool         // true si se declaró con corchetes: a[...]
	ArraySize Expression   // tamaño entre corchetes; nil si no se escribió
	Value     Expression   // valor inicial de una variable simple; nil si no hay
	Elements  []Expression // valores entre llaves de un arreglo; nil si no hay
}

func (vd *VarDeclaration) statementNode() {}
func (vd *VarDeclaration) TokenLiteral() string {
	return vd.Token.Literal
}

// String devuelve, por ejemplo, "const int N = 5;" o "int a[3] = {1, 2, 3};".
func (vd *VarDeclaration) String() string {
	prefix := ""
	if vd.Const {
		prefix = "const "
	}
	return prefix + vd.Type + " " + vd.declaratorString() + ";"
}

// declaratorString devuelve la parte de la declaración que va DESPUÉS del
// tipo: "a[3] = {1, 2, 3}" o "x = 5". Se reutiliza en VarDeclarationList.
func (vd *VarDeclaration) declaratorString() string {
	var out bytes.Buffer

	out.WriteString(vd.Name.String())
	if vd.IsArray {
		out.WriteString("[")
		if vd.ArraySize != nil {
			out.WriteString(vd.ArraySize.String())
		}
		out.WriteString("]")
	}
	if vd.Value != nil {
		out.WriteString(" = " + vd.Value.String())
	}
	if vd.Elements != nil {
		out.WriteString(" = {" + joinExpressions(vd.Elements) + "}")
	}

	return out.String()
}

// VarDeclarationList agrupa varias declaraciones escritas en una sola
// sentencia con el mismo tipo:
//
//	int a = 1, b = 2, c;
//
// equivale a  int a = 1; int b = 2; int c;  (en ese orden y en el mismo
// ámbito). Solo se usa cuando hay DOS o más variables; con una sola el
// parser devuelve directamente un VarDeclaration.
type VarDeclarationList struct {
	Token        token.Token       // el token del tipo
	Declarations []*VarDeclaration // una por variable, en orden
}

func (vl *VarDeclarationList) statementNode() {}
func (vl *VarDeclarationList) TokenLiteral() string {
	return vl.Token.Literal
}

// String devuelve, por ejemplo, "int a = 1, b = 2;".
func (vl *VarDeclarationList) String() string {
	parts := []string{}
	for _, d := range vl.Declarations {
		parts = append(parts, d.declaratorString())
	}
	prefix := ""
	if len(vl.Declarations) > 0 && vl.Declarations[0].Const {
		prefix = "const "
	}
	return prefix + vl.Token.Literal + " " + strings.Join(parts, ", ") + ";"
}

// Param es un parámetro de una función: su tipo y su nombre.
//
//	int suma(int a, int b[])   →   {Type:"int" Name:a}  {Type:"int" Name:b IsArray:true}
//
// En un prototipo (int f(int);) el nombre se puede omitir y Name queda nil.
// No es un nodo del AST (no implementa Node): es solo un dato de
// FunctionDeclaration.
type Param struct {
	Type    string      // "int", "double", "bool" o "string"
	Name    *Identifier // nombre del parámetro; nil solo en prototipos
	IsArray bool        // true si se escribió con corchetes: int a[]
}

// String devuelve, por ejemplo, "int a" o "int a[]" o "int".
func (pa Param) String() string {
	s := pa.Type
	if pa.Name != nil {
		s += " " + pa.Name.String()
	}
	if pa.IsArray {
		s += "[]"
	}
	return s
}

// FunctionDeclaration representa la DEFINICIÓN de una función con nombre:
//
//	int suma(int a, int b) { return a + b; }
//	void saluda() { cout << "hola"; }
//	int main() { ... return 0; }
//
// También representa un PROTOTIPO (declaración sin cuerpo):
//
//	int suma(int, int);      →   Body = nil
type FunctionDeclaration struct {
	Token      token.Token     // el token del tipo de retorno
	ReturnType string          // "int", "double", "bool", "string" o "void"
	Name       *Identifier     // nombre de la función (suma)
	Params     []Param         // parámetros en orden; vacío si no hay
	Body       *BlockStatement // cuerpo; nil si es solo un prototipo
}

func (fd *FunctionDeclaration) statementNode() {}
func (fd *FunctionDeclaration) TokenLiteral() string {
	return fd.Token.Literal
}

// String devuelve, por ejemplo, "int suma(int a, int b) { return (a + b); }".
func (fd *FunctionDeclaration) String() string {
	params := []string{}
	for _, pa := range fd.Params {
		params = append(params, pa.String())
	}

	head := fd.ReturnType + " " + fd.Name.String() +
		"(" + strings.Join(params, ", ") + ")"

	if fd.Body == nil {
		return head + ";" // prototipo
	}
	return head + " " + fd.Body.String()
}

// ---------------------------------------------------------------------------
// Sentencias
// ---------------------------------------------------------------------------

// ReturnStatement representa el retorno desde una función:
//
//	return a + b;     ReturnValue = (a + b)
//	return;           ReturnValue = nil (funciones void)
type ReturnStatement struct {
	Token       token.Token // el token 'return'
	ReturnValue Expression  // valor que se devuelve; nil en "return;"
}

func (rs *ReturnStatement) statementNode() {}
func (rs *ReturnStatement) TokenLiteral() string {
	return rs.Token.Literal
}
func (rs *ReturnStatement) String() string {
	if rs.ReturnValue == nil {
		return "return;"
	}
	return "return " + rs.ReturnValue.String() + ";"
}

// ExpressionStatement "envuelve" una expresión para poder usarla como
// sentencia. Es necesario porque un programa es una lista de SENTENCIAS,
// pero es válido escribir una expresión sola terminada en ';':
//
//	x = 10;
//	suma(1, 2);
//	i++;
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
		return es.Expression.String() + ";"
	}
	return ";"
}

// BlockStatement es una lista de sentencias entre llaves. Es el cuerpo de
// funciones, if, while, for y do-while, y también puede escribirse solo
// para abrir un ámbito nuevo:  { int y = 2; }
//
// Cuando el cuerpo de un if/else/while/for/do es UNA sola sentencia sin
// llaves (if (x) return 1;), el parser la envuelve en un BlockStatement;
// así el evaluador siempre recibe un bloque.
type BlockStatement struct {
	Token      token.Token // el token '{' (o el primer token de la sentencia envuelta)
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

// CoutStatement representa la impresión en pantalla:
//
//	cout << "Hola " << nombre << endl;
//
// Values guarda, en orden, cada expresión que va después de un "<<".
// endl aparece en la lista como un *EndlLiteral.
type CoutStatement struct {
	Token  token.Token  // el token 'cout'
	Values []Expression // lo que se imprime, en orden
}

func (cs *CoutStatement) statementNode() {}
func (cs *CoutStatement) TokenLiteral() string {
	return cs.Token.Literal
}

// String devuelve, por ejemplo, "cout << "Hola " << nombre << endl;".
func (cs *CoutStatement) String() string {
	var out bytes.Buffer

	out.WriteString("cout")
	for _, v := range cs.Values {
		out.WriteString(" << " + v.String())
	}
	out.WriteString(";")

	return out.String()
}

// BreakStatement representa  break;  (sale del ciclo más interno).
type BreakStatement struct {
	Token token.Token // el token 'break'
}

func (bs *BreakStatement) statementNode() {}
func (bs *BreakStatement) TokenLiteral() string {
	return bs.Token.Literal
}
func (bs *BreakStatement) String() string {
	return "break;"
}

// ContinueStatement representa  continue;  (salta a la siguiente vuelta
// del ciclo más interno).
type ContinueStatement struct {
	Token token.Token // el token 'continue'
}

func (cs *ContinueStatement) statementNode() {}
func (cs *ContinueStatement) TokenLiteral() string {
	return cs.Token.Literal
}
func (cs *ContinueStatement) String() string {
	return "continue;"
}

// ---------------------------------------------------------------------------
// Estructuras de control
//
// Nota: IfExpression, WhileExpression y ForExpression conservan el nombre
// que tenían en el proyecto base para no romper el código de los
// compañeros, pero en C++ son SENTENCIAS: ahora implementan Statement y el
// parser las pone directamente en la lista de sentencias (ya no van dentro
// de un ExpressionStatement).
// ---------------------------------------------------------------------------

// IfExpression representa un condicional:
//
//	if (<Condition>) <Consequence> else <Alternative>
//	ej: if (x > 0) { y = 1; } else { y = 0; }
//
// No existe un nodo "ElseIf": un "else if" es un else cuyo bloque contiene
// otro IfExpression. Es decir:
//
//	if (a) {..} else if (b) {..} else {..}
//	  ≡
//	if (a) {..} else { if (b) {..} else {..} }
type IfExpression struct {
	Token       token.Token     // el token 'if'
	Condition   Expression      // condición a evaluar
	Consequence *BlockStatement // bloque que se ejecuta si es verdadera
	Alternative *BlockStatement // bloque del else; nil si no hay else
}

func (ie *IfExpression) statementNode() {}
func (ie *IfExpression) TokenLiteral() string {
	return ie.Token.Literal
}
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if (")
	out.WriteString(ie.Condition.String())
	out.WriteString(") ")
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
//	ej: while (x > 0) { x--; }
type WhileExpression struct {
	Token     token.Token     // el token 'while'
	Condition Expression      // se evalúa antes de cada vuelta
	Body      *BlockStatement // cuerpo del ciclo
}

func (we *WhileExpression) statementNode() {}
func (we *WhileExpression) TokenLiteral() string {
	return we.Token.Literal
}
func (we *WhileExpression) String() string {
	return "while (" + we.Condition.String() + ") " + we.Body.String()
}

// DoWhileStatement representa un ciclo que se ejecuta AL MENOS una vez y
// luego se repite mientras la condición sea verdadera:
//
//	do <Body> while (<Condition>);
//	ej: do { x--; } while (x > 0);
type DoWhileStatement struct {
	Token     token.Token     // el token 'do'
	Body      *BlockStatement // cuerpo del ciclo
	Condition Expression      // se evalúa DESPUÉS de cada vuelta
}

func (dw *DoWhileStatement) statementNode() {}
func (dw *DoWhileStatement) TokenLiteral() string {
	return dw.Token.Literal
}
func (dw *DoWhileStatement) String() string {
	return "do " + dw.Body.String() + " while (" + dw.Condition.String() + ");"
}

// ForExpression representa un ciclo for de C++:
//
//	for (<Init>; <Condition>; <Update>) <Body>
//	ej: for (int i = 0; i < 10; i++) { ... }
//
// Significado:
//  1. Init se ejecuta una sola vez, antes de empezar.
//  2. Antes de cada vuelta se evalúa Condition; si es falsa, el ciclo termina.
//  3. Se ejecuta Body.
//  4. Se ejecuta Update y se vuelve al paso 2.
//
// Init es una SENTENCIA: un *VarDeclaration (int i = 0), un
// *VarDeclarationList (int i = 0, j = 5) o un *ExpressionStatement (i = 0).
// Cualquiera de las tres partes del encabezado puede omitirse (queda nil):
// "for (;;) { ... }" es un ciclo infinito.
type ForExpression struct {
	Token     token.Token     // el token 'for'
	Init      Statement       // inicialización: int i = 0
	Condition Expression      // condición:      i < 10
	Update    Expression      // actualización:  i++
	Body      *BlockStatement // cuerpo del ciclo
}

func (fe *ForExpression) statementNode() {}
func (fe *ForExpression) TokenLiteral() string {
	return fe.Token.Literal
}
func (fe *ForExpression) String() string {
	var out bytes.Buffer

	out.WriteString("for (")
	if fe.Init != nil {
		// Init.String() termina en ';'; se quita para no imprimir ";;"
		// junto con el separador que se agrega abajo.
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

// DoubleLiteral es un número con parte decimal:   3.14   10.0
type DoubleLiteral struct {
	Token token.Token // el token DOUBLE (Literal = "3.14", como texto)
	Value float64     // el valor ya convertido a número (3.14)
}

func (dl *DoubleLiteral) expressionNode() {}
func (dl *DoubleLiteral) TokenLiteral() string {
	return dl.Token.Literal
}
func (dl *DoubleLiteral) String() string {
	return dl.Token.Literal
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

// StringLiteral es una cadena de texto:   "hola\n"
// Value ya trae los escapes convertidos: "\n" es un salto de línea real.
type StringLiteral struct {
	Token token.Token // el token STRING
	Value string      // el contenido sin comillas
}

func (sl *StringLiteral) expressionNode() {}
func (sl *StringLiteral) TokenLiteral() string {
	return sl.Token.Literal
}

// String vuelve a poner las comillas y a escribir los escapes (\n, \t, ...)
// para que el resultado se vea como en el código fuente.
func (sl *StringLiteral) String() string {
	return strconv.Quote(sl.Value)
}

// EndlLiteral representa la palabra endl (fin de línea) en un cout:
//
//	cout << x << endl;
//
// Al evaluarse debe producir un salto de línea "\n".
type EndlLiteral struct {
	Token token.Token // el token 'endl'
}

func (el *EndlLiteral) expressionNode() {}
func (el *EndlLiteral) TokenLiteral() string {
	return el.Token.Literal
}
func (el *EndlLiteral) String() string {
	return "endl"
}

// ---------------------------------------------------------------------------
// Operadores
// ---------------------------------------------------------------------------

// PrefixExpression es un operador que va ANTES de su operando:
//
//	<Operator><Right>
//	ej: -5   !ok   ++i   --i
//
// Con ++ y -- el operando siempre es un Identifier o un IndexExpression
// (el parser lo verifica). ++i incrementa i y produce el valor NUEVO.
type PrefixExpression struct {
	Token    token.Token // el token del operador
	Operator string      // "-", "!", "++" o "--"
	Right    Expression  // el operando
}

func (pe *PrefixExpression) expressionNode() {}
func (pe *PrefixExpression) TokenLiteral() string {
	return pe.Token.Literal
}
func (pe *PrefixExpression) String() string {
	// Los paréntesis muestran cómo se agrupó: "-a * b" → "((-a) * b)".
	return "(" + pe.Operator + pe.Right.String() + ")"
}

// PostfixExpression es un operador que va DESPUÉS de su operando:
//
//	<Left><Operator>
//	ej: i++   arr[0]--
//
// El operando siempre es un Identifier o un IndexExpression (el parser lo
// verifica). i++ incrementa i pero produce el valor ANTERIOR.
type PostfixExpression struct {
	Token    token.Token // el token del operador ('++' o '--')
	Left     Expression  // el operando
	Operator string      // "++" o "--"
}

func (pe *PostfixExpression) expressionNode() {}
func (pe *PostfixExpression) TokenLiteral() string {
	return pe.Token.Literal
}
func (pe *PostfixExpression) String() string {
	return "(" + pe.Left.String() + pe.Operator + ")"
}

// InfixExpression es un operador binario, que va ENTRE sus dos operandos:
//
//	<Left> <Operator> <Right>
//	ej: 5 + 3   a * b   x == y   a && b   a || b   x << 2
//
// Como Left y Right son a su vez expresiones, pueden ser otros
// InfixExpression; así se forman árboles como (1 + (2 * 3)).
// && y || también son InfixExpression: el evaluador decide si hace
// evaluación en cortocircuito.
type InfixExpression struct {
	Token    token.Token // el token del operador ('+', '*', '==', '&&', ...)
	Left     Expression  // operando izquierdo
	Operator string      // el operador como texto
	Right    Expression  // operando derecho
}

func (ie *InfixExpression) expressionNode() {}
func (ie *InfixExpression) TokenLiteral() string {
	return ie.Token.Literal
}
func (ie *InfixExpression) String() string {
	// Los paréntesis muestran la precedencia con la que se agrupó.
	return "(" + ie.Left.String() + " " + ie.Operator + " " + ie.Right.String() + ")"
}

// AssignExpression representa la asignación de un nuevo valor:
//
//	<Left> <Operator> <Value>
//	ej: x = x + 1    arr[0] = 5    total += arr[i]
//
// A diferencia de VarDeclaration (que DECLARA una variable), esta cambia el
// valor de algo que ya existe. Es una expresión (no una sentencia) para
// poder usarla, por ejemplo, en la parte de actualización de un for o
// encadenada: a = b = 3.
// El parser solo permite que Left sea un Identifier o un IndexExpression.
//
// Operator es "=" o uno compuesto: "+=", "-=", "*=", "/=", "%=".
// x += 2 significa x = x + 2.
type AssignExpression struct {
	Token    token.Token // el token del operador de asignación
	Left     Expression  // destino: variable (x) o posición (arr[0])
	Operator string      // "=", "+=", "-=", "*=", "/=" o "%="
	Value    Expression  // valor de la derecha
}

func (ae *AssignExpression) expressionNode() {}
func (ae *AssignExpression) TokenLiteral() string {
	return ae.Token.Literal
}
func (ae *AssignExpression) String() string {
	// Entre paréntesis para que se vea la asociatividad: (a = (b = 3)).
	return "(" + ae.Left.String() + " " + ae.Operator + " " + ae.Value.String() + ")"
}

// ---------------------------------------------------------------------------
// Llamadas, índices y miembros
// ---------------------------------------------------------------------------

// CallExpression representa una LLAMADA a una función:
//
//	<Function>(<Arguments>)
//	ej: suma(1, 2 * 3)     factorial(n - 1)     s.size()
//
// Function es un Identifier (suma) o, en una llamada a método, un
// MemberExpression (s.size).
type CallExpression struct {
	Token     token.Token  // el token '('
	Function  Expression   // lo que se llama
	Arguments []Expression // valores que se pasan (1, 2 * 3)
}

func (ce *CallExpression) expressionNode() {}
func (ce *CallExpression) TokenLiteral() string {
	return ce.Token.Literal
}
func (ce *CallExpression) String() string {
	return ce.Function.String() + "(" + joinExpressions(ce.Arguments) + ")"
}

// IndexExpression representa el acceso a un elemento de un arreglo:
//
//	<Left>[<Index>]
//	ej: arr[0]   arr[i + 1]
//
// Se usa tanto para leer (x = arr[i]) como para escribir (arr[i] = x;
// en ese caso es el Left de un AssignExpression).
type IndexExpression struct {
	Token token.Token // el token '['
	Left  Expression  // lo que se indexa (arr)
	Index Expression  // el índice (0)
}

func (ie *IndexExpression) expressionNode() {}
func (ie *IndexExpression) TokenLiteral() string {
	return ie.Token.Literal
}
func (ie *IndexExpression) String() string {
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

// MemberExpression representa el acceso a un miembro con punto:
//
//	<Object>.<Member>
//	ej: s.size     (en  s.size()  es la Function de un CallExpression)
type MemberExpression struct {
	Token  token.Token // el token '.'
	Object Expression  // lo que está a la izquierda del punto (s)
	Member *Identifier // el nombre del miembro (size)
}

func (me *MemberExpression) expressionNode() {}
func (me *MemberExpression) TokenLiteral() string {
	return me.Token.Literal
}
func (me *MemberExpression) String() string {
	return me.Object.String() + "." + me.Member.String()
}

// ---------------------------------------------------------------------------
// Utilidades
// ---------------------------------------------------------------------------

// joinExpressions une varias expresiones separadas por ", ".
// Se usa para argumentos de llamadas y elementos de arreglos.
func joinExpressions(list []Expression) string {
	parts := []string{}
	for _, e := range list {
		parts = append(parts, e.String())
	}
	return strings.Join(parts, ", ")
}
