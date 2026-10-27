package parser

import (
	"polaobesahermosa/ast"
	"polaobesahermosa/lexer"
	"reflect"
	"testing"
)

// parseOK parsea la entrada y hace fallar la prueba si hubo algún error.
func parseOK(t *testing.T, input string) *ast.Program {
	t.Helper()
	p := New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("errores inesperados para %q:\n%v", input, p.Errors())
	}
	return program
}

// onlyStatement parsea la entrada, exige que tenga exactamente una
// sentencia y la devuelve.
func onlyStatement(t *testing.T, input string) ast.Statement {
	t.Helper()
	program := parseOK(t, input)
	if len(program.Statements) != 1 {
		t.Fatalf("se esperaba 1 sentencia, se obtuvieron %d: %s",
			len(program.Statements), program.String())
	}
	return program.Statements[0]
}

// strOrEmpty devuelve n.String() o "" si n es nil. Permite comparar campos
// opcionales en las tablas.
func strOrEmpty(n ast.Node) string {
	if n == nil || reflect.ValueOf(n).IsNil() {
		return ""
	}
	return n.String()
}

// strList convierte una lista de expresiones en textos (nil → nil).
func strList(list []ast.Expression) []string {
	if list == nil {
		return nil
	}
	out := []string{}
	for _, e := range list {
		out = append(out, e.String())
	}
	return out
}

// TestPrecedencia revisa cómo se agrupan los operadores. String() pone
// paréntesis alrededor de cada operación, así se ve el árbol que se armó.
func TestPrecedencia(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1 + 2 * 3;", "(1 + (2 * 3));"},
		{"1 * 2 + 3;", "((1 * 2) + 3);"},
		{"a + b - c;", "((a + b) - c);"},
		{"a * b / c % d;", "(((a * b) / c) % d);"},
		{"(1 + 2) * 3;", "((1 + 2) * 3);"},
		{"-a + b;", "((-a) + b);"},
		{"a - -b;", "(a - (-b));"},
		{"!a == b;", "((!a) == b);"},
		{"!true != false;", "((!true) != false);"},
		{"a < b == c > d;", "((a < b) == (c > d));"},
		{"a && b || c;", "((a && b) || c);"},
		{"a || b && c;", "(a || (b && c));"},
		{"a + b < c && d;", "(((a + b) < c) && d);"},
		{"1 << 2 + 3;", "(1 << (2 + 3));"},
		{"a << b << c;", "((a << b) << c);"},
		{"a = b = 3;", "(a = (b = 3));"},
		{"x = a || b;", "(x = (a || b));"},
		{"x += y * 2;", "(x += (y * 2));"},
		{"x -= 1;", "(x -= 1);"},
		{"x *= 2;", "(x *= 2);"},
		{"x /= 2;", "(x /= 2);"},
		{"x %= 2;", "(x %= 2);"},
		{"i++ + 1;", "((i++) + 1);"},
		{"++i * 2;", "((++i) * 2);"},
		{"-i--;", "(-(i--));"},
		{"a+++b;", "((a++) + b);"},
		{"arr[i]++;", "((arr[i])++);"},
		{"-a[0];", "(-(a[0]));"},
		{"a * f(b);", "(a * f(b));"},
		{"f(a, b + 1)[2];", "(f(a, (b + 1))[2]);"},
		{"g();", "g();"},
		{"x = 3.5 * 2;", "(x = (3.5 * 2));"},
		{`s = "a\n";`, `(s = "a\n");`},
		{"arr[i] = arr[i - 1] + 1;", "((arr[i]) = ((arr[(i - 1)]) + 1));"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseOK(t, tt.input).String()
			if got != tt.want {
				t.Fatalf("se esperaba %s, se obtuvo %s", tt.want, got)
			}
		})
	}
}

