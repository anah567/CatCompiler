// builtins.go define las funciones integradas del intérprete: funciones
// libres (to_string, stoi, abs, ...) y métodos (s.size(), v.push_back(x)).
package evaluator

import (
	"math"
	"strconv"
	"strings"

	"polaobesahermosa/ast"
	"polaobesahermosa/objects"
)

// builtins relaciona cada nombre con su función integrada. El evaluador
// solo la busca aquí si el nombre NO está en el entorno, así que una
// función del usuario con el mismo nombre tiene prioridad.
var builtins = map[string]*objects.Builtin{
	"to_string": {Name: "to_string", Fn: func(args ...objects.Object) objects.Object {
		if err := checkArgs("to_string", args, 1); err != nil {
			return err
		}
		switch v := args[0].(type) {
		case *objects.Integer, *objects.Double:
			return &objects.String{Value: v.Inspect()}
		}
		return newError("to_string no admite %s", args[0].Type())
	}},
	"stoi": {Name: "stoi", Fn: func(args ...objects.Object) objects.Object {
		if err := checkArgs("stoi", args, 1); err != nil {
			return err
		}
		s, ok := args[0].(*objects.String)
		if !ok {
			return newError("stoi espera STRING, recibió %s", args[0].Type())
		}
		n, err := strconv.ParseInt(strings.TrimSpace(s.Value), 10, 64)
		if err != nil {
			return newError("stoi: \"%s\" no es un entero válido", s.Value)
		}
		return &objects.Integer{Value: n}
	}},
	"stod": {Name: "stod", Fn: func(args ...objects.Object) objects.Object {
		if err := checkArgs("stod", args, 1); err != nil {
			return err
		}
		s, ok := args[0].(*objects.String)
		if !ok {
			return newError("stod espera STRING, recibió %s", args[0].Type())
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(s.Value), 64)
		if err != nil {
			return newError("stod: \"%s\" no es un número válido", s.Value)
		}
		return &objects.Double{Value: f}
	}},
	"abs": {Name: "abs", Fn: func(args ...objects.Object) objects.Object {
		if err := checkArgs("abs", args, 1); err != nil {
			return err
		}
		switch v := args[0].(type) {
		case *objects.Integer:
			if v.Value < 0 {
				return &objects.Integer{Value: -v.Value}
			}
			return v
		case *objects.Double:
			return &objects.Double{Value: math.Abs(v.Value)}
		}
		return newError("abs no admite %s", args[0].Type())
	}},
	"max": {Name: "max", Fn: func(args ...objects.Object) objects.Object { return minMax("max", args) }},
	"min": {Name: "min", Fn: func(args ...objects.Object) objects.Object { return minMax("min", args) }},
	"pow": {Name: "pow", Fn: func(args ...objects.Object) objects.Object {
		if err := checkArgs("pow", args, 2); err != nil {
			return err
		}
		a, okA := toFloat(args[0])
		b, okB := toFloat(args[1])
		if !okA || !okB {
			return newError("pow espera números")
		}
		return &objects.Double{Value: math.Pow(a, b)}
	}},
	"sqrt": {Name: "sqrt", Fn: func(args ...objects.Object) objects.Object {
		if err := checkArgs("sqrt", args, 1); err != nil {
			return err
		}
		a, ok := toFloat(args[0])
		if !ok {
			return newError("sqrt espera un número")
		}
		if a < 0 {
			return newError("sqrt de un número negativo")
		}
		return &objects.Double{Value: math.Sqrt(a)}
	}},
}

// evalMethodCall ejecuta llamadas de la forma objeto.metodo(args).
func evalMethodCall(member *ast.MemberExpression, argExprs []ast.Expression, env *objects.Environment) objects.Object {
	receiver := Eval(member.Object, env)
	if isError(receiver) {
		return receiver
	}
	args, err := evalExpressions(argExprs, env)
	if err != nil {
		return err
	}
	name := member.Member.Value

	switch r := receiver.(type) {
	case *objects.String:
		switch name {
		case "size", "length":
			if e := checkArgs(name, args, 0); e != nil {
				return e
			}
			return &objects.Integer{Value: int64(len(r.Value))}
		}
	case *objects.Array:
		switch name {
		case "size":
			if e := checkArgs(name, args, 0); e != nil {
				return e
			}
			return &objects.Integer{Value: int64(len(r.Elements))}
		case "push_back":
			if e := checkArgs(name, args, 1); e != nil {
				return e
			}
			r.Elements = append(r.Elements, args[0])
			return objects.Null
		}
	}
	return newError("%s no tiene el método %s", receiver.Type(), name)
}

func checkArgs(name string, args []objects.Object, want int) *objects.Error {
	if len(args) != want {
		return newError("%s espera %d argumento(s), llegaron %d", name, want, len(args))
	}
	return nil
}

func toFloat(o objects.Object) (float64, bool) {
	switch v := o.(type) {
	case *objects.Integer:
		return float64(v.Value), true
	case *objects.Double:
		return v.Value, true
	}
	return 0, false
}

func minMax(name string, args []objects.Object) objects.Object {
	if err := checkArgs(name, args, 2); err != nil {
		return err
	}
	a, okA := toFloat(args[0])
	b, okB := toFloat(args[1])
	if !okA || !okB {
		return newError("%s espera números", name)
	}
	if (name == "max") == (a >= b) {
		return args[0]
	}
	return args[1]
}
