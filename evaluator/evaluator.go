// Package evaluator recorre el AST y ejecuta cada nodo.
//
// La regla principal es recursiva: para calcular 1 + (2 * 3), primero se
// evalúan los hijos de la suma; el hijo derecho vuelve a evaluar sus propios
// hijos. Los resultados son objetos de objects, no nodos del AST.
package evaluator

import (
	"fmt"
	"math"
	"math/big"

	"polaobesahermosa/ast"
	"polaobesahermosa/objects"
)

// Eval es la entrada del intérprete. env conserva las variables entre
// llamadas si el mismo entorno se reutiliza, como hace el REPL.
// Cada case explica qué significa un tipo de nodo. Las señales ReturnValue
// y Error se propagan sin ejecutar lo que queda del programa.
func Eval(node ast.Node, env *objects.Environment) objects.Object {
	switch n := node.(type) {
	case *ast.Program:
		return evalProgram(n.Statements, env)
	case *ast.BlockStatement:
		return evalBlock(n.Statements, env)
	case *ast.ExpressionStatement:
		return Eval(n.Expression, env)
	case *ast.LetStatement:
		value := Eval(n.Value, env)
		if isError(value) {
			return value
		}
		env.Set(n.Name.Value, value)
		return objects.Null
	case *ast.ReturnStatement:
		value := Eval(n.ReturnValue, env)
		if isError(value) {
			return value
		}
		return &objects.ReturnValue{Value: value}
	case *ast.IntegerLiteral:
		return &objects.Integer{Value: n.Value}
	case *ast.Boolean:
		return nativeBool(n.Value)
	case *ast.StringLiteral:
		return &objects.String{Value: n.Value}
	case *ast.Identifier:
		if value, ok := env.Get(n.Value); ok {
			return value
		}
		return newError("identificador no definido: %s", n.Value)
	case *ast.PrefixExpression:
		right := Eval(n.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefix(n.Operator, right)
	case *ast.InfixExpression:
		left := Eval(n.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(n.Right, env)
		if isError(right) {
			return right
		}
		return evalInfix(n.Operator, left, right)
	case *ast.AssignExpression:
		return evalAssign(n, env)
	case *ast.IfExpression:
		condition := Eval(n.Condition, env)
		if isError(condition) {
			return condition
		}
		if truthy(condition) {
			return Eval(n.Consequence, env)
		}
		if n.Alternative != nil {
			return Eval(n.Alternative, env)
		}
		return objects.Null
	case *ast.WhileExpression:
		return evalWhile(n, env)
	case *ast.ForExpression:
		return evalFor(n, env)
	case *ast.FunctionLiteral:
		// Se guarda el entorno de DEFINICIÓN; el cuerpo espera hasta la llamada.
		return &objects.Function{Parameters: n.Parameters, Body: n.Body, Env: env}
	case *ast.CallExpression:
		fn := Eval(n.Function, env)
		if isError(fn) {
			return fn
		}
		args, err := evalExpressions(n.Arguments, env)
		if err != nil {
			return err
		}
		return applyFunction(fn, args)
	case *ast.ArrayLiteral:
		elements, err := evalExpressions(n.Elements, env)
		if err != nil {
			return err
		}
		return &objects.Array{Elements: elements}
	case *ast.IndexExpression:
		left := Eval(n.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(n.Index, env)
		if isError(index) {
			return index
		}
		return evalIndex(left, index)
	case *ast.HashLiteral:
		return evalHash(n, env)
	case nil:
		return newError("nodo incompleto del AST")
	default:
		return newError("nodo no soportado: %T", node)
	}
}

// Un programa entrega el último resultado, pero desenvuelve return al
// llegar al nivel superior. Un bloque conserva la envoltura para que return
// pueda atravesar if, while y for hasta la llamada de función.
func evalProgram(statements []ast.Statement, env *objects.Environment) objects.Object {
	result := objects.Object(objects.Null)
	for _, statement := range statements {
		result = Eval(statement, env)
		switch value := result.(type) {
		case *objects.ReturnValue:
			return value.Value
		case *objects.Error:
			return value
		}
	}
	return result
}

func evalBlock(statements []ast.Statement, env *objects.Environment) objects.Object {
	result := objects.Object(objects.Null)
	for _, statement := range statements {
		result = Eval(statement, env)
		if result.Type() == objects.RETURN_VALUE_OBJ || isError(result) {
			return result
		}
	}
	return result
}

// Solo false y null son falsos. Por eso 0, "" y [] son verdaderos, una
// convención explícita del lenguaje que se usa en if y en ambos ciclos.
func truthy(value objects.Object) bool {
	if value.Type() == objects.NULL_OBJ {
		return false
	}
	if b, ok := value.(*objects.Boolean); ok {
		return b.Value
	}
	return true
}

func nativeBool(value bool) objects.Object {
	if value {
		return objects.True
	}
	return objects.False
}

func evalPrefix(operator string, right objects.Object) objects.Object {
	switch operator {
	case "!":
		return nativeBool(!truthy(right))
	case "-":
		value, ok := right.(*objects.Integer)
		if !ok {
			return newError("operador - no admite %s", right.Type())
		}
		if value.Value == math.MinInt64 {
			return newError("desbordamiento al negar %d", value.Value)
		}
		return &objects.Integer{Value: -value.Value}
	}
	return newError("operador prefijo desconocido: %s", operator)
}

// Se comprueban los tipos antes de operar: sumar un entero y un texto es
// un error del lenguaje, no un error de tipos de Go.
func evalInfix(operator string, left, right objects.Object) objects.Object {
	if left.Type() != right.Type() {
		return newError("tipos incompatibles: %s %s %s", left.Type(), operator, right.Type())
	}
	switch l := left.(type) {
	case *objects.Integer:
		r := right.(*objects.Integer)
		switch operator {
		case "+":
			return checkedInteger(new(big.Int).Add(big.NewInt(l.Value), big.NewInt(r.Value)))
		case "-":
			return checkedInteger(new(big.Int).Sub(big.NewInt(l.Value), big.NewInt(r.Value)))
		case "*":
			return checkedInteger(new(big.Int).Mul(big.NewInt(l.Value), big.NewInt(r.Value)))
		case "/":
			if r.Value == 0 {
				return newError("división por cero")
			}
			if l.Value == math.MinInt64 && r.Value == -1 {
				return newError("desbordamiento en división")
			}
			return &objects.Integer{Value: l.Value / r.Value}
		case "%":
			if r.Value == 0 {
				return newError("módulo por cero")
			}
			return &objects.Integer{Value: l.Value % r.Value}
		case "**":
			if r.Value < 0 {
				return newError("exponente negativo: %d", r.Value)
			}
			// big.Int calcula sin desbordamiento intermedio; al final se
			// comprueba que el resultado cabe en el int64 del lenguaje.
			// Para bases 0, 1 y -1, incluso exponentes enormes son baratos.
			if l.Value == 0 && r.Value > 0 {
				return &objects.Integer{Value: 0}
			}
			if l.Value == 1 {
				return &objects.Integer{Value: 1}
			}
			if l.Value == -1 {
				return &objects.Integer{Value: 1 - 2*(r.Value&1)}
			}
			// Una base de magnitud >= 2 con exponente >= 64 nunca cabe.
			if r.Value >= 64 {
				return newError("desbordamiento de entero")
			}
			return checkedInteger(new(big.Int).Exp(big.NewInt(l.Value), big.NewInt(r.Value), nil))
		case "<":
			return nativeBool(l.Value < r.Value)
		case ">":
			return nativeBool(l.Value > r.Value)
		case "<=":
			return nativeBool(l.Value <= r.Value)
		case ">=":
			return nativeBool(l.Value >= r.Value)
		case "==":
			return nativeBool(l.Value == r.Value)
		case "!=":
			return nativeBool(l.Value != r.Value)
		}
	case *objects.Boolean:
		r := right.(*objects.Boolean)
		if operator == "==" {
			return nativeBool(l.Value == r.Value)
		}
		if operator == "!=" {
			return nativeBool(l.Value != r.Value)
		}
	case *objects.String:
		r := right.(*objects.String)
		switch operator {
		case "+":
			return &objects.String{Value: l.Value + r.Value}
		case "==":
			return nativeBool(l.Value == r.Value)
		case "!=":
			return nativeBool(l.Value != r.Value)
		}
	case *objects.Array:
		r := right.(*objects.Array)
		if operator == "+" {
			// Una lista de movimientos de Hanói se arma uniendo las listas
			// de los subproblemas. Se copia para no modificar los operandos:
			// [1] + [2] produce [1, 2] y ambas listas originales siguen igual.
			elements := make([]objects.Object, 0, len(l.Elements)+len(r.Elements))
			elements = append(elements, l.Elements...)
			elements = append(elements, r.Elements...)
			return &objects.Array{Elements: elements}
		}
		if operator == "==" {
			return nativeBool(left == right)
		}
		if operator == "!=" {
			return nativeBool(left != right)
		}
	default:
		// Para hashes y funciones, la igualdad comprueba identidad:
		// dos referencias al mismo objeto son iguales; dos literales, no.
		if operator == "==" {
			return nativeBool(left == right)
		}
		if operator == "!=" {
			return nativeBool(left != right)
		}
	}
	return newError("operador no admitido: %s %s %s", left.Type(), operator, right.Type())
}

// checkedInteger convierte un cálculo exacto de big.Int al int64 del
// lenguaje. Esto evita que Go dé silenciosamente un resultado que cambió
// de signo al sobrepasar el límite de 64 bits.
func checkedInteger(value *big.Int) objects.Object {
	if !value.IsInt64() {
		return newError("desbordamiento de entero")
	}
	return &objects.Integer{Value: value.Int64()}
}

// La última expresión ejecutada es el valor del ciclo. Si no hubo vueltas,
// se obtiene null. Las señales de error y return interrumpen el ciclo.
func evalWhile(n *ast.WhileExpression, env *objects.Environment) objects.Object {
	result := objects.Object(objects.Null)
	for {
		condition := Eval(n.Condition, env)
		if isError(condition) {
			return condition
		}
		if !truthy(condition) {
			return result
		}
		result = Eval(n.Body, env)
		if result.Type() == objects.RETURN_VALUE_OBJ || isError(result) {
			return result
		}
	}
}

func evalFor(n *ast.ForExpression, env *objects.Environment) objects.Object {
	// El encabezado y el cuerpo comparten el entorno actual, como while.
	if n.Init != nil {
		init := Eval(n.Init, env)
		if init.Type() == objects.RETURN_VALUE_OBJ || isError(init) {
			return init
		}
	}
	result := objects.Object(objects.Null)
	for {
		if n.Condition != nil {
			condition := Eval(n.Condition, env)
			if isError(condition) {
				return condition
			}
			if !truthy(condition) {
				return result
			}
		}
		result = Eval(n.Body, env)
		if result.Type() == objects.RETURN_VALUE_OBJ || isError(result) {
			return result
		}
		if n.Update != nil {
			update := Eval(n.Update, env)
			if isError(update) {
				return update
			}
		}
	}
}

func evalExpressions(expressions []ast.Expression, env *objects.Environment) ([]objects.Object, *objects.Error) {
	results := make([]objects.Object, 0, len(expressions))
	for _, expression := range expressions {
		value := Eval(expression, env)
		if err, ok := value.(*objects.Error); ok {
			return nil, err
		}
		results = append(results, value)
	}
	return results, nil
}

func applyFunction(value objects.Object, args []objects.Object) objects.Object {
	fn, ok := value.(*objects.Function)
	if !ok {
		return newError("%s no es una función", value.Type())
	}
	if len(args) != len(fn.Parameters) {
		return newError("cantidad de argumentos incorrecta: se esperaban %d, llegaron %d", len(fn.Parameters), len(args))
	}
	// Cada llamada tiene sus propios parámetros; el enlace exterior apunta
	// al entorno de definición, no al entorno del llamador.
	local := objects.NewEnclosedEnvironment(fn.Env)
	for i, parameter := range fn.Parameters {
		local.Set(parameter.Value, args[i])
	}
	result := Eval(fn.Body, local)
	if returned, ok := result.(*objects.ReturnValue); ok {
		return returned.Value
	}
	return result
}

func evalHash(n *ast.HashLiteral, env *objects.Environment) objects.Object {
	hash := &objects.Hash{Pairs: make(map[objects.HashKey]objects.HashPair)}
	for _, pair := range n.Pairs {
		key := Eval(pair.Key, env)
		if isError(key) {
			return key
		}
		hashable, ok := key.(objects.Hashable)
		if !ok {
			return newError("tipo de clave no admitido: %s", key.Type())
		}
		value := Eval(pair.Value, env)
		if isError(value) {
			return value
		}
		hash.Put(hashable, value)
	}
	return hash
}

func evalIndex(container, index objects.Object) objects.Object {
	switch value := container.(type) {
	case *objects.Array:
		i, ok := index.(*objects.Integer)
		if !ok {
			return newError("índice de arreglo debe ser INTEGER, recibido %s", index.Type())
		}
		if i.Value < 0 || i.Value >= int64(len(value.Elements)) {
			return objects.Null
		}
		return value.Elements[i.Value]
	case *objects.Hash:
		key, ok := index.(objects.Hashable)
		if !ok {
			return newError("tipo de clave no admitido: %s", index.Type())
		}
		if pair, exists := value.Pairs[key.HashKey()]; exists {
			return pair.Value
		}
		return objects.Null
	default:
		return newError("no se puede indexar %s", container.Type())
	}
}

// En una asignación indexada se evalúa primero el contenedor y el índice,
// una sola vez cada uno. Así matriz[0][1] modifica el arreglo interior real.
func evalAssign(n *ast.AssignExpression, env *objects.Environment) objects.Object {
	switch target := n.Left.(type) {
	case *ast.Identifier:
		if _, exists := env.Get(target.Value); !exists {
			return newError("identificador no definido: %s", target.Value)
		}
		value := Eval(n.Value, env)
		if isError(value) {
			return value
		}
		env.Assign(target.Value, value)
		return value
	case *ast.IndexExpression:
		container := Eval(target.Left, env)
		if isError(container) {
			return container
		}
		index := Eval(target.Index, env)
		if isError(index) {
			return index
		}
		// Validar antes de evaluar el nuevo valor evita efectos secundarios
		// cuando el destino no admite escritura.
		switch c := container.(type) {
		case *objects.Array:
			i, ok := index.(*objects.Integer)
			if !ok {
				return newError("índice de arreglo debe ser INTEGER, recibido %s", index.Type())
			}
			if i.Value < 0 || i.Value >= int64(len(c.Elements)) {
				return newError("índice de arreglo fuera de rango: %d", i.Value)
			}
			value := Eval(n.Value, env)
			if isError(value) {
				return value
			}
			c.Elements[i.Value] = value
			return value
		case *objects.Hash:
			key, ok := index.(objects.Hashable)
			if !ok {
				return newError("tipo de clave no admitido: %s", index.Type())
			}
			value := Eval(n.Value, env)
			if isError(value) {
				return value
			}
			c.Put(key, value)
			return value
		default:
			return newError("no se puede asignar por índice a %s", container.Type())
		}
	}
	return newError("destino de asignación no admitido: %T", n.Left)
}

func isError(value objects.Object) bool { return value != nil && value.Type() == objects.ERROR_OBJ }
func newError(format string, args ...any) *objects.Error {
	return &objects.Error{Message: fmt.Sprintf(format, args...)}
}
