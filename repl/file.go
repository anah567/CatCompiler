package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"polaobesahermosa/evaluator"
	"polaobesahermosa/lexer"
	"polaobesahermosa/objects"
	"polaobesahermosa/parser"
)

// RunFile lee y ejecuta un programa .ssg completo. El archivo se entrega
// de una sola vez al parser: una función o un bloque puede ocupar varias
// líneas. env es el mismo del REPL cuando se usa :cargar, así que las
// variables definidas en el archivo quedan disponibles después.
//
// Primero se analiza todo el archivo; si hay errores de sintaxis, no se
// ejecuta ninguna sentencia. Un error durante la ejecución sí puede dejar
// cambios hechos por sentencias anteriores, como ocurre en el REPL.
func RunFile(path string, out io.Writer, env *objects.Environment) error {
	if filepath.Ext(path) != ".ssg" {
		return fmt.Errorf("el archivo debe tener extensión .ssg: %s", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("no se pudo leer %s: %w", path, err)
	}
	p := parser.New(lexer.New(string(content)))
	program := p.ParseProgram()
	if errors := p.Errors(); len(errors) > 0 {
		return fmt.Errorf("errores de sintaxis en %s:\n%s", path, strings.Join(errors, "\n"))
	}
	result := evaluator.Eval(program, env)
	if result.Type() == objects.ERROR_OBJ {
		return fmt.Errorf("error al ejecutar %s: %s", path, result.Inspect())
	}
	_, err = fmt.Fprintln(out, result.Inspect())
	return err
}