// TestNodosDeExpresion revisa el tipo de nodo y los campos que produce
// cada clase de expresión.
func TestNodosDeExpresion(t *testing.T) {
	t.Run("DoubleLiteral", func(t *testing.T) {
		stmt := onlyStatement(t, "3.25;").(*ast.ExpressionStatement)
		lit, ok := stmt.Expression.(*ast.DoubleLiteral)
		if !ok || lit.Value != 3.25 {
			t.Fatalf("se esperaba DoubleLiteral 3.25, se obtuvo %#v", stmt.Expression)
		}
	})
	t.Run("IntegerLiteral", func(t *testing.T) {
		stmt := onlyStatement(t, "42;").(*ast.ExpressionStatement)
		lit, ok := stmt.Expression.(*ast.IntegerLiteral)
		if !ok || lit.Value != 42 {
			t.Fatalf("se esperaba IntegerLiteral 42, se obtuvo %#v", stmt.Expression)
		}
	})
	t.Run("StringLiteral con escape", func(t *testing.T) {
		stmt := onlyStatement(t, `"a\tb";`).(*ast.ExpressionStatement)
		lit, ok := stmt.Expression.(*ast.StringLiteral)
		if !ok || lit.Value != "a\tb" {
			t.Fatalf("se esperaba StringLiteral \"a\\tb\", se obtuvo %#v", stmt.Expression)
		}
	})
	t.Run("Boolean", func(t *testing.T) {
		stmt := onlyStatement(t, "false;").(*ast.ExpressionStatement)
		b, ok := stmt.Expression.(*ast.Boolean)
		if !ok || b.Value != false {
			t.Fatalf("se esperaba Boolean false, se obtuvo %#v", stmt.Expression)
		}
	})
	t.Run("PrefixExpression ++", func(t *testing.T) {
		stmt := onlyStatement(t, "++x;").(*ast.ExpressionStatement)
		pe, ok := stmt.Expression.(*ast.PrefixExpression)
		if !ok || pe.Operator != "++" || pe.Right.String() != "x" {
			t.Fatalf("se esperaba PrefixExpression ++x, se obtuvo %#v", stmt.Expression)
		}
	})
	t.Run("PostfixExpression --", func(t *testing.T) {
		stmt := onlyStatement(t, "x--;").(*ast.ExpressionStatement)
		pe, ok := stmt.Expression.(*ast.PostfixExpression)
		if !ok || pe.Operator != "--" || pe.Left.String() != "x" {
			t.Fatalf("se esperaba PostfixExpression x--, se obtuvo %#v", stmt.Expression)
		}
	})
	t.Run("AssignExpression compuesta", func(t *testing.T) {
		stmt := onlyStatement(t, "total += arr[i];").(*ast.ExpressionStatement)
		ae, ok := stmt.Expression.(*ast.AssignExpression)
		if !ok || ae.Operator != "+=" || ae.Left.String() != "total" ||
			ae.Value.String() != "(arr[i])" {
			t.Fatalf("se esperaba total += arr[i], se obtuvo %#v", stmt.Expression)
		}
		if _, ok := ae.Value.(*ast.IndexExpression); !ok {
			t.Fatalf("el valor debía ser IndexExpression, es %T", ae.Value)
		}
	})
	t.Run("CallExpression", func(t *testing.T) {
		stmt := onlyStatement(t, "suma(1, 2 * 3);").(*ast.ExpressionStatement)
		call, ok := stmt.Expression.(*ast.CallExpression)
		if !ok || call.Function.String() != "suma" ||
			!reflect.DeepEqual(strList(call.Arguments), []string{"1", "(2 * 3)"}) {
			t.Fatalf("se esperaba suma(1, 2 * 3), se obtuvo %#v", stmt.Expression)
		}
	})
	t.Run("posición del token", func(t *testing.T) {
		stmt := onlyStatement(t, "\n  a / b;").(*ast.ExpressionStatement)
		ie := stmt.Expression.(*ast.InfixExpression)
		if ie.Token.Line != 2 || ie.Token.Column != 5 {
			t.Fatalf("el '/' debía estar en 2:5, está en %d:%d", ie.Token.Line, ie.Token.Column)
		}
	})
}

