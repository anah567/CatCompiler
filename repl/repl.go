// Paquete repl implementa el REPL (Read-Eval-Print Loop) de CatCompiler:
// un ciclo interactivo que
//
//	Read  → lee una línea que escribe el usuario,
//	Eval  → la procesa (por ahora con el lexer o con el parser, todavía no
//	        hay evaluador),
//	Print → muestra el resultado,
//	Loop  → y vuelve a empezar.
//
// El REPL tiene dos MODOS:
//   - lexer:  muestra la lista de tokens de la línea.
//   - parser: muestra el programa reconstruido y el AST en forma de árbol,
//     o los errores de sintaxis si los hay.
//
// El modo se elige al iniciar y se puede cambiar con los comandos :lexer y
// :parser. Los comandos del REPL empiezan con ':' para no confundirse con
// código del lenguaje.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"polaobesahermosa/ast"
	"polaobesahermosa/lexer"
	"polaobesahermosa/parser"
	"polaobesahermosa/token"
)

// Modos de funcionamiento del REPL. Se usa el mismo texto en el prompt
// ("lexer>> " o "parser>> ") para que el usuario sepa en qué modo está.
const (
	modoLexer  = "lexer"
	modoParser = "parser"
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
			switch line {
			case ":lexer":
				modo = modoLexer
				fmt.Fprintln(out, "Modo lexer: se mostrarán los tokens.")
			case ":parser":
				modo = modoParser
				fmt.Fprintln(out, "Modo parser: se mostrará el AST.")
			case ":ayuda":
				imprimirAyuda(out)
			case ":salir":
				fmt.Fprintln(out, "¡Miau! Hasta luego.")
				return
			default:
				fmt.Fprintf(out, "Comando desconocido %q. Escribe :ayuda\n", line)
			}
			continue // se vuelve a mostrar el prompt
		}

		// 4. Si no es comando, es código: se procesa según el modo actual.
		if modo == modoLexer {
			probarLexer(out, line)
		} else {
			probarParser(out, line)
		}
	}
}

// elegirModo muestra un menú para que el usuario elija si quiere probar
// el lexer o el parser. Acepta "1"/"lexer" o "2"/"parser" (sin importar
// mayúsculas). Si la respuesta no es válida, vuelve a preguntar.
//
// Devuelve el modo elegido y true, o "" y false si la entrada se terminó
// antes de elegir (en ese caso el REPL debe cerrarse).
func elegirModo(scanner *bufio.Scanner, out io.Writer) (string, bool) {
	for {
		fmt.Fprintln(out, "\n¿Qué quieres probar?")
		fmt.Fprintln(out, "  1) Lexer  (ver los tokens)")
		fmt.Fprintln(out, "  2) Parser (ver el AST)")
		fmt.Fprint(out, "Opción: ")

		if !scanner.Scan() {
			return "", false
		}

		switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
		case "1", "lexer":
			return modoLexer, true
		case "2", "parser":
			return modoParser, true
		default:
			fmt.Fprintln(out, "Opción no válida, escribe 1 o 2.")
		}
	}
}

// imprimirAyuda muestra la lista de comandos disponibles en el REPL.
func imprimirAyuda(out io.Writer) {
	fmt.Fprintln(out, "\nComandos:")
	fmt.Fprintln(out, "  :lexer   cambiar a modo lexer")
	fmt.Fprintln(out, "  :parser  cambiar a modo parser")
	fmt.Fprintln(out, "  :ayuda   mostrar esta ayuda")
	fmt.Fprintln(out, "  :salir   salir del REPL")
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
//   - prefijo:  texto que va al inicio de cada renglón de los HIJOS de
//     este nodo. Se construye con "│   " (si el ancestro todavía tiene
//     hermanos debajo, hay que seguir dibujando su línea vertical) o con
//     "    " (si el ancestro era el último, ya no hay línea que dibujar).
//
// Ejemplo de salida para "let x = 1 + 2":
//
//	Program
//	└── LetStatement x
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

		// rama:    lo que se dibuja justo antes del hijo.
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

	// --- Sentencias ---
	case *ast.LetStatement:
		return "LetStatement " + n.Name.Value, []hijo{{"Valor", n.Value}}
	case *ast.ReturnStatement:
		return "ReturnStatement", []hijo{{"Valor", n.ReturnValue}}
	case *ast.ExpressionStatement:
		return "ExpressionStatement", []hijo{{"", n.Expression}}
	case *ast.BlockStatement:
		return "BlockStatement", sentencias(n.Statements)

	// --- Literales (hojas del árbol: no tienen hijos) ---
	case *ast.Identifier:
		return "Identifier " + n.Value, nil
	case *ast.IntegerLiteral:
		return fmt.Sprintf("IntegerLiteral %d", n.Value), nil
	case *ast.Boolean:
		return fmt.Sprintf("Boolean %t", n.Value), nil
	case *ast.StringLiteral:
		return fmt.Sprintf("StringLiteral %q", n.Value), nil // %q agrega comillas

	// --- Operadores ---
	case *ast.PrefixExpression:
		return "PrefixExpression " + n.Operator, []hijo{{"Derecha", n.Right}}
	case *ast.InfixExpression:
		return "InfixExpression " + n.Operator,
			[]hijo{{"Izquierda", n.Left}, {"Derecha", n.Right}}
	case *ast.AssignExpression:
		return "AssignExpression =",
			[]hijo{{"Destino", n.Left}, {"Valor", n.Value}}

	// --- Control de flujo ---
	case *ast.IfExpression:
		hijos := []hijo{{"Condición", n.Condition}, {"Entonces", n.Consequence}}
		if n.Alternative != nil { // el else es opcional
			hijos = append(hijos, hijo{"Sino", n.Alternative})
		}
		return "IfExpression", hijos
	case *ast.WhileExpression:
		return "WhileExpression",
			[]hijo{{"Condición", n.Condition}, {"Cuerpo", n.Body}}
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
	case *ast.FunctionLiteral:
		// Los parámetros se muestran en la misma línea: FunctionLiteral (a, b)
		params := []string{}
		for _, p := range n.Parameters {
			params = append(params, p.Value)
		}
		return "FunctionLiteral (" + strings.Join(params, ", ") + ")",
			[]hijo{{"Cuerpo", n.Body}}
	case *ast.CallExpression:
		// Primero la función que se llama y luego cada argumento numerado.
		hijos := []hijo{{"Función", n.Function}}
		for i, arg := range n.Arguments {
			hijos = append(hijos, hijo{fmt.Sprintf("Arg %d", i), arg})
		}
		return "CallExpression", hijos

	// --- Estructuras de datos ---
	case *ast.ArrayLiteral:
		// Cada elemento se etiqueta con su posición: [0], [1], ...
		hijos := []hijo{}
		for i, el := range n.Elements {
			hijos = append(hijos, hijo{fmt.Sprintf("[%d]", i), el})
		}
		return "ArrayLiteral", hijos
	case *ast.IndexExpression:
		return "IndexExpression",
			[]hijo{{"Objeto", n.Left}, {"Índice", n.Index}}
	case *ast.HashLiteral:
		// Cada pareja se muestra como dos hijos seguidos: Clave y Valor.
		hijos := []hijo{}
		for _, pair := range n.Pairs {
			hijos = append(hijos, hijo{"Clave", pair.Key}, hijo{"Valor", pair.Value})
		}
		return "HashLiteral", hijos

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
