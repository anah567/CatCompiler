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
	"strings"

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

	case *ast.VarDeclaration:
		return evalDeclaration(n, env)

	case *ast.VarDeclarationList:
		for _, d := range n.Declarations {
			if r := evalDeclaration(d, env); isError(r) {
				return r
			}
		}
		return objects.Null

	case *ast.FunctionDeclaration:
		// Un prototipo no tiene cuerpo, por eso todavía no se ejecuta.
		if n.Body == nil {
			return objects.Null
		}

		if n.Name == nil {
			return newError("función sin nombre")
		}

		// Guardamos los nombres de los parámetros para mantener
		// la compatibilidad con las funciones existentes.
		params := make([]*ast.Identifier, len(n.Params))

		for i, param := range n.Params {
			if param.Name == nil {
				return newError(
					"parámetro sin nombre en función %s",
					n.Name.Value,
				)
			}
			params[i] = param.Name
		}

		// También guardamos los tipos de parámetros y de retorno.
		fn := &objects.Function{
			Parameters: params,
			Params:     n.Params,
			ReturnType: n.ReturnType,
			Body:       n.Body,
			Env:        env,
		}

		if !env.Declare(n.Name.Value, fn, "function", true) {
			return newError("nombre ya declarado: %s", n.Name.Value)
		}

		return objects.Null

	case *ast.CoutStatement:
		// Guardamos temporalmente el texto que queremos imprimir.
		var out strings.Builder

		for _, expression := range n.Values {
			// Evaluamos cada expresión del cout.
			value := Eval(expression, env)

			if isError(value) {
				return value
			}

			// Agregamos el valor convertido a texto.
			out.WriteString(value.Inspect())
		}

		// Imprimimos en la salida configurada para este entorno.
		// También funciona dentro de funciones y ciclos.
		if _, err := fmt.Fprint(env.Output(), out.String()); err != nil {
			return newError("no se pudo escribir la salida: %s", err)
		}

		return objects.Null
	case *ast.EndlLiteral:
		return &objects.String{Value: "\n"}

	case *ast.DoubleLiteral:
		return &objects.Double{Value: n.Value}

	case *ast.BreakStatement:
		return &objects.BreakSignal{}

	case *ast.ContinueStatement:
		return &objects.ContinueSignal{}

	case *ast.DoWhileStatement:
		for {
			result := Eval(n.Body, env)
			if isError(result) ||
				result.Type() == objects.RETURN_VALUE_OBJ {
				return result
			}
			if result.Type() == objects.BREAK_OBJ {
				return objects.Null
			}

			condition := Eval(n.Condition, env)
			if isError(condition) {
				return condition
			}
			if !truthy(condition) {
				break
			}
		}
		return objects.Null

	case *ast.PostfixExpression:
		return evalIncrement(n.Left, n.Operator, false, env)

	case *ast.MemberExpression:
		return newError(
			"miembro todavía no implementado: %s",
			n.Member.Value,
		)

	case *ast.ReturnStatement:
		value := objects.Object(objects.Null)
		if n.ReturnValue != nil {
			value = Eval(n.ReturnValue, env)
		}
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
		if builtin, ok := builtins[n.Value]; ok {
			return builtin
		}
		return newError("identificador no definido: %s", n.Value)

	case *ast.PrefixExpression:
		if n.Operator == "++" || n.Operator == "--" {
			return evalIncrement(n.Right, n.Operator, true, env)
		}
		right := Eval(n.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefix(n.Operator, right)

	case *ast.InfixExpression:
		// Primero evaluamos la expresión izquierda.
		left := Eval(n.Left, env)
		if isError(left) {
			return left
		}

		// En &&, si la izquierda es falsa, no necesitamos
		// evaluar la derecha.
		if n.Operator == "&&" && !truthy(left) {
			return objects.False
		}

		// En ||, si la izquierda es verdadera, no necesitamos
		// evaluar la derecha.
		if n.Operator == "||" && truthy(left) {
			return objects.True
		}

		// Para los demás casos evaluamos la expresión derecha.
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

	case *ast.CallExpression:
		// s.size(), v.push_back(x): llamadas a método.
		if member, ok := n.Function.(*ast.MemberExpression); ok {
			return evalMethodCall(member, n.Arguments, env)
		}
		fn := Eval(n.Function, env)
		if isError(fn) {
			return fn
		}
		args, err := evalExpressions(n.Arguments, env)
		if err != nil {
			return err
		}
		return applyFunction(fn, args)

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
		case *objects.BreakSignal, *objects.ContinueSignal:
			return newError("break/continue fuera de un ciclo")
		}
	}
	return result
}