// TestVarDeclaration revisa los campos de cada forma de declarar una variable.
func TestVarDeclaration(t *testing.T) {
	tests := []struct {
		input     string
		typ       string
		name      string
		isConst   bool
		isArray   bool
		arraySize string   // "" = nil
		value     string   // "" = nil
		elements  []string // nil = sin lista
		str       string   // String() del nodo
	}{
		{"int edad = 20;", "int", "edad", false, false, "", "20", nil, "int edad = 20;"},
		{"double p = 3.5;", "double", "p", false, false, "", "3.5", nil, "double p = 3.5;"},
		{"bool b = true;", "bool", "b", false, false, "", "true", nil, "bool b = true;"},
		{`string s = "hola";`, "string", "s", false, false, "", `"hola"`, nil, `string s = "hola";`},
		{"int x;", "int", "x", false, false, "", "", nil, "int x;"},
		{"int y = x * 2 + 1;", "int", "y", false, false, "", "((x * 2) + 1)", nil, "int y = ((x * 2) + 1);"},
		{"const int N = 5;", "int", "N", true, false, "", "5", nil, "const int N = 5;"},
		{"int a[5];", "int", "a", false, true, "5", "", nil, "int a[5];"},
		{"int a[N + 1];", "int", "a", false, true, "(N + 1)", "", nil, "int a[(N + 1)];"},
		{"int arr[3] = {1, 2, 3};", "int", "arr", false, true, "3", "",
			[]string{"1", "2", "3"}, "int arr[3] = {1, 2, 3};"},
		{"int v[] = {4, 5};", "int", "v", false, true, "", "",
			[]string{"4", "5"}, "int v[] = {4, 5};"},
		{"double d[2] = {};", "double", "d", false, true, "2", "",
			[]string{}, "double d[2] = {};"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			decl, ok := onlyStatement(t, tt.input).(*ast.VarDeclaration)
			if !ok {
				t.Fatalf("se esperaba *ast.VarDeclaration")
			}
			if decl.Type != tt.typ || decl.Name.Value != tt.name ||
				decl.Const != tt.isConst || decl.IsArray != tt.isArray {
				t.Fatalf("campos: Type=%q Name=%q Const=%v IsArray=%v",
					decl.Type, decl.Name.Value, decl.Const, decl.IsArray)
			}
			if got := strOrEmpty(decl.ArraySize); got != tt.arraySize {
				t.Fatalf("ArraySize: se esperaba %q, se obtuvo %q", tt.arraySize, got)
			}
			if got := strOrEmpty(decl.Value); got != tt.value {
				t.Fatalf("Value: se esperaba %q, se obtuvo %q", tt.value, got)
			}
			if got := strList(decl.Elements); !reflect.DeepEqual(got, tt.elements) {
				t.Fatalf("Elements: se esperaba %#v, se obtuvo %#v", tt.elements, got)
			}
			if decl.String() != tt.str {
				t.Fatalf("String: se esperaba %q, se obtuvo %q", tt.str, decl.String())
			}
		})
	}
}

// TestVarDeclarationList revisa la declaración de varias variables en una
// sola sentencia.
func TestVarDeclarationList(t *testing.T) {
	list, ok := onlyStatement(t, "int a = 1, b, c[2];").(*ast.VarDeclarationList)
	if !ok {
		t.Fatalf("se esperaba *ast.VarDeclarationList")
	}
	if len(list.Declarations) != 3 {
		t.Fatalf("se esperaban 3 declaraciones, hay %d", len(list.Declarations))
	}
	names := []string{}
	for _, d := range list.Declarations {
		if d.Type != "int" {
			t.Fatalf("todas debían ser int, %s es %s", d.Name.Value, d.Type)
		}
		names = append(names, d.Name.Value)
	}
	if !reflect.DeepEqual(names, []string{"a", "b", "c"}) {
		t.Fatalf("nombres: %v", names)
	}
	if !list.Declarations[2].IsArray {
		t.Fatalf("c debía ser arreglo")
	}
	if list.String() != "int a = 1, b, c[2];" {
		t.Fatalf("String: %q", list.String())
	}
}

