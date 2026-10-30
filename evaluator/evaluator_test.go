// Estas pruebas comprueban que las variables declaradas dentro
// de un bloque no se puedan utilizar fuera de él.
package evaluator

import (
	"strings"
	"testing"

	"polaobesahermosa/lexer"
	"polaobesahermosa/objects"
	"polaobesahermosa/parser"
)

// ejecutarCodigo convierte el código en un AST y después lo evalúa.
// Si hay errores de sintaxis, la prueba se detiene.
func ejecutarCodigo(t *testing.T, codigo string) objects.Object {
	t.Helper()

	p := parser.New(lexer.New(codigo))
	programa := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Errores del parser: %v", p.Errors())
	}

	return Eval(programa, objects.NewEnvironment())
}

// TestVariableLocal comprueba que una variable declarada
// dentro de un if no exista fuera de ese bloque.
func TestVariableLocal(t *testing.T) {
	codigo := `
		int x = 10;
		if (true) {
			int y = 20;
			x = 30;
		}
		y;
	`

	resultado := ejecutarCodigo(t, codigo)

	errorResultado, ok := resultado.(*objects.Error)
	if !ok {
		t.Fatalf("Se esperaba un error, se obtuvo: %T (%s)",
			resultado, resultado.Inspect())
	}

	if !strings.Contains(errorResultado.Message, "identificador no definido") {
		t.Fatalf("Error inesperado: %s", errorResultado.Message)
	}
}

// TestModificarVariableExterna comprueba que un bloque
// pueda modificar una variable declarada fuera de él.
func TestModificarVariableExterna(t *testing.T) {
	codigo := `
		int x = 10;
		if (true) {
			x = 30;
		}
		x;
	`

	resultado := ejecutarCodigo(t, codigo)

	entero, ok := resultado.(*objects.Integer)
	if !ok {
		t.Fatalf("Se esperaba un entero, se obtuvo: %T (%s)",
			resultado, resultado.Inspect())
	}

	if entero.Value != 30 {
		t.Fatalf("Se esperaba 30, se obtuvo %d", entero.Value)
	}
}

