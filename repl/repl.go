// Paquete repl implementa el REPL (Read-Eval-Print Loop) de CatCompiler:
// un ciclo interactivo que
//
//	Read  → lee una línea que escribe el usuario,
//	Eval  → la procesa con el lexer, parser o evaluador,
//	Print → muestra el resultado,
//	Loop  → y vuelve a empezar.
//
// El REPL tiene tres MODOS:
//   - lexer: muestra la lista de tokens de la línea.
//   - parser: muestra el programa reconstruido y el AST en forma de árbol,
//     o los errores de sintaxis si los hay.
//   - evaluator: ejecuta el código y muestra su valor o error.
//
// El modo se elige al iniciar y se puede cambiar con :lexer, :parser y
// :evaluator. El comando :cargar ejecuta un archivo de código.
// Todos empiezan con ':' para no confundirse con código del lenguaje.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"polaobesahermosa/ast"
	"polaobesahermosa/evaluator"
	"polaobesahermosa/lexer"
	"polaobesahermosa/objects"
	"polaobesahermosa/parser"
	"polaobesahermosa/token"
)

// Modos de funcionamiento del REPL. Se usa el mismo texto en el prompt
// ("lexer>> " o "parser>> ") para que el usuario sepa en qué modo está.
const (
	modoLexer     = "lexer"
	modoParser    = "parser"
	modoEvaluator = "evaluator"
)

// Start inicia el REPL.
//
// Recibe un io.Reader (de dónde leer) y un io.Writer (dónde escribir) en
// lugar de usar directamente la consola. En main.go se le pasan os.Stdin y
// os.Stdout, pero gracias a esto también se podría probar con un texto o
// un archivo como entrada.
//
// Termina cuando el usuario escribe :salir o cuando se acaba la entrada
// (por ejemplo al presionar Ctrl+D).
func Start(in io.Reader, out io.Writer) {
	// bufio.Scanner lee la entrada línea por línea.
	scanner := bufio.NewScanner(in)

	// 1. Se pregunta qué se quiere probar.
	modo, ok := elegirModo(scanner, out)
	if !ok {
		return // la entrada terminó sin elegir modo
	}

	// Un solo entorno vive durante toda la sesión: una variable declarada
	// en una línea puede utilizarse en la siguiente.
	env := objects.NewEnvironment()

	// Configuramos la salida que utilizará cout en el REPL.
	env.SetOutput(out)

	imprimirAyuda(out)

	// 2. Ciclo principal del REPL.
	for {
		// El prompt muestra el modo actual, por ejemplo "parser>> ".
		fmt.Fprintf(out, "%s>> ", modo)

		// Scan lee la siguiente línea; devuelve false al final de la entrada.
		if !scanner.Scan() {
			return
		}

		// Se quitan espacios al inicio y al final; las líneas vacías se ignoran.
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 3. Si la línea empieza con ':' es un comando del REPL, no código.
		if strings.HasPrefix(line, ":") {
			// :cargar recibe el resto de la línea como ruta. Se reconoce
			// antes del switch porque, a diferencia de los otros comandos,
			// incluye un argumento que cambia cada vez.
			if line == ":cargar" || strings.HasPrefix(line, ":cargar ") {
				path := strings.TrimSpace(strings.TrimPrefix(line, ":cargar"))
				if path == "" {
					fmt.Fprintln(out, "Uso: :cargar ruta/al/archivo.cpp")
				} else if err := RunFile(path, out, env); err != nil {
					fmt.Fprintln(out, err)
				}
				continue
			}

			switch line {
			case ":lexer":
				modo = modoLexer
				fmt.Fprintln(out, "Modo lexer: se mostrarán los tokens.")

			case ":parser":
				modo = modoParser
				fmt.Fprintln(out, "Modo parser: se mostrará el AST.")

			case ":evaluator":
				modo = modoEvaluator
				fmt.Fprintln(out, "Modo evaluator: se ejecutará el código.")

			case ":ayuda":
				imprimirAyuda(out)

			case ":salir":
				fmt.Fprintln(out, "¡Miau! Hasta luego.")
				return

			default:
				fmt.Fprintf(
					out,
					"Comando desconocido %q. Escribe :ayuda\n",
					line,
				)
			}

			continue // se vuelve a mostrar el prompt
		}

		// 4. Si no es comando, es código: se procesa según el modo actual.
		switch modo {
		case modoLexer:
			probarLexer(out, line)

		case modoParser:
			probarParser(out, line)

		case modoEvaluator:
			probarEvaluator(out, line, env)
		}
	}
}