// TestFunctionDeclaration revisa definiciones, prototipos y parámetros.
func TestFunctionDeclaration(t *testing.T) {
	tests := []struct {
		input      string
		returnType string
		name       string
		params     []string // Param.String() de cada parámetro
		hasBody    bool
		str        string
	}{
		{"int suma(int a, int b) { return a + b; }", "int", "suma",
			[]string{"int a", "int b"}, true,
			"int suma(int a, int b) { return (a + b); }"},
		{"void saluda() { }", "void", "saluda", []string{}, true, "void saluda() { }"},
		{"int main() { return 0; }", "int", "main", []string{}, true,
			"int main() { return 0; }"},
		{"int factorial(int n) { return n * factorial(n - 1); }", "int", "factorial",
			[]string{"int n"}, true,
			"int factorial(int n) { return (n * factorial((n - 1))); }"},
		{"double promedio(int v[], int n) { return 0.0; }", "double", "promedio",
			[]string{"int v[]", "int n"}, true,
			"double promedio(int v[], int n) { return 0.0; }"},
		{"int f(int);", "int", "f", []string{"int"}, false, "int f(int);"},
		{"bool g(string s, double);", "bool", "g", []string{"string s", "double"}, false,
			"bool g(string s, double);"},
		{"void nada() { return; }", "void", "nada", []string{}, true, "void nada() { return; }"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			fn, ok := onlyStatement(t, tt.input).(*ast.FunctionDeclaration)
			if !ok {
				t.Fatalf("se esperaba *ast.FunctionDeclaration")
			}
			if fn.ReturnType != tt.returnType || fn.Name.Value != tt.name {
				t.Fatalf("ReturnType=%q Name=%q", fn.ReturnType, fn.Name.Value)
			}
			params := []string{}
			for _, pa := range fn.Params {
				params = append(params, pa.String())
			}
			if !reflect.DeepEqual(params, tt.params) {
				t.Fatalf("Params: se esperaba %v, se obtuvo %v", tt.params, params)
			}
			if (fn.Body != nil) != tt.hasBody {
				t.Fatalf("Body: se esperaba cuerpo=%v", tt.hasBody)
			}
			if fn.String() != tt.str {
				t.Fatalf("String: se esperaba %q, se obtuvo %q", tt.str, fn.String())
			}
		})
	}
}

// TestSentenciasSimples revisa return, bloques sueltos, using y la
// sentencia vacía.
func TestSentenciasSimples(t *testing.T) {
	t.Run("return sin valor", func(t *testing.T) {
		rs := onlyStatement(t, "return;").(*ast.ReturnStatement)
		if rs.ReturnValue != nil {
			t.Fatalf("ReturnValue debía ser nil")
		}
	})
	t.Run("return con valor", func(t *testing.T) {
		rs := onlyStatement(t, "return x + 1;").(*ast.ReturnStatement)
		if rs.ReturnValue.String() != "(x + 1)" {
			t.Fatalf("ReturnValue: %s", rs.ReturnValue.String())
		}
	})
	t.Run("bloque suelto", func(t *testing.T) {
		block := onlyStatement(t, "{ int y = 2; y++; }").(*ast.BlockStatement)
		if len(block.Statements) != 2 {
			t.Fatalf("se esperaban 2 sentencias en el bloque")
		}
	})
	t.Run("using namespace std no genera nodo", func(t *testing.T) {
		if _, ok := onlyStatement(t, "using namespace std; int x;").(*ast.VarDeclaration); !ok {
			t.Fatalf("la única sentencia debía ser la declaración")
		}
	})
	t.Run("sentencia vacía", func(t *testing.T) {
		if n := len(parseOK(t, ";;").Statements); n != 0 {
			t.Fatalf("se esperaban 0 sentencias, hay %d", n)
		}
	})
}

