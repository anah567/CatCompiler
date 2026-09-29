# CatCompiler

CatCompiler es un proyecto educativo en Go para analizar un lenguaje pequeño. Incluye un analizador léxico (*lexer*), un analizador sintáctico (*parser*) y un REPL que permite ver los tokens o el árbol de sintaxis abstracta (AST) de una línea de código. Por ahora no evalúa ni ejecuta las expresiones.

## Requisitos

- Go 1.26.3 o una versión compatible con la indicada en `go.mod`.
- No requiere dependencias externas.

## Inicio rápido

Desde la raíz del proyecto:

```bash
go run .
```

Al iniciar, elige `1` para ver los tokens o `2` para ver el AST. Luego escribe una línea de código y presiona Enter. El REPL procesa cada línea por separado.

Por ejemplo, en modo parser:

```text
Opción: 2
parser>> let x = 1 + 2 * 3;
Programa:
  let x = (1 + (2 * 3));
AST:
Program
└── LetStatement x
    └── Valor: InfixExpression +
        ├── Izquierda: IntegerLiteral 1
        └── Derecha: InfixExpression *
            ├── Izquierda: IntegerLiteral 2
            └── Derecha: IntegerLiteral 3
```

En modo lexer, la misma línea muestra cada token con su tipo y literal. Los errores de sintaxis se muestran en modo parser.

### Comandos del REPL

| Comando | Acción |
| --- | --- |
| `:lexer` | Cambia al modo de tokens. |
| `:parser` | Cambia al modo de AST. |
| `:ayuda` | Muestra los comandos disponibles. |
| `:salir` | Cierra el REPL. |

También puedes salir con Ctrl+D.

## Sintaxis reconocida

- Declaraciones `let`, `return` y expresiones.
- Enteros, cadenas entre comillas dobles, booleanos, identificadores, arreglos y hashes.
- Operadores aritméticos (`+`, `-`, `*`, `/`, `%`, `**`), de comparación (`==`, `!=`, `<`, `>`, `<=`, `>=`), asignación (`=`) y prefijos (`-`, `!`).
- Condicionales `if`, `elif` y `else`; ciclos `while` y `for`.
- Funciones con `function`, llamadas e indexación con `[]`.

El lexer reconoce identificadores con letras ASCII y guion bajo; los dígitos se usan para números enteros. No hay evaluador ni soporte para ejecutar programas completos desde un archivo.

## Estructura

| Ruta | Función |
| --- | --- |
| `main.go` | Inicia la aplicación y el REPL. |
| `token/` | Define los tipos de token y las palabras reservadas. |
| `lexer/` | Convierte texto fuente en tokens. |
| `parser/` | Construye el AST mediante un parser de Pratt. |
| `ast/` | Define los nodos del árbol sintáctico. |
| `repl/` | Gestiona la entrada interactiva y muestra tokens o AST. |

## Desarrollo

Para compilar y revisar los paquetes:

```bash
go build ./...
go test ./...
```

Actualmente el repositorio no contiene archivos de prueba; `go test ./...` comprueba que los paquetes compilen.