// elegirModo muestra un menú para escoger lexer, parser o evaluator.
// Acepta tanto el número como el nombre, sin importar mayúsculas. Si la
// respuesta no es válida, vuelve a preguntar.
//
// Devuelve el modo elegido y true, o "" y false si la entrada se terminó
// antes de elegir (en ese caso el REPL debe cerrarse).
func elegirModo(scanner *bufio.Scanner, out io.Writer) (string, bool) {
	for {
		fmt.Fprintln(out, "\n¿Qué quieres probar?")
		fmt.Fprintln(out, "  1) Lexer  (ver los tokens)")
		fmt.Fprintln(out, "  2) Parser (ver el AST)")
		fmt.Fprintln(out, "  3) Evaluator (ejecutar código)")
		fmt.Fprint(out, "Opción: ")

		if !scanner.Scan() {
			return "", false
		}

		switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
		case "1", "lexer":
			return modoLexer, true

		case "2", "parser":
			return modoParser, true

		case "3", "evaluator":
			return modoEvaluator, true

		default:
			fmt.Fprintln(out, "Opción no válida, escribe 1, 2 o 3.")
		}
	}
}

// imprimirAyuda muestra la lista de comandos disponibles en el REPL.
func imprimirAyuda(out io.Writer) {
	fmt.Fprintln(out, "\nComandos:")
	fmt.Fprintln(out, "  :lexer     cambiar a modo lexer")
	fmt.Fprintln(out, "  :parser    cambiar a modo parser")
	fmt.Fprintln(out, "  :evaluator cambiar a modo evaluator")
	fmt.Fprintln(out, "  :cargar archivo.cpp ejecutar un archivo en el entorno actual")
	fmt.Fprintln(out, "  :ayuda     mostrar esta ayuda")
	fmt.Fprintln(out, "  :salir     salir del REPL")
	fmt.Fprintln(out)
}

// probarLexer pasa la línea por el lexer e imprime cada token, uno por
// renglón, hasta llegar a EOF (que no se imprime). Por ejemplo, "x + 1"
// muestra:
//
//	{Type:IDENT Literal:x}
//	{Type:+ Literal:+}
//	{Type:INT Literal:1}
//
// El formato %+v de fmt imprime una estructura con los nombres de sus campos.
func probarLexer(out io.Writer, line string) {
	l := lexer.New(line)

	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		fmt.Fprintf(out, "%+v\n", tok)
	}
}

// probarParser pasa la línea por el lexer y el parser, y muestra:
//   - si hubo errores de sintaxis: la lista de errores (y nada más, porque
//     el AST puede estar incompleto y tener nodos nil);
//   - si no: el programa reconstruido con String(), donde los paréntesis
//     muestran cómo se agruparon los operadores, y el AST dibujado como árbol.
func probarParser(out io.Writer, line string) {
	l := lexer.New(line)        // el lexer convierte el texto en tokens
	p := parser.New(l)          // el parser consume esos tokens
	program := p.ParseProgram() // y construye el AST

	if len(p.Errors()) > 0 {
		imprimirErroresParser(out, p.Errors())
		return
	}

	fmt.Fprintln(out, "Programa:")
	fmt.Fprintf(out, "  %s\n", program.String())

	fmt.Fprintln(out, "AST:")
	imprimirArbol(out, program, "", "")
}

// probarEvaluator repite las dos primeras fases (lexer y parser), revisa
// errores de sintaxis y solo entonces ejecuta el AST. Las declaraciones
// producen null; mostrarlo deja claro que no son expresiones con resultado.
func probarEvaluator(out io.Writer, line string, env *objects.Environment) {
	p := parser.New(lexer.New(line))
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		imprimirErroresParser(out, p.Errors())
		return
	}

	result := evaluator.Eval(program, env)

	if result != nil {
		fmt.Fprintln(out, result.Inspect())
	}
}

// imprimirErroresParser muestra los errores de sintaxis encontrados, uno
// por renglón, junto con Pola, la mascota del compilador.
func imprimirErroresParser(out io.Writer, errors []string) {
	fmt.Fprintln(out, " /\\_/\\   ¡Miau! Pola encontró errores de sintaxis:")
	fmt.Fprintln(out, "( o.o )")

	for _, msg := range errors {
		fmt.Fprintf(out, "\t%s\n", msg)
	}
}

// ---------------------------------------------------------------------------
// Impresión del AST en forma de árbol
//
// Se hace en dos partes:
//   - describirNodo sabe, para cada TIPO de nodo, qué texto mostrar y
//     cuáles son sus hijos.
//   - imprimirArbol recorre el árbol recursivamente y dibuja las ramas
//     (├── └── │) sin necesidad de saber qué tipo de nodo es cada uno.
// ---------------------------------------------------------------------------

// hijo representa un nodo hijo junto con una etiqueta que dice qué papel
// cumple dentro de su padre (por ejemplo "Condición", "Cuerpo" o
// "Izquierda"). La etiqueta puede ser "" si no hace falta.
type hijo struct {
	etiqueta string
	nodo     ast.Node
}

