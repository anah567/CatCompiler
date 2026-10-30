package repl

import (
	"bytes"
	"strings"
	"testing"
)

// ejecutarREPL simula lo que una persona escribe
// y guarda todo lo que aparece en la terminal.
func ejecutarREPL(entrada string) string {
	var salida bytes.Buffer

	Start(strings.NewReader(entrada), &salida)

	return salida.String()
}

// TestREPLSeleccionarModos comprueba que se puedan
// seleccionar los tres modos del intérprete.
func TestREPLSeleccionarModos(t *testing.T) {
	pruebas := []struct {
		nombre   string
		entrada  string
		esperado string
	}{
		{
			"lexer",
			"1\n:salir\n",
			"lexer>>",
		},
		{
			"parser",
			"2\n:salir\n",
			"parser>>",
		},
		{
			"evaluator",
			"3\n:salir\n",
			"evaluator>>",
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			salida := ejecutarREPL(prueba.entrada)

			if !strings.Contains(salida, prueba.esperado) {
				t.Fatalf(
					"se esperaba encontrar %q en la salida:\n%s",
					prueba.esperado,
					salida,
				)
			}
		})
	}
}

// TestREPLCambiarModo comprueba que los comandos
// permitan cambiar entre Lexer, Parser y Evaluator.
func TestREPLCambiarModo(t *testing.T) {
	entrada := "1\n:parser\n:evaluator\n:lexer\n:salir\n"

	salida := ejecutarREPL(entrada)

	esperados := []string{
		"Modo parser:",
		"Modo evaluator:",
		"Modo lexer:",
	}

	for _, esperado := range esperados {
		if !strings.Contains(salida, esperado) {
			t.Fatalf(
				"no se encontró %q en la salida:\n%s",
				esperado,
				salida,
			)
		}
	}
}

// TestREPLEvaluarOperacion comprueba que el modo
// Evaluator ejecute una operación matemática.
func TestREPLEvaluarOperacion(t *testing.T) {
	entrada := "3\n5 + 3;\n:salir\n"

	salida := ejecutarREPL(entrada)

	if !strings.Contains(salida, "evaluator>> 8") {
		t.Fatalf(
			"se esperaba el resultado 8, se obtuvo:\n%s",
			salida,
		)
	}
}

// TestRELPersistenciaVariables comprueba que las variables
// se conserven entre instrucciones del REPL.
func TestRELPersistenciaVariables(t *testing.T) {
	entrada := "3\nint x = 10;\nx = 20;\nx;\n:salir\n"

	salida := ejecutarREPL(entrada)

	if !strings.Contains(salida, "evaluator>> 20") {
		t.Fatalf(
			"se esperaba que x conservara el valor 20:\n%s",
			salida,
		)
	}
}

// TestREPLComandoAyuda comprueba que se muestre
// la lista de comandos disponibles.
func TestREPLComandoAyuda(t *testing.T) {
	entrada := "1\n:ayuda\n:salir\n"

	salida := ejecutarREPL(entrada)

	esperados := []string{
		":lexer",
		":parser",
		":evaluator",
		":cargar",
		":salir",
	}

	for _, esperado := range esperados {
		if !strings.Contains(salida, esperado) {
			t.Fatalf(
				"no se encontró el comando %q:\n%s",
				esperado,
				salida,
			)
		}
	}
}

// TestREPLComandoDesconocido comprueba que
// se informe cuando un comando no existe.
func TestREPLComandoDesconocido(t *testing.T) {
	entrada := "1\n:inventado\n:salir\n"

	salida := ejecutarREPL(entrada)

	if !strings.Contains(salida, "Comando desconocido") {
		t.Fatalf(
			"se esperaba un mensaje de comando desconocido:\n%s",
			salida,
		)
	}
}

// TestREPLSalir comprueba que el comando :salir
// termine correctamente la sesión.
func TestREPLSalir(t *testing.T) {
	entrada := "1\n:salir\n"

	salida := ejecutarREPL(entrada)

	if !strings.Contains(salida, "Hasta luego") {
		t.Fatalf(
			"se esperaba el mensaje de despedida:\n%s",
			salida,
		)
	}
}

// TestREPLErrorSintaxis comprueba que el REPL
// muestre los errores encontrados por el Parser.
func TestREPLErrorSintaxis(t *testing.T) {
	entrada := "3\nint x = ;\n:salir\n"

	salida := ejecutarREPL(entrada)

	if !strings.Contains(salida, "errores de sintaxis") {
		t.Fatalf(
			"se esperaba un error de sintaxis:\n%s",
			salida,
		)
	}
}