// TestVariableLocalFor comprueba que una variable declarada
// en el encabezado de un for no exista después del ciclo.
func TestVariableLocalFor(t *testing.T) {
	codigo := `
		for (int i = 0; i < 3; i++) {
			int temporal = i;
		}
		i;
	`

	resultado := ejecutarCodigo(t, codigo)

	errorResultado, ok := resultado.(*objects.Error)
	if !ok {
		t.Fatalf(
			"Se esperaba un error, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}

	if !strings.Contains(errorResultado.Message, "identificador no definido") {
		t.Fatalf("Error inesperado: %s", errorResultado.Message)
	}
}

// TestIncrementoArreglo comprueba los incrementos
// y decrementos sobre posiciones de un arreglo.
func TestIncrementoArreglo(t *testing.T) {
	codigo := `
		int numeros[3] = {10, 20, 30};

		numeros[0]++;
		++numeros[1];
		numeros[2]--;

		numeros[0] + numeros[1] + numeros[2];
	`

	resultado := ejecutarCodigo(t, codigo)

	entero, ok := resultado.(*objects.Integer)
	if !ok {
		t.Fatalf(
			"Se esperaba un entero, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}

	// 11 + 21 + 29 = 61
	if entero.Value != 61 {
		t.Fatalf(
			"Se esperaba 61, se obtuvo %d",
			entero.Value,
		)
	}
}

// TestModificarArregloConstante comprueba que no se pueda
// cambiar una posición de un arreglo declarado como const.
func TestModificarArregloConstante(t *testing.T) {
	codigo := `
		const int numeros[3] = {10, 20, 30};
		numeros[0] = 99;
	`

	resultado := ejecutarCodigo(t, codigo)

	errorResultado, ok := resultado.(*objects.Error)
	if !ok {
		t.Fatalf(
			"Se esperaba un error, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}

	if !strings.Contains(errorResultado.Message, "constante") {
		t.Fatalf("Error inesperado: %s", errorResultado.Message)
	}
}

// TestIncrementarArregloConstante comprueba que no se
// pueda incrementar una posición de un arreglo const.
func TestIncrementarArregloConstante(t *testing.T) {
	codigo := `
		const int numeros[3] = {10, 20, 30};
		numeros[0]++;
	`

	resultado := ejecutarCodigo(t, codigo)

	errorResultado, ok := resultado.(*objects.Error)
	if !ok {
		t.Fatalf(
			"Se esperaba un error, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}

	if !strings.Contains(errorResultado.Message, "constante") {
		t.Fatalf("Error inesperado: %s", errorResultado.Message)
	}
}

// TestFuncionTipada comprueba que una función reciba
// parámetros y devuelva el resultado esperado.
func TestFuncionTipada(t *testing.T) {
	codigo := `
		int sumar(int a, int b) {
			return a + b;
		}
		sumar(10, 20);
	`

	resultado := ejecutarCodigo(t, codigo)

	entero, ok := resultado.(*objects.Integer)
	if !ok {
		t.Fatalf(
			"Se esperaba un entero, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}

	if entero.Value != 30 {
		t.Fatalf("Se esperaba 30, se obtuvo %d", entero.Value)
	}
}

// TestFuncionTipoIncorrecto comprueba que se detecte
// un argumento incompatible con el tipo declarado.
func TestFuncionTipoIncorrecto(t *testing.T) {
	codigo := `
		int duplicar(int numero) {
			return numero * 2;
		}
		duplicar("hola");
	`

	resultado := ejecutarCodigo(t, codigo)

	if _, ok := resultado.(*objects.Error); !ok {
		t.Fatalf(
			"Se esperaba un error de tipos, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}
}

// TestFuncionRetornoIncorrecto comprueba que una función
// no pueda devolver un tipo diferente al declarado.
func TestFuncionRetornoIncorrecto(t *testing.T) {
	codigo := `
		int obtenerNumero() {
			return "hola";
		}
		obtenerNumero();
	`

	resultado := ejecutarCodigo(t, codigo)

	if _, ok := resultado.(*objects.Error); !ok {
		t.Fatalf(
			"Se esperaba un error de retorno, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}
}

// TestFuncionRecursiva comprueba que una función
// pueda llamarse a sí misma para calcular un factorial.
func TestFuncionRecursiva(t *testing.T) {
	codigo := `
		int factorial(int n) {
			if (n <= 1) {
				return 1;
			}
			return n * factorial(n - 1);
		}
		factorial(5);
	`

	resultado := ejecutarCodigo(t, codigo)

	entero, ok := resultado.(*objects.Integer)
	if !ok {
		t.Fatalf(
			"Se esperaba un entero, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}

	if entero.Value != 120 {
		t.Fatalf("Se esperaba 120, se obtuvo %d", entero.Value)
	}
}

// TestCortocircuitoAND comprueba que && no evalúe
// la derecha cuando la izquierda es falsa.
func TestCortocircuitoAND(t *testing.T) {
	codigo := `
		false && (10 / 0 > 1);
	`

	resultado := ejecutarCodigo(t, codigo)

	booleano, ok := resultado.(*objects.Boolean)
	if !ok || booleano.Value {
		t.Fatalf(
			"Se esperaba false, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}
}

// TestCortocircuitoOR comprueba que || no evalúe
// la derecha cuando la izquierda es verdadera.
func TestCortocircuitoOR(t *testing.T) {
	codigo := `
		true || (10 / 0 > 1);
	`

	resultado := ejecutarCodigo(t, codigo)

	booleano, ok := resultado.(*objects.Boolean)
	if !ok || !booleano.Value {
		t.Fatalf(
			"Se esperaba true, se obtuvo: %T (%s)",
			resultado,
			resultado.Inspect(),
		)
	}
}

// TestOperacionesMatematicas comprueba las operaciones básicas.
func TestOperacionesMatematicas(t *testing.T) {
	pruebas := []struct {
		nombre   string
		codigo   string
		esperado int64
	}{
		{"suma", "5 + 3;", 8},
		{"resta", "10 - 4;", 6},
		{"multiplicacion", "6 * 3;", 18},
		{"division", "20 / 4;", 5},
		{"modulo", "10 % 3;", 1},
		{"precedencia", "2 + 3 * 4;", 14},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			resultado := ejecutarCodigo(t, prueba.codigo)

			entero, ok := resultado.(*objects.Integer)
			if !ok {
				t.Fatalf("Se esperaba un entero, se obtuvo %T", resultado)
			}

			if entero.Value != prueba.esperado {
				t.Fatalf("Se esperaba %d, se obtuvo %d",
					prueba.esperado, entero.Value)
			}
		})
	}
}

// TestCondicionales comprueba que se ejecute la rama correcta.
func TestCondicionales(t *testing.T) {
	pruebas := []struct {
		nombre   string
		codigo   string
		esperado int64
	}{
		{
			"if verdadero",
			"int x = 0; if (true) { x = 10; } x;",
			10,
		},
		{
			"if falso",
			"int x = 0; if (false) { x = 10; } x;",
			0,
		},
		{
			"if else",
			"int x = 0; if (false) { x = 10; } else { x = 20; } x;",
			20,
		},
		{
			"else if",
			"int x = 0; if (false) { x = 10; } else if (true) { x = 30; } x;",
			30,
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			resultado := ejecutarCodigo(t, prueba.codigo)

			entero, ok := resultado.(*objects.Integer)
			if !ok {
				t.Fatalf("Se esperaba un entero, se obtuvo %T", resultado)
			}

			if entero.Value != prueba.esperado {
				t.Fatalf("Se esperaba %d, se obtuvo %d",
					prueba.esperado, entero.Value)
			}
		})
	}
} // TestCiclos comprueba que los ciclos se ejecuten correctamente.
func TestCiclos(t *testing.T) {
	pruebas := []struct {
		nombre   string
		codigo   string
		esperado int64
	}{
		{
			"while",
			`int x = 0;
			 while (x < 5) { x++; }
			 x;`,
			5,
		},
		{
			"for",
			`int suma = 0;
			 for (int i = 1; i <= 5; i++) {
			     suma += i;
			 }
			 suma;`,
			15,
		},
		{
			"do while",
			`int x = 0;
			 do { x++; } while (x < 3);
			 x;`,
			3,
		},
		{
			"do while ejecuta al menos una vez",
			`int x = 10;
			 do { x++; } while (x < 5);
			 x;`,
			11,
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			resultado := ejecutarCodigo(t, prueba.codigo)

			entero, ok := resultado.(*objects.Integer)
			if !ok {
				t.Fatalf("Se esperaba un entero, se obtuvo %T (%s)",
					resultado, resultado.Inspect())
			}

			if entero.Value != prueba.esperado {
				t.Fatalf("Se esperaba %d, se obtuvo %d",
					prueba.esperado, entero.Value)
			}
		})
	}
}

// TestBreakContinue comprueba las instrucciones break y continue.
func TestBreakContinue(t *testing.T) {
	pruebas := []struct {
		nombre   string
		codigo   string
		esperado int64
	}{
		{
			"break en for",
			`int x = 0;
			 for (int i = 0; i < 10; i++) {
			     if (i == 4) { break; }
			     x++;
			 }
			 x;`,
			4,
		},
		{
			"continue en for",
			`int suma = 0;
			 for (int i = 0; i < 5; i++) {
			     if (i == 2) { continue; }
			     suma += i;
			 }
			 suma;`,
			8,
		},
		{
			"break en while",
			`int x = 0;
			 while (true) {
			     x++;
			     if (x == 3) { break; }
			 }
			 x;`,
			3,
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			resultado := ejecutarCodigo(t, prueba.codigo)

			entero, ok := resultado.(*objects.Integer)
			if !ok {
				t.Fatalf("Se esperaba un entero, se obtuvo %T (%s)",
					resultado, resultado.Inspect())
			}

			if entero.Value != prueba.esperado {
				t.Fatalf("Se esperaba %d, se obtuvo %d",
					prueba.esperado, entero.Value)
			}
		})
	}
}

// TestArreglosAdicionales comprueba lectura y modificación de arreglos.
func TestArreglosAdicionales(t *testing.T) {
	pruebas := []struct {
		nombre   string
		codigo   string
		esperado int64
	}{
		{
			"leer elemento",
			`int a[3] = {10, 20, 30}; a[1];`,
			20,
		},
		{
			"modificar elemento",
			`int a[3] = {10, 20, 30}; a[1] = 50; a[1];`,
			50,
		},
		{
			"sumar elementos",
			`int a[3] = {10, 20, 30}; a[0] + a[1] + a[2];`,
			60,
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			resultado := ejecutarCodigo(t, prueba.codigo)

			entero, ok := resultado.(*objects.Integer)
			if !ok {
				t.Fatalf("Se esperaba un entero, se obtuvo %T (%s)",
					resultado, resultado.Inspect())
			}

			if entero.Value != prueba.esperado {
				t.Fatalf("Se esperaba %d, se obtuvo %d",
					prueba.esperado, entero.Value)
			}
		})
	}
}

// TestConstantesAdicionales comprueba que no puedan modificarse.
func TestConstantesAdicionales(t *testing.T) {
	codigo := `
		const int x = 10;
		x = 20;
	`

	resultado := ejecutarCodigo(t, codigo)

	err, ok := resultado.(*objects.Error)
	if !ok {
		t.Fatalf("Se esperaba un error, se obtuvo %T (%s)",
			resultado, resultado.Inspect())
	}

	if !strings.Contains(err.Message, "constante") {
		t.Fatalf("Error inesperado: %s", err.Message)
	}
}

// TestErroresEjecucion comprueba errores durante la evaluación.
func TestErroresEjecucion(t *testing.T) {
	pruebas := []struct {
		nombre string
		codigo string
	}{
		{"division por cero", "10 / 0;"},
		{"variable inexistente", "variableQueNoExiste;"},
		{"indice fuera de rango", "int a[2] = {1, 2}; a[5];"},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			resultado := ejecutarCodigo(t, prueba.codigo)

			if _, ok := resultado.(*objects.Error); !ok {
				t.Fatalf("Se esperaba un error, se obtuvo %T (%s)",
					resultado, resultado.Inspect())
			}
		})
	}
}