// imprimirArbol dibuja 'nodo' y, recursivamente, todos sus descendientes.
//
//   - etiqueta: papel del nodo dentro de su padre ("" para la raíz).
//   - prefijo: texto que va al inicio de cada renglón de los HIJOS de
//     este nodo. Se construye con "│   " (si el ancestro todavía tiene
//     hermanos debajo, hay que seguir dibujando su línea vertical) o con
//     "    " (si el ancestro era el último, ya no hay línea que dibujar).
//
// Ejemplo de salida para "int x = 1 + 2":
//
//	Program
//	└── VarDeclaration int x
//	    └── Valor: InfixExpression +
//	        ├── Izquierda: IntegerLiteral 1
//	        └── Derecha: IntegerLiteral 2
func imprimirArbol(out io.Writer, nodo ast.Node, etiqueta, prefijo string) {
	// Se imprime el propio nodo (el prefijo y la rama ya los escribió el padre).
	descripcion, hijos := describirNodo(nodo)

	if etiqueta != "" {
		descripcion = etiqueta + ": " + descripcion
	}

	fmt.Fprintln(out, descripcion)

	// Luego cada hijo, con su rama.
	for i, h := range hijos {
		ultimo := i == len(hijos)-1

		// rama: lo que se dibuja justo antes del hijo.
		// sangria: lo que se agrega al prefijo para los hijos de ese hijo.
		// El último hijo usa └── y ya no continúa la línea vertical.
		rama, sangria := "├── ", "│   "

		if ultimo {
			rama, sangria = "└── ", "    "
		}

		fmt.Fprint(out, prefijo+rama)
		imprimirArbol(out, h.nodo, h.etiqueta, prefijo+sangria)
	}
}

