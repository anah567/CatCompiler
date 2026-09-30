// Package objects contiene los valores que existen mientras corre un programa.
//
// El AST describe qué escribió el estudiante; un Object representa el
// resultado de ejecutarlo. Por ejemplo, IntegerLiteral(2) es sintaxis y
// &Integer{Value: 2} es el valor que usa el evaluador.
package objects

import (
	"fmt"
	"strings"

	"polaobesahermosa/ast"
)

// ObjectType permite distinguir valores sin depender de su texto impreso.
type ObjectType string

const (
	INTEGER_OBJ      ObjectType = "INTEGER"
	BOOLEAN_OBJ      ObjectType = "BOOLEAN"
	STRING_OBJ       ObjectType = "STRING"
	NULL_OBJ         ObjectType = "NULL"
	RETURN_VALUE_OBJ ObjectType = "RETURN_VALUE"
	ERROR_OBJ        ObjectType = "ERROR"
	FUNCTION_OBJ     ObjectType = "FUNCTION"
	ARRAY_OBJ        ObjectType = "ARRAY"
	HASH_OBJ         ObjectType = "HASH"
)

// Object es la interfaz común de todos los valores del intérprete.
// Type sirve para comprobar operaciones; Inspect sirve para mostrarlos.
type Object interface {
	Type() ObjectType
	Inspect() string
}

// Integer guarda enteros con signo de 64 bits, igual que el AST.
type Integer struct{ Value int64 }

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

// Boolean guarda un valor lógico. Se reutilizan True y False para evitar
// crear otro objeto cada vez que se evalúa una comparación.
type Boolean struct{ Value bool }

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

var (
	True  = &Boolean{Value: true}
	False = &Boolean{Value: false}
	// Null representa ausencia de valor: un if sin rama ejecutada, un índice
	// fuera del arreglo o una sentencia let, entre otros casos.
	Null = &NullObject{}
)

// String es texto sin las comillas del código fuente.
type String struct{ Value string }

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

// NullObject tiene un único valor posible, compartido como Null.
type NullObject struct{}

func (n *NullObject) Type() ObjectType { return NULL_OBJ }
func (n *NullObject) Inspect() string  { return "null" }

// ReturnValue es una señal interna. Si aparece dentro de varios bloques,
// viaja hacia arriba sin evaluar las sentencias restantes. Al llegar a la
// llamada de función (o al programa principal) se extrae su Value.
type ReturnValue struct{ Value Object }

func (r *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (r *ReturnValue) Inspect() string  { return r.Value.Inspect() }

// Error es también una señal interna: detiene la evaluación y lleva el
// mensaje hasta el REPL, sin producir un panic de Go.
type Error struct{ Message string }

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

// Function conserva tanto el cuerpo sin ejecutar como el entorno donde se
// definió. Ese entorno capturado permite que las funciones sean cierres:
// function(x) { function(y) { x + y } } recuerda x al retornar.
// También permite recursión: el entorno se comparte por puntero y, después
// de ejecutar `let fibonacci = function(...) {...}`, ese mismo entorno ya
// contiene el nombre fibonacci cuando el cuerpo se llama a sí mismo.
type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	params := make([]string, len(f.Parameters))
	for i, p := range f.Parameters {
		params[i] = p.Value
	}
	return "function(" + strings.Join(params, ", ") + ") " + f.Body.String()
}

// Array conserva el orden y admite modificar posiciones existentes.
type Array struct{ Elements []Object }

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	parts := make([]string, len(a.Elements))
	for i, element := range a.Elements {
		parts[i] = element.Inspect()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// HashKey es una clave comparable por Go. El tipo forma parte de ella para
// que 1, true y "1" ocupen entradas diferentes del diccionario.
type HashKey struct {
	Type  ObjectType
	Value string
}

// Hashable limita las claves a enteros, booleanos y cadenas. Un arreglo o
// una función no tienen una representación estable para usarse como clave.
type Hashable interface {
	Object
	HashKey() HashKey
}

func (i *Integer) HashKey() HashKey { return HashKey{INTEGER_OBJ, i.Inspect()} }
func (b *Boolean) HashKey() HashKey { return HashKey{BOOLEAN_OBJ, b.Inspect()} }
func (s *String) HashKey() HashKey  { return HashKey{STRING_OBJ, s.Value} }

// HashPair retiene la clave original para mostrar el hash y su valor.
type HashPair struct {
	Key   Object
	Value Object
}

// Hash usa un mapa para buscar rápidamente y Order para imprimir siempre
// en el orden en que se escribieron las claves. Una clave repetida reemplaza
// su valor sin moverla de lugar.
type Hash struct {
	Pairs map[HashKey]HashPair
	Order []HashKey
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	parts := make([]string, 0, len(h.Order))
	for _, key := range h.Order {
		pair := h.Pairs[key]
		parts = append(parts, pair.Key.Inspect()+": "+pair.Value.Inspect())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// Put agrega o reemplaza una pareja y mantiene Order sincronizado.
func (h *Hash) Put(key Hashable, value Object) {
	if h.Pairs == nil {
		h.Pairs = make(map[HashKey]HashPair)
	}
	hashKey := key.HashKey()
	if _, exists := h.Pairs[hashKey]; !exists {
		h.Order = append(h.Order, hashKey)
	}
	h.Pairs[hashKey] = HashPair{Key: key, Value: value}
}

// Environment es la tabla de variables. outer apunta al ámbito de la
// función que creó este entorno, de modo que una llamada encuentra primero
// sus parámetros y luego las variables visibles al definirse.
type Environment struct {
	store map[string]Object
	outer *Environment
}

// NewEnvironment crea el ámbito principal del programa o del REPL.
func NewEnvironment() *Environment { return &Environment{store: make(map[string]Object)} }

// NewEnclosedEnvironment crea un ámbito local conectado con otro.
func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{store: make(map[string]Object), outer: outer}
}

// Get busca primero aquí y, si hace falta, sigue por los ámbitos exteriores.
func (e *Environment) Get(name string) (Object, bool) {
	if value, ok := e.store[name]; ok {
		return value, true
	}
	if e.outer != nil {
		return e.outer.Get(name)
	}
	return nil, false
}

// Set declara o redefine un nombre en ESTE ámbito. Se usa en let y para
// enlazar los parámetros de una llamada.
func (e *Environment) Set(name string, value Object) Object {
	e.store[name] = value
	return value
}

// Assign cambia la definición visible más cercana. A diferencia de Set,
// no inventa una variable si el nombre nunca se declaró.
func (e *Environment) Assign(name string, value Object) bool {
	if _, ok := e.store[name]; ok {
		e.store[name] = value
		return true
	}
	if e.outer != nil {
		return e.outer.Assign(name, value)
	}
	return false
}
