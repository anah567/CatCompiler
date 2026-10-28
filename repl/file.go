// Paquete repl permite ejecutar código de forma interactiva
// y también ejecutar programas completos desde archivos .cpp.
package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"polaobesahermosa/ast"
	"polaobesahermosa/evaluator"
	"polaobesahermosa/lexer"
	"polaobesahermosa/objects"
	"polaobesahermosa/parser"
	"polaobesahermosa/token"
)

// RunFile lee y ejecuta un programa .cpp completo.
// Utiliza el entorno recibido para guardar variables y funciones.
func RunFile(path string, out io.Writer, env *objects.Environment) error { // Configuramos el destino de salida del programa.
	// Así, cout escribirá donde indique RunFile.
	env.SetOutput(out)

	// Verificamos que el archivo tenga extensión .cpp.
	if filepath.Ext(path) != ".cpp" {
		return fmt.Errorf("el archivo debe tener extensión .cpp: %s", path)
	}

	// Leemos el contenido del archivo.
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("no se pudo leer %s: %w", path, err)
	}

	// Creamos el lexer y el parser.
	p := parser.New(lexer.New(string(content)))

	// Convertimos el código en un árbol de sintaxis (AST).
	program := p.ParseProgram()

	// Comprobamos si hay errores de sintaxis.
	if errors := p.Errors(); len(errors) > 0 {
		return fmt.Errorf(
			"errores de sintaxis en %s:\n%s",
			path,
			strings.Join(errors, "\n"),
		)
	}

	// Evaluamos el programa para registrar las funciones
	// y ejecutar las instrucciones del nivel principal.
	result := evaluator.Eval(program, env)

	// Comprobamos si ocurrió un error durante la evaluación.
	if result != nil && result.Type() == objects.ERROR_OBJ {
		return fmt.Errorf(
			"error al ejecutar %s: %s",
			path,
			result.Inspect(),
		)
	}

	// Buscamos si el programa declaró una función main.
	// Solo la ejecutamos automáticamente al cargar un archivo.
	_, existeMain := env.Get("main")

	if existeMain {

		// Construimos una llamada equivalente a escribir main().
		// Así reutilizamos la ejecución normal de funciones
		// que ya tiene implementada el evaluador.
		llamadaMain := &ast.CallExpression{
			Token: token.Token{
				Type:    token.LPAREN,
				Literal: "(",
			},
			Function: &ast.Identifier{
				Token: token.Token{
					Type:    token.IDENT,
					Literal: "main",
				},
				Value: "main",
			},
			Arguments: []ast.Expression{},
		}

		// Ejecutamos main() dentro del entorno del programa.
		resultadoMain := evaluator.Eval(llamadaMain, env)

		// Si main() produce un error, lo mostramos.
		if resultadoMain != nil &&
			resultadoMain.Type() == objects.ERROR_OBJ {
			return fmt.Errorf(
				"error al ejecutar main() en %s: %s",
				path,
				resultadoMain.Inspect(),
			)
		}

		// No imprimimos el valor de retorno de main.
		// Por ejemplo, return 0 no debe aparecer en pantalla.
		return nil
	}

	// Si el archivo no tiene main, conservamos el comportamiento
	// de ejecutar sus instrucciones directamente.
	if result != nil && result.Type() != objects.NULL_OBJ {
		_, err = fmt.Fprintln(out, result.Inspect())
		return err
	}

	return nil
}