// evalBlock ejecuta las sentencias de un bloque.
//
// Cada bloque crea un ámbito local para sus variables.
// Esto significa que una variable declarada dentro de un if,
// while o cualquier bloque no existe fuera de él.
//
// El nuevo entorno sigue conectado con el exterior.
// Por eso, el bloque puede consultar y modificar variables
// que fueron declaradas antes.
//
// Si aparece una señal de return, break, continue o error,
// se detiene la ejecución del bloque.
func evalBlock(statements []ast.Statement, env *objects.Environment) objects.Object {
	// Creamos un ámbito local conectado con el entorno exterior.
	local := objects.NewEnclosedEnvironment(env)

	result := objects.Object(objects.Null)

	for _, statement := range statements {
		// Ejecutamos las sentencias dentro del ámbito local.
		result = Eval(statement, local)

		// Si hay una señal, detenemos el bloque.
		if isSignal(result) {
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

// nativeBool reutiliza los objetos booleanos compartidos.
func nativeBool(value bool) objects.Object {
	if value {
		return objects.True
	}
	return objects.False
}

// evalPrefix ejecuta operadores que aparecen antes de un valor.
func evalPrefix(operator string, right objects.Object) objects.Object {
	switch operator {
	case "!":
		return nativeBool(!truthy(right))

	case "+":
		return right

	case "-":
		if d, ok := right.(*objects.Double); ok {
			return &objects.Double{Value: -d.Value}
		}

		value, ok := right.(*objects.Integer)
		if !ok {
			return newError("operador - no admite %s", right.Type())
		}
		if value.Value == math.MinInt64 {
			return newError(
				"desbordamiento al negar %d",
				value.Value,
			)
		}
		return &objects.Integer{Value: -value.Value}
	}
	return newError("operador prefijo desconocido: %s", operator)
}

// Se comprueban los tipos antes de operar: sumar un entero y un texto es
// un error del lenguaje, no un error de tipos de Go.
func evalInfix(operator string, left, right objects.Object) objects.Object {
	if operator == "&&" {
		return nativeBool(truthy(left) && truthy(right))
	}
	if operator == "||" {
		return nativeBool(truthy(left) || truthy(right))
	}

	// Se permite combinar int y double convirtiendo el entero a decimal.
	if left.Type() != right.Type() {
		if l, ok := left.(*objects.Integer); ok {
			if _, ok := right.(*objects.Double); ok {
				left = &objects.Double{Value: float64(l.Value)}
			}
		}
		if r, ok := right.(*objects.Integer); ok {
			if _, ok := left.(*objects.Double); ok {
				right = &objects.Double{Value: float64(r.Value)}
			}
		}
	}

	if left.Type() != right.Type() {
		return newError(
			"tipos incompatibles: %s %s %s",
			left.Type(), operator, right.Type(),
		)
	}

	switch l := left.(type) {
	case *objects.Double:
		r := right.(*objects.Double)

		switch operator {
		case "+":
			return &objects.Double{Value: l.Value + r.Value}
		case "-":
			return &objects.Double{Value: l.Value - r.Value}
		case "*":
			return &objects.Double{Value: l.Value * r.Value}
		case "/":
			if r.Value == 0 {
				return newError("división por cero")
			}
			return &objects.Double{Value: l.Value / r.Value}
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

	case *objects.Integer:
		r := right.(*objects.Integer)

		switch operator {
		case "+":
			return checkedInteger(
				new(big.Int).Add(big.NewInt(l.Value), big.NewInt(r.Value)),
			)

		case "-":
			return checkedInteger(
				new(big.Int).Sub(big.NewInt(l.Value), big.NewInt(r.Value)),
			)

		case "*":
			return checkedInteger(
				new(big.Int).Mul(big.NewInt(l.Value), big.NewInt(r.Value)),
			)

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

			return checkedInteger(
				new(big.Int).Exp(
					big.NewInt(l.Value),
					big.NewInt(r.Value),
					nil,
				),
			)

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
			elements := make(
				[]objects.Object,
				0,
				len(l.Elements)+len(r.Elements),
			)
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

	return newError(
		"operador no admitido: %s %s %s",
		left.Type(), operator, right.Type(),
	)
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

		if result.Type() == objects.BREAK_OBJ {
			return objects.Null
		}
		if result.Type() == objects.CONTINUE_OBJ {
			continue
		}
		if isSignal(result) {
			return result
		}
	}
}

// evalFor ejecuta la inicialización, condición, cuerpo y actualización.
//
// El ciclo tiene su propio ámbito. Una variable declarada
// en el encabezado, como int i = 0, solamente existe
// mientras se ejecuta el for.
//
// El cuerpo también crea sus propios ámbitos mediante evalBlock.
// Las variables externas siguen siendo accesibles.
func evalFor(n *ast.ForExpression, env *objects.Environment) objects.Object {
	// Creamos un entorno exclusivo para el ciclo.
	local := objects.NewEnclosedEnvironment(env)

	// Ejecutamos la inicialización una sola vez.
	if n.Init != nil {
		init := Eval(n.Init, local)
		if isSignal(init) {
			return init
		}
	}

	result := objects.Object(objects.Null)

	for {
		// Comprobamos la condición antes de cada vuelta.
		if n.Condition != nil {
			condition := Eval(n.Condition, local)
			if isError(condition) {
				return condition
			}

			if !truthy(condition) {
				return result
			}
		}

		// Ejecutamos el cuerpo del ciclo.
		result = Eval(n.Body, local)

		// break termina el ciclo.
		if result.Type() == objects.BREAK_OBJ {
			return objects.Null
		}

		// Los errores y return deben propagarse.
		if result.Type() != objects.CONTINUE_OBJ && isSignal(result) {
			return result
		}

		// Ejecutamos la actualización.
		// También se ejecuta cuando hubo continue.
		if n.Update != nil {
			update := Eval(n.Update, local)
			if isSignal(update) {
				return update
			}
		}
	}
}

// evalExpressions evalúa los argumentos de una función en orden.
func evalExpressions(
	expressions []ast.Expression,
	env *objects.Environment,
) ([]objects.Object, *objects.Error) {
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

// applyFunction ejecuta una función con sus argumentos.
// Comprueba la cantidad de argumentos, sus tipos y el retorno.
// Cada llamada tiene su propio entorno para evitar que los
// parámetros de una llamada modifiquen los de otra.
// callDepth cuenta las llamadas activas de funciones del usuario.
var callDepth int

const maxCallDepth = 10000

func applyFunction(value objects.Object, args []objects.Object) objects.Object {
	if builtin, ok := value.(*objects.Builtin); ok {
		return builtin.Fn(args...)
	}
	fn, ok := value.(*objects.Function)
	if !ok {
		return newError("%s no es una función", value.Type())
	}

	// Verificamos la cantidad de argumentos.
	if len(args) != len(fn.Parameters) {
		return newError(
			"cantidad de argumentos incorrecta: se esperaban %d, llegaron %d",
			len(fn.Parameters),
			len(args),
		)
	}

	// Límite de profundidad: una recursión sin caso base produce un
	// error del lenguaje en lugar de tumbar el intérprete.
	callDepth++
	defer func() { callDepth-- }()
	if callDepth > maxCallDepth {
		return newError("desbordamiento de pila: más de %d llamadas anidadas", maxCallDepth)
	}

	// Creamos un ámbito local para esta llamada.
	// Conservamos el entorno original para permitir recursión.
	local := objects.NewEnclosedEnvironment(fn.Env)

	for i, parameter := range fn.Parameters {
		argument := args[i]
		paramType := ""

		if i < len(fn.Params) {
			paramType = fn.Params[i].Type

			// Los parámetros declarados como arreglos deben
			// recibir un arreglo.
			if fn.Params[i].IsArray {
				if _, ok := argument.(*objects.Array); !ok {
					return newError(
						"el parámetro %s debe recibir un arreglo",
						parameter.Value,
					)
				}
			} else {
				// Validamos el tipo de los parámetros normales.
				converted, err := convertType(
					paramType,
					argument,
					false,
				)
				if err != nil {
					return newError(
						"parámetro %s: %s",
						parameter.Value,
						err.Message,
					)
				}
				argument = converted
			}
		}

		// Declaramos el parámetro dentro de la función.
		if !local.Declare(
			parameter.Value,
			argument,
			paramType,
			false,
		) {
			return newError(
				"parámetro repetido: %s",
				parameter.Value,
			)
		}
	}

	// Ejecutamos el cuerpo de la función.
	result := Eval(fn.Body, local)

	if isError(result) {
		return result
	}

	// Extraemos el valor devuelto por return.
	returned, hasReturn := result.(*objects.ReturnValue)

	if hasReturn {
		result = returned.Value
	} else {
		result = objects.Null
	}

	// Las funciones void no deben devolver un valor.
	if fn.ReturnType == "void" {
		if hasReturn && result.Type() != objects.NULL_OBJ {
			return newError(
				"una función void no puede devolver un valor",
			)
		}
		return objects.Null
	}

	// Las funciones tipadas deben devolver un valor.
	if fn.ReturnType != "" && fn.ReturnType != "auto" {
		if !hasReturn {
			return newError(
				"la función debe devolver un valor de tipo %s",
				fn.ReturnType,
			)
		}

		converted, err := convertType(
			fn.ReturnType,
			result,
			false,
		)
		if err != nil {
			return newError(
				"tipo de retorno incorrecto: %s",
				err.Message,
			)
		}

		return converted
	}

	return result
}

// evalIndex obtiene un elemento de un arreglo o diccionario.
func evalIndex(container, index objects.Object) objects.Object {
	switch value := container.(type) {
	case *objects.Array:
		i, ok := index.(*objects.Integer)
		if !ok {
			return newError(
				"índice de arreglo debe ser INTEGER, recibido %s",
				index.Type(),
			)
		}
		if i.Value < 0 || i.Value >= int64(len(value.Elements)) {
			return newError("índice de arreglo fuera de rango: %d", i.Value)
		}
		return value.Elements[i.Value]

	case *objects.Hash:
		key, ok := index.(objects.Hashable)
		if !ok {
			return newError(
				"tipo de clave no admitido: %s",
				index.Type(),
			)
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
func evalAssign(
	n *ast.AssignExpression,
	env *objects.Environment,
) objects.Object {
	switch target := n.Left.(type) {
	case *ast.Identifier:
		if _, exists := env.Get(target.Value); !exists {
			return newError(
				"identificador no definido: %s",
				target.Value,
			)
		}

		value := Eval(n.Value, env)
		if isError(value) {
			return value
		}

		typ, constant, _ := env.GetVariableInfo(target.Value)
		if constant {
			return newError(
				"no se puede modificar constante: %s",
				target.Value,
			)
		}

		if n.Operator != "=" {
			current, _ := env.Get(target.Value)
			value = evalInfix(
				strings.TrimSuffix(n.Operator, "="),
				current,
				value,
			)
			if isError(value) {
				return value
			}
		}

		converted, err := convertType(typ, value, false)
		if err != nil {
			return err
		}

		if !env.AssignChecked(target.Value, converted) {
			return newError(
				"asignación no permitida: %s",
				target.Value,
			)
		}
		return converted

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
				return newError(
					"índice de arreglo debe ser INTEGER, recibido %s",
					index.Type(),
				)
			}
			if i.Value < 0 || i.Value >= int64(len(c.Elements)) {
				return newError(
					"índice de arreglo fuera de rango: %d",
					i.Value,
				)
			}

			value := Eval(n.Value, env)
			if isError(value) {
				return value
			}

			if n.Operator != "=" {
				value = evalInfix(
					strings.TrimSuffix(n.Operator, "="),
					c.Elements[i.Value],
					value,
				)
				if isError(value) {
					return value
				}
			}

			if id, ok := target.Left.(*ast.Identifier); ok {
				typ, constant, _ := env.GetVariableInfo(id.Value)
				if constant {
					return newError(
						"no se puede modificar constante: %s",
						id.Value,
					)
				}
				converted, err := convertType(typ, value, false)
				if err != nil {
					return err
				}
				value = converted
			}

			c.Elements[i.Value] = value
			return value

		case *objects.Hash:
			key, ok := index.(objects.Hashable)
			if !ok {
				return newError(
					"tipo de clave no admitido: %s",
					index.Type(),
				)
			}

			value := Eval(n.Value, env)
			if isError(value) {
				return value
			}
			c.Put(key, value)
			return value

		default:
			return newError(
				"no se puede asignar por índice a %s",
				container.Type(),
			)
		}
	}

	return newError(
		"destino de asignación no admitido: %T",
		n.Left,
	)
}

// isError comprueba si un valor representa un error del intérprete.
func isError(value objects.Object) bool {
	return value != nil && value.Type() == objects.ERROR_OBJ
}

// newError crea un error con un mensaje descriptivo.
func newError(format string, args ...any) *objects.Error {
	return &objects.Error{
		Message: fmt.Sprintf(format, args...),
	}
}

// evalDeclaration ejecuta declaraciones de variables y arreglos de C++.
func evalDeclaration(
	n *ast.VarDeclaration,
	env *objects.Environment,
) objects.Object {
	if n.Name == nil {
		return newError("declaración sin nombre")
	}

	var value objects.Object = objects.Null

	if n.IsArray {
		size := 0

		if n.ArraySize != nil {
			sz := Eval(n.ArraySize, env)
			if isError(sz) {
				return sz
			}

			integer, ok := sz.(*objects.Integer)
			if !ok || integer.Value < 0 || integer.Value > 1000000 {
				return newError("tamaño de arreglo inválido")
			}
			size = int(integer.Value)
		}

		if len(n.Elements) > size {
			if n.ArraySize != nil {
				return newError("demasiados elementos para el arreglo")
			}
			size = len(n.Elements)
		}

		arr := &objects.Array{
			Elements: make([]objects.Object, size),
		}

		for i := range arr.Elements {
			arr.Elements[i] = defaultValue(n.Type)
		}

		for i, expr := range n.Elements {
			v := Eval(expr, env)
			if isError(v) {
				return v
			}

			converted, err := convertType(n.Type, v, false)
			if err != nil {
				return err
			}
			arr.Elements[i] = converted
		}

		value = arr

	} else if n.Value != nil {
		value = Eval(n.Value, env)
		if isError(value) {
			return value
		}

	} else {
		value = defaultValue(n.Type)
	}

	if n.Const && n.Value == nil && !n.IsArray {
		return newError(
			"const requiere inicialización: %s",
			n.Name.Value,
		)
	}

	converted, err := convertType(n.Type, value, n.IsArray)
	if err != nil {
		return err
	}

	if !env.Declare(n.Name.Value, converted, n.Type, n.Const) {
		return newError(
			"variable ya declarada: %s",
			n.Name.Value,
		)
	}

	return objects.Null
}

// defaultValue devuelve el valor inicial de una variable sin asignación.
func defaultValue(typ string) objects.Object {
	switch typ {
	case "int":
		return &objects.Integer{Value: 0}
	case "double":
		return &objects.Double{Value: 0}
	case "bool":
		return objects.False
	case "string":
		return &objects.String{Value: ""}
	}
	return objects.Null
}

// evalIncrement modifica una variable o una posición de un arreglo.
// Acepta ++ y --, tanto antes como después del valor.
//
// Prefijo: ++x modifica x y devuelve el valor nuevo.
// Postfijo: x++ modifica x y devuelve el valor anterior.
func evalIncrement(
	expr ast.Expression,
	operator string,
	prefix bool,
	env *objects.Environment,
) objects.Object {
	var old objects.Object

	// Esta función guarda cómo modificar el destino.
	// Se ejecuta después de calcular y validar el nuevo valor.
	var guardar func(objects.Object) bool

	switch target := expr.(type) {

	case *ast.Identifier:
		// Buscamos la variable.
		value, exists := env.Get(target.Value)
		if !exists {
			return newError(
				"identificador no definido: %s",
				target.Value,
			)
		}

		// Impedimos modificar constantes.
		_, constant, _ := env.GetVariableInfo(target.Value)
		if constant {
			return newError(
				"no se puede modificar constante: %s",
				target.Value,
			)
		}

		old = value

		guardar = func(value objects.Object) bool {
			return env.AssignChecked(target.Value, value)
		}

	case *ast.IndexExpression:
		// Obtenemos el arreglo que queremos modificar.
		container := Eval(target.Left, env)
		if isError(container) {
			return container
		}

		array, ok := container.(*objects.Array)
		if !ok {
			return newError(
				"incremento por índice requiere un arreglo",
			)
		}

		// Evaluamos la posición.
		index := Eval(target.Index, env)
		if isError(index) {
			return index
		}

		position, ok := index.(*objects.Integer)
		if !ok {
			return newError(
				"el índice del arreglo debe ser entero",
			)
		}

		if position.Value < 0 ||
			position.Value >= int64(len(array.Elements)) {
			return newError(
				"índice de arreglo fuera de rango: %d",
				position.Value,
			)
		}

		// Comprobamos si el arreglo es constante.
		if id, ok := target.Left.(*ast.Identifier); ok {
			_, constant, _ := env.GetVariableInfo(id.Value)
			if constant {
				return newError(
					"no se puede modificar constante: %s",
					id.Value,
				)
			}
		}

		old = array.Elements[position.Value]

		guardar = func(value objects.Object) bool {
			array.Elements[position.Value] = value
			return true
		}

	default:
		return newError(
			"incremento requiere una variable o posición de arreglo",
		)
	}

	// Determinamos si sumamos o restamos uno.
	delta := int64(1)
	if operator == "--" {
		delta = -1
	}

	var updated objects.Object

	switch number := old.(type) {
	case *objects.Integer:
		updated = checkedInteger(
			new(big.Int).Add(
				big.NewInt(number.Value),
				big.NewInt(delta),
			),
		)

	case *objects.Double:
		updated = &objects.Double{
			Value: number.Value + float64(delta),
		}

	default:
		return newError(
			"incremento no admite %s",
			old.Type(),
		)
	}

	// Comprobamos que no haya desbordamiento.
	if isError(updated) {
		return updated
	}

	// Guardamos el nuevo valor.
	if !guardar(updated) {
		return newError("no se pudo modificar el valor")
	}

	if prefix {
		return updated
	}

	return old
}

// isSignal identifica valores que interrumpen la ejecución normal.
func isSignal(value objects.Object) bool {
	if value == nil {
		return false
	}

	switch value.Type() {
	case objects.ERROR_OBJ,
		objects.RETURN_VALUE_OBJ,
		objects.BREAK_OBJ,
		objects.CONTINUE_OBJ:
		return true
	}
	return false
}

// convertType comprueba los tipos declarados y permite int hacia double.
func convertType(
	typ string,
	value objects.Object,
	array bool,
) (objects.Object, *objects.Error) {
	if array {
		if _, ok := value.(*objects.Array); !ok {
			return nil, newError("se esperaba arreglo")
		}
		return value, nil
	}

	if typ == "" || typ == "auto" || typ == "function" {
		return value, nil
	}

	if typ == "double" {
		if n, ok := value.(*objects.Integer); ok {
			return &objects.Double{
				Value: float64(n.Value),
			}, nil
		}
	}

	expected := map[string]objects.ObjectType{
		"int":    objects.INTEGER_OBJ,
		"double": objects.DOUBLE_OBJ,
		"bool":   objects.BOOLEAN_OBJ,
		"string": objects.STRING_OBJ,
	}

	t, ok := expected[typ]
	if !ok {
		return nil, newError("tipo desconocido: %s", typ)
	}

	if value.Type() != t {
		return nil, newError(
			"se esperaba %s y se recibió %s",
			typ,
			value.Type(),
		)
	}

	return value, nil
}