// TestErrores revisa el mensaje EXACTO (tipo, línea, columna y texto) de
// cada error, y que no aparezcan errores en cascada.
func TestErrores(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "falta ';' al final",
			input: "int x = 5",
			want:  []string{"Error sintáctico [línea 1, columna 10]: se esperaba ';' pero se encontró el fin del archivo"},
		},
		{
			name:  "falta ';' antes de '}'",
			input: "int main() {\n  int x = 5\n}",
			want:  []string{"Error sintáctico [línea 3, columna 1]: se esperaba ';' pero se encontró '}'"},
		},
		{
			name:  "falta ';' en una expresión",
			input: "x = 1\ny = 2;",
			want:  []string{"Error sintáctico [línea 2, columna 1]: se esperaba ';' pero se encontró 'y'"},
		},
		{
			name:  "tipo sin nombre",
			input: "int ;",
			want:  []string{"Error sintáctico [línea 1, columna 5]: se esperaba un nombre después del tipo 'int' pero se encontró ';'"},
		},
		{
			name:  "paréntesis sin cerrar",
			input: "x = (1 + 2;",
			want:  []string{"Error sintáctico [línea 1, columna 11]: se esperaba ')' pero se encontró ';'"},
		},
		{
			name:  "bloque sin '}'",
			input: "int main() { int x = 1;",
			want:  []string{"Error sintáctico [línea 1, columna 24]: falta '}' para cerrar el bloque abierto en la línea 1"},
		},
		{
			name:  "expresión vacía",
			input: "int x = ;",
			want:  []string{"Error sintáctico [línea 1, columna 9]: se esperaba una expresión pero se encontró ';'"},
		},
		{
			name:  "asignar a un número",
			input: "5 = x;",
			want:  []string{"Error sintáctico [línea 1, columna 3]: no se puede asignar a 5: se esperaba una variable o un elemento de arreglo"},
		},
		{
			name:  "incrementar un número",
			input: "++5;",
			want:  []string{"Error sintáctico [línea 1, columna 1]: el operador ++ necesita una variable o un elemento de arreglo, no 5"},
		},
		{
			name:  "variable void",
			input: "void v;",
			want:  []string{"Error sintáctico [línea 1, columna 1]: la variable 'v' no puede ser de tipo void"},
		},
		{
			name:  "constante sin valor",
			input: "const int N;",
			want:  []string{"Error sintáctico [línea 1, columna 11]: la constante 'N' debe tener un valor inicial"},
		},
		{
			name:  "arreglo sin tamaño ni valores",
			input: "int a[];",
			want:  []string{"Error sintáctico [línea 1, columna 5]: el arreglo 'a' necesita un tamaño o una lista de valores"},
		},
		{
			name:  "función dentro de función",
			input: "int main() { int f() { return 1; } return 0; }",
			want:  []string{"Error sintáctico [línea 1, columna 18]: no se puede declarar la función 'f' dentro de un bloque (solo en el nivel superior)"},
		},
		{
			name:  "parámetro sin tipo",
			input: "int f(x) { }",
			want:  []string{"Error sintáctico [línea 1, columna 7]: se esperaba el tipo del parámetro pero se encontró 'x'"},
		},
		{
			name:  "parámetro sin nombre en una definición",
			input: "int f(int) { return 1; }",
			want:  []string{"Error sintáctico [línea 1, columna 5]: el parámetro 1 de la función 'f' necesita un nombre"},
		},
		{
			name:  "using incompleto",
			input: "using namespace;",
			want:  []string{"Error sintáctico [línea 1, columna 16]: se esperaba 'std' pero se encontró ';'"},
		},
		{
			name:  "error léxico: carácter no válido",
			input: "int x = 5 @;",
			want:  []string{"Error léxico [línea 1, columna 11]: carácter no válido '@'"},
		},
		{
			name:  "error léxico seguido de error sintáctico",
			input: `x = "hola;`,
			want: []string{
				"Error léxico [línea 1, columna 5]: cadena sin cerrar: falta '\"'",
				"Error sintáctico [línea 1, columna 11]: se esperaba una expresión pero se encontró el fin del archivo",
			},
		},
		{
			// Dos sentencias con error separadas por una correcta: se
			// reportan las dos, pero ninguna "en cascada".
			name:  "varios errores sin cascada",
			input: "int x = ;\nint y = 2;\ny = ;",
			want: []string{
				"Error sintáctico [línea 1, columna 9]: se esperaba una expresión pero se encontró ';'",
				"Error sintáctico [línea 3, columna 5]: se esperaba una expresión pero se encontró ';'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(lexer.New(tt.input))
			p.ParseProgram()
			if !reflect.DeepEqual(p.Errors(), tt.want) {
				t.Fatalf("errores:\nse esperaba %q\nse obtuvo   %q", tt.want, p.Errors())
			}
		})
	}
}
