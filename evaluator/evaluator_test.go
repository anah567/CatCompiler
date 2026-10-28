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