// describirNodo devuelve, para un nodo dado:
//   - el texto que lo describe (tipo de nodo + dato principal, por ejemplo
//     "InfixExpression +" o "Identifier x"), y
//   - la lista de sus hijos, cada uno con su etiqueta.
//
// Usa un "type switch" (switch n := nodo.(type)) para averiguar el tipo
// concreto del nodo; dentro de cada case, 'n' ya tiene ese tipo concreto y
// se puede acceder a sus campos.
//
// Si se agrega un nodo nuevo al AST, hay que agregar aquí su case; si no,
// se mostrará solo su nombre de tipo (caso default) sin hijos.
func describirNodo(nodo ast.Node) (string, []hijo) {
	switch n := nodo.(type) {
	case *ast.Program:
		return "Program", sentencias(n.Statements)

	// --- Declaraciones y sentencias ---

	case *ast.VarDeclaration:
		// Muestra el tipo, nombre, valor inicial y datos del arreglo.
		name := "<sin nombre>"
		if n.Name != nil {
			name = n.Name.Value
		}

		description := "VarDeclaration " + n.Type + " " + name
		if n.Const {
			description = "const " + description
		}

		hijos := []hijo{}

		if n.ArraySize != nil {
			hijos = append(hijos, hijo{"Tamaño", n.ArraySize})
		}

		if n.Value != nil {
			hijos = append(hijos, hijo{"Valor", n.Value})
		}

		for i, element := range n.Elements {
			hijos = append(
				hijos,
				hijo{fmt.Sprintf("Elemento [%d]", i), element},
			)
		}

		if n.IsArray {
			description += " []"
		}

		return description, hijos

	case *ast.VarDeclarationList:
		// Una declaración puede contener varias variables.
		hijos := []hijo{}

		for i, declaration := range n.Declarations {
			hijos = append(
				hijos,
				hijo{fmt.Sprintf("Variable %d", i+1), declaration},
			)
		}

		return "VarDeclarationList", hijos

	case *ast.ReturnStatement:
		if n.ReturnValue == nil {
			return "ReturnStatement", nil
		}
		return "ReturnStatement", []hijo{{"Valor", n.ReturnValue}}

	case *ast.ExpressionStatement:
		return "ExpressionStatement", []hijo{{"", n.Expression}}

	case *ast.BlockStatement:
		return "BlockStatement", sentencias(n.Statements)

	case *ast.CoutStatement:
		// Cada valor de cout se muestra en el orden en que se imprime.
		hijos := []hijo{}

		for i, value := range n.Values {
			hijos = append(
				hijos,
				hijo{fmt.Sprintf("Salida %d", i+1), value},
			)
		}

		return "CoutStatement", hijos

	case *ast.BreakStatement:
		return "BreakStatement", nil

	case *ast.ContinueStatement:
		return "ContinueStatement", nil

	// --- Literales (hojas del árbol: no tienen hijos) ---

	case *ast.Identifier:
		return "Identifier " + n.Value, nil

	case *ast.IntegerLiteral:
		return fmt.Sprintf("IntegerLiteral %d", n.Value), nil

	case *ast.DoubleLiteral:
		return fmt.Sprintf("DoubleLiteral %g", n.Value), nil

	case *ast.Boolean:
		return fmt.Sprintf("Boolean %t", n.Value), nil

	case *ast.StringLiteral:
		return fmt.Sprintf("StringLiteral %q", n.Value), nil // %q agrega comillas

	case *ast.EndlLiteral:
		return "EndlLiteral", nil

	// --- Operadores ---

	case *ast.PrefixExpression:
		return "PrefixExpression " + n.Operator,
			[]hijo{{"Derecha", n.Right}}

	case *ast.PostfixExpression:
		return "PostfixExpression " + n.Operator,
			[]hijo{{"Izquierda", n.Left}}

	case *ast.InfixExpression:
		return "InfixExpression " + n.Operator,
			[]hijo{
				{"Izquierda", n.Left},
				{"Derecha", n.Right},
			}

	case *ast.AssignExpression:
		return "AssignExpression " + n.Operator,
			[]hijo{
				{"Destino", n.Left},
				{"Valor", n.Value},
			}

	// --- Control de flujo ---

	case *ast.IfExpression:
		hijos := []hijo{
			{"Condición", n.Condition},
			{"Entonces", n.Consequence},
		}

		if n.Alternative != nil { // el else es opcional
			hijos = append(hijos, hijo{"Sino", n.Alternative})
		}

		return "IfExpression", hijos

	case *ast.WhileExpression:
		return "WhileExpression",
			[]hijo{
				{"Condición", n.Condition},
				{"Cuerpo", n.Body},
			}

	case *ast.DoWhileStatement:
		// El cuerpo se ejecuta antes de comprobar la condición.
		return "DoWhileStatement",
			[]hijo{
				{"Cuerpo", n.Body},
				{"Condición", n.Condition},
			}

	case *ast.ForExpression:
		// Las tres partes del encabezado son opcionales: solo se muestran
		// las que existen.
		hijos := []hijo{}

		if n.Init != nil {
			hijos = append(hijos, hijo{"Inicio", n.Init})
		}

		if n.Condition != nil {
			hijos = append(hijos, hijo{"Condición", n.Condition})
		}

		if n.Update != nil {
			hijos = append(hijos, hijo{"Actualización", n.Update})
		}

		hijos = append(hijos, hijo{"Cuerpo", n.Body})

		return "ForExpression", hijos

	// --- Funciones ---

	case *ast.FunctionDeclaration:
		// Los parámetros se muestran con sus tipos y nombres.
		params := []string{}

		for _, param := range n.Params {
			name := ""
			if param.Name != nil {
				name = param.Name.Value
			}

			description := strings.TrimSpace(param.Type + " " + name)

			if param.IsArray {
				description += "[]"
			}

			params = append(params, description)
		}

		name := "<sin nombre>"
		if n.Name != nil {
			name = n.Name.Value
		}

		description := "FunctionDeclaration " +
			n.ReturnType + " " +
			name + "(" + strings.Join(params, ", ") + ")"

		if n.Body == nil {
			return description + " [prototipo]", nil
		}

		return description, []hijo{{"Cuerpo", n.Body}}

	case *ast.CallExpression:
		// Primero la función que se llama y luego cada argumento numerado.
		hijos := []hijo{{"Función", n.Function}}

		for i, arg := range n.Arguments {
			hijos = append(
				hijos,
				hijo{fmt.Sprintf("Arg %d", i), arg},
			)
		}

		return "CallExpression", hijos

	// --- Estructuras de datos ---

	case *ast.IndexExpression:
		return "IndexExpression",
			[]hijo{
				{"Objeto", n.Left},
				{"Índice", n.Index},
			}

	case *ast.MemberExpression:
		// Representa expresiones como texto.size().
		member := "<sin miembro>"
		if n.Member != nil {
			member = n.Member.Value
		}

		return "MemberExpression ." + member,
			[]hijo{{"Objeto", n.Object}}

	// --- Casos especiales ---

	case nil:
		// Un hijo que no existe (por ejemplo un return sin valor).
		return "<vacío>", nil

	default:
		// Tipo de nodo que describirNodo todavía no conoce.
		return fmt.Sprintf("%T", nodo), nil
	}
}

// sentencias convierte una lista de sentencias en una lista de hijos sin
// etiqueta. Se usa para Program y BlockStatement, cuyos hijos son
// simplemente sus sentencias en orden.
func sentencias(stmts []ast.Statement) []hijo {
	hijos := []hijo{}

	for _, s := range stmts {
		hijos = append(hijos, hijo{"", s})
	}

	return hijos
}
