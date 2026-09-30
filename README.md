# CatCompiler

CatCompiler es un proyecto educativo en Go para analizar y ejecutar un lenguaje pequeño. Incluye un analizador léxico (*lexer*), un analizador sintáctico (*parser*), objetos de ejecución y un evaluador. El REPL permite ver tokens, ver el árbol de sintaxis abstracta (AST) o ejecutar código.

## Requisitos

- Go 1.26.3 o una versión compatible con la indicada en `go.mod`.
- No requiere dependencias externas.

## Inicio rápido

Desde la raíz del proyecto:

```bash
go run .
```

Al iniciar, elige `1` para ver los tokens, `2` para ver el AST o `3` para ejecutar código. Luego escribe una línea y presiona Enter. En modo evaluador, las variables persisten entre líneas de la misma sesión.

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

En modo evaluador:

```text
Opción: 3
evaluator>> let doble = function(x) { x * 2 };
null
evaluator>> doble(7)
14
evaluator>> let a = [1, 2]; a[0] = 9; a[0]
9
```

`null` indica que una declaración no produce un resultado. Los índices de lectura inexistentes también devuelven `null`; escribir fuera de un arreglo produce un error.

### Ejemplos para copiar y pegar en el REPL

Ejecuta `go run .`, elige `3` y pega **una línea por vez**. El REPL conserva las variables de las líneas anteriores:

```text
let base = 10;
base + 5
let doble = function(x) { return x * 2; };
doble(base)
let valores = [3, 6, 9];
valores[1]
valores[1] = 8;
valores[1]
if (base > 5) { "mayor" } else { "menor" }
let total = 0; for (let i = 0; i < 3; i = i + 1) { total = total + valores[i]; } total
```

En orden, las expresiones muestran `15`, `20`, `6`, `8`, `8`, `mayor` y `20`. Las líneas que solo declaran variables muestran `null`. Para empezar de cero, cierra el REPL y ejecútalo de nuevo.

### Ejecutar un archivo `.ssg`

El único archivo de ejemplos, [ejemplos/programa.ssg](ejemplos/programa.ssg), reúne el programa básico y las funciones recursivas de Ackermann, Fibonacci y Torres de Hanói. Desde la raíz del proyecto puedes ejecutarlo directamente:

```bash
go run . ejemplos/programa.ssg
```

La salida es un hash con los cuatro resultados: `resultado: 24`, `ackermann: 7`, `fibonacci: 21` y `hanoi` con siete movimientos. También puedes cargarlo durante una sesión del REPL, desde cualquier modo:

```text
:cargar ejemplos/programa.ssg
```

Después de cargarlo, cambia a `:evaluator` si hace falta y prueba `resultado`, `ackermann(2, 2)`, `fibonacci(8)` o `hanoi(3, "A", "C", "B")`. Todas las funciones quedan disponibles en la sesión. La ruta se interpreta desde el directorio donde ejecutaste `go run .`. Los archivos `.ssg` admiten bloques y funciones repartidos en varias líneas; el REPL interactivo procesa cada línea por separado. El lenguaje todavía no admite comentarios dentro de `.ssg`.

### Funciones recursivas

Estas funciones se llaman a sí mismas. Ya están juntas en `ejemplos/programa.ssg`. Si prefieres escribirlas directamente en el REPL, elige el modo `3` y pega **cada definición completa en una sola línea**; luego pega su llamada.

**Ackermann** (a veces escrita «Ackerman»; `ackermann(2, 2)` produce `7`):

```text
let ackermann = function(m, n) { if (m == 0) { return n + 1; } if (n == 0) { return ackermann(m - 1, 1); } return ackermann(m - 1, ackermann(m, n - 1)); };
ackermann(2, 2)
```

**Fibonacci** (`fibonacci(8)` produce `21`):

```text
let fibonacci = function(n) { if (n <= 1) { return n; } return fibonacci(n - 1) + fibonacci(n - 2); };
fibonacci(8)
```

**Torres de Hanói** (`hanoi(3, "A", "C", "B")` devuelve los siete movimientos necesarios para llevar los discos de A a C, usando B como auxiliar):

```text
let hanoi = function(n, origen, destino, auxiliar) { if (n == 0) { return []; } return hanoi(n - 1, origen, auxiliar, destino) + [[origen, destino]] + hanoi(n - 1, auxiliar, destino, origen); };
hanoi(3, "A", "C", "B")
```

Cada par del resultado es `[origen, destino]`; el primero es `[A, C]` y el último también. Aquí `+` une dos arreglos y crea uno nuevo, por lo que se pueden combinar los movimientos de cada llamada recursiva.

Usa entradas pequeñas con Ackermann: el número de llamadas crece muy rápido.

### Comandos del REPL

| Comando | Acción |
| --- | --- |
| `:lexer` | Cambia al modo de tokens. |
| `:parser` | Cambia al modo de AST. |
| `:evaluator` | Cambia al modo de ejecución. |
| `:cargar ruta.ssg` | Lee y ejecuta un archivo en el entorno actual. |
| `:ayuda` | Muestra los comandos disponibles. |
| `:salir` | Cierra el REPL. |

También puedes salir con Ctrl+D.

## Sintaxis reconocida

- Declaraciones `let`, `return` y expresiones.
- Enteros, cadenas entre comillas dobles, booleanos, identificadores, arreglos y hashes.
- Operadores aritméticos (`+`, `-`, `*`, `/`, `%`, `**`), de comparación (`==`, `!=`, `<`, `>`, `<=`, `>=`), asignación (`=`) y prefijos (`-`, `!`). `+` también concatena cadenas o arreglos del mismo tipo.
- Condicionales `if`, `elif` y `else`; ciclos `while` y `for`.
- Funciones con `function`, llamadas e indexación con `[]`.

El evaluador soporta expresiones, variables, asignación, condicionales, ciclos, funciones con cierres, arreglos y hashes. Solo `false` y `null` se consideran falsos. Los hashes admiten claves enteras, booleanas o de texto. El lexer reconoce identificadores con letras ASCII y guion bajo; los dígitos se usan para números enteros.

## Estructura

| Ruta | Función |
| --- | --- |
| `main.go` | Inicia la aplicación y el REPL. |
| `token/` | Define los tipos de token y las palabras reservadas. |
| `lexer/` | Convierte texto fuente en tokens. |
| `parser/` | Construye el AST mediante un parser de Pratt. |
| `ast/` | Define los nodos del árbol sintáctico. |
| `objects/objects.go` | Define los valores durante la ejecución y los entornos de variables. |
| `evaluator/evaluator.go` | Recorre el AST y calcula sus resultados. |
| `repl/` | Gestiona la entrada interactiva y muestra tokens, AST o resultados. |
| `ejemplos/programa.ssg` | Todos los ejemplos en un único programa ejecutable. |

## Desarrollo

Para compilar y revisar los paquetes:

```bash
go build ./...
go test ./...
```

`go test ./...` comprueba las operaciones, el control de flujo, las funciones, las colecciones y los errores del evaluador.
