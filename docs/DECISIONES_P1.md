# Decisiones de diseño — Persona 1 (sintaxis de C++)

Alcance: `token/`, `lexer/`, `parser/`, `ast/`, sus pruebas y estos documentos.
Rama: `persona1/sintaxis-cpp` (sale de `evaluator`).
Contrato de nodos para el grupo: [CONTRATO_AST.md](CONTRATO_AST.md).
Gramática: [gramatica_cpp.ebnf](gramatica_cpp.ebnf).

Cómo verificar mi parte:

```bash
gofmt -l token lexer parser ast        # no debe imprimir nada
go vet  ./token ./lexer ./parser ./ast
go test -race ./token ./lexer ./parser ./ast
go test ./parser -run '^$' -fuzz=FuzzParser -fuzztime=30s   # opcional
```

## 1. Qué se eliminó del lenguaje base y por qué

| Eliminado | Por qué |
|---|---|
| `let x = 5;` (`LET`, `LetStatement`) | En C++ las variables se declaran con su tipo: `int x = 5;` → `VarDeclaration`. |
| `function(a){}` (`FUNCTION`, `FunctionLiteral`) | C++ no tiene funciones anónimas como valores (sin lambdas en el subconjunto). Las funciones tienen nombre y tipos: `FunctionDeclaration`. |
| `elif` | En C++ se escribe `else if`. Sale "gratis": el cuerpo del `else` puede ser otra sentencia `if`. |
| `**` (potencia) | No existe en C++ (se usa `pow`). |
| Hashes `{"a": 1}` (`HashLiteral`, token `:`) | No existen en C++. Además, `{` al inicio de una expresión ya no puede ser un hash: ahora `{` en una sentencia es un bloque. |
| Arreglos literales `[1, 2]` (`ArrayLiteral`) | En C++ un arreglo se declara: `int a[3] = {1, 2, 3};` → campo `Elements` de `VarDeclaration`. |
| `;` opcional | En C++ es obligatorio. |
| Condición sin paréntesis (`if x > 1 {`) | En C++ los paréntesis son obligatorios. |

## 2. Decisiones tomadas (la opción más simple, ver detalles en el código)

**Tokens / lexer**

1. `string`, `cout`, `endl` y `std` son **palabras reservadas** del lenguaje, aunque en C++
   real son nombres de la biblioteca estándar. Simplifica el parser (`cout` inicia una
   sentencia, `string` es un tipo). Consecuencia: no se puede llamar `string` a una variable.
2. El token `ILLEGAL` lleva en `Literal` el **mensaje** del error léxico (por ejemplo
   `cadena sin cerrar: falta '"'`), no el texto original. Así no hizo falta agregar otro
   campo a `token.Token`.
3. El lexer no se detiene en un error léxico: entrega el `ILLEGAL` y sigue con el resto.
4. Cualquier línea que empiece con `#` se descarta entera (no hay preprocesador). `#define`
   se aceptaría sin efecto.
5. Los identificadores pueden tener dígitos después de la primera letra (`suma2`); el lexer
   base solo aceptaba letras.
6. Un número `3.5` es `DOUBLE` solo si hay dígitos a ambos lados del punto. `3.` y `.5` no
   son doubles; `1e5` no se soporta. `3.4.5` es error léxico (número mal formado).
7. Los enteros se leen en base 10 (`010` vale 10; en C++ sería octal).
8. Una cadena no puede tener un salto de línea real adentro (igual que C++): se reporta
   "cadena sin cerrar" en esa línea y el lexer sigue en la línea siguiente.
9. Escapes soportados: `\n`, `\t`, `\"`, `\\`. Cualquier otro (`\q`) es error léxico. El
   `Literal` del token `STRING` ya trae los escapes convertidos.
10. `&` o `|` solos son error léxico (no hay referencias ni operadores de bits).
11. Columnas: cada byte cuenta 1 (una tabulación cuenta 1 columna; una letra con tilde dentro
    de una cadena cuenta 2 porque ocupa 2 bytes en UTF-8).

**Parser / AST**

12. Posición de un error sintáctico = posición del token **encontrado** (no del final del
    token anterior). Ej.: en `int x = 5` + salto de línea + `}` el error de `;` sale en la
    línea de la `}`.
13. Los errores léxicos se reportan en `nextToken` como `Error léxico [...]` y el token se
    **descarta**; el parser sigue como si el carácter no estuviera (así `int x = 5 @;` da un
    solo error).
14. Recuperación de errores: `synchronize` salta hasta `;`, hasta la `}` que cierra un
    bloque abierto en la zona saltada, o hasta antes de la `}` del bloque de afuera. Evita
    errores en cascada; a cambio, puede saltarse una sentencia buena pegada a la mala.
15. Se mantuvo `Errors() []string`. **No** se agregó `ErrorList()`: los textos ya traen tipo,
    línea y columna.
16. `IfExpression`, `WhileExpression` y `ForExpression` **conservan el nombre** del proyecto
    base (para no romper más código de los compañeros) pero ahora son **sentencias**
    (implementan `Statement`) y van directo en la lista de sentencias.
17. Cuerpo de `if/else/while/for/do`: bloque **o una sola sentencia**; la sentencia sola se
    envuelve en un `BlockStatement`. El evaluador siempre recibe un bloque.
18. `else if` = `else` cuyo cuerpo es una sentencia `if`. El `else` se une al `if` más
    cercano (problema del "else colgante", igual que C++).
19. `endl` es un nodo propio, `EndlLiteral` (no un `Identifier`).
20. `int a = 1, b = 2;` produce un `VarDeclarationList`; con una sola variable se produce
    un `VarDeclaration` (nunca una lista de 1).
21. No hay nodo `ArrayInitializer`: la lista `{1, 2, 3}` va en `VarDeclaration.Elements`. Se
    permite `int a[] = {1, 2};` (tamaño = cantidad de valores, lo decide el evaluador).
22. `using namespace std;` se verifica pero **no genera nodo**. `std::cout` no se soporta (no
    hay token `::`).
23. Las funciones solo se declaran en el nivel superior (`blockDepth`). Prototipos
    `int f(int);` → `Body == nil`. En una definición todos los parámetros deben tener nombre.
    `f(void)` no se soporta (un parámetro no puede ser `void`).
24. `cout << valor`: cada valor se parsea con precedencia `SHIFT`, así se detiene en el
    siguiente `<<`. `cout << a < b;` es error (como en C++ hay que escribir `(a < b)`).
25. `<<` y `>>` también existen como operadores infijos normales (`1 << 2`). Si el evaluador
    no quiere soportar desplazamientos, puede dar error de ejecución.
26. `++`/`--` y la asignación exigen como operando un `Identifier` o un `IndexExpression`
    (lo revisa el parser: `5 = x;` y `++5;` son errores sintácticos).
27. `String()` de los nodos cambió para parecerse a C++: `ExpressionStatement` termina en `;`,
    la asignación va entre paréntesis `(a = (b = 3))`, el `if` imprime `if (cond)`, y
    `Program` separa sentencias con un espacio.

**Lo que el parser NO revisa (lo debe hacer el evaluador)**

- Tipos: `int x = "hola";` y `bool b = 3.5;` parsean sin error.
- Variables o funciones no declaradas; variables repetidas en el mismo ámbito.
- `break`/`continue` fuera de un ciclo; `return` fuera de una función o con/sin valor
  según el tipo; asignar a una `const`.
- Que exista `main`.

## 3. Limitaciones conocidas

- No hay `cin`, `char` (ni `'a'`), `vector<int>`, `switch`, operador ternario `?:`,
  operador coma (`i++, j--`), conversiones `(int)x`, `+x` unario, `struct`/`class`,
  punteros ni referencias.
- `vector<int>` se dejó fuera a propósito: `>>` en `vector<vector<int>>` y `<`/`>` como
  delimitadores de plantilla complican el lexer/parser.
- Solo letras ASCII en identificadores.
- Un `for (int f() {}; ...)` no se rechaza de forma especial (caso muy raro).

## 4. Lo que Persona 2 y Persona 3 deben adaptar

Al día de hoy `go build ./...` falla en `evaluator` (esperado):

```text
evaluator/evaluator.go:29:12: undefined: ast.LetStatement
evaluator/evaluator.go:87:12: undefined: ast.FunctionLiteral
evaluator/evaluator.go:100:12: undefined: ast.ArrayLiteral
evaluator/evaluator.go:116:12: undefined: ast.HashLiteral
evaluator/evaluator.go:394:22: undefined: ast.HashLiteral
```

`repl` no compila mientras `evaluator` no compile, y además `repl/repl.go`
(`describirNodo`) usa esos mismos cuatro nodos.

**Persona 2 (evaluador / objects)**
- Quitar los `case` de los 4 nodos eliminados y `evalHash`.
- Agregar `case` para: `VarDeclaration`, `VarDeclarationList`, `FunctionDeclaration`,
  `CoutStatement`, `DoubleLiteral`, `EndlLiteral`, `PostfixExpression`,
  `DoWhileStatement`, `BreakStatement`, `ContinueStatement`, `MemberExpression` (opcional).
- `AssignExpression.Operator` (`+=`, ...), `PrefixExpression` con `++`/`--`, `&&`/`||`.
- `ReturnStatement.ReturnValue` puede ser `nil` (hoy `Eval(nil)` da "nodo incompleto").
- `objects.Function` usa `[]*ast.Identifier`; ahora los parámetros son `[]ast.Param`.
- Nuevo objeto para `double`; valores por defecto de `int x;`; llamar a `main`.
- Para errores de ejecución con posición: `n.Token.Line`, `n.Token.Column`.

**Persona 3 (REPL / archivos / ejemplos / pruebas)**
- `describirNodo`: casos nuevos y eliminados (tabla en CONTRATO_AST.md §6).
- `repl/file.go` exige extensión `.ssg`; debería aceptar `.cpp`.
- `ejemplos/programa.ssg` y el README usan la sintaxis vieja (`let`, `function`, hashes).
- Los `;` son obligatorios: en el REPL `5 + 5` sin `;` es error sintáctico.
- El modo lexer puede mostrar `tok.Line` y `tok.Column`.
- El programa de referencia está en `parser/testdata/programa.cpp` (se puede copiar a
  `examples/`).

## 5. Para qué usé IA y cómo verifiqué

Usé un asistente de IA (Claude Code, modelo Claude Opus) en una sesión de trabajo para
escribir el código de mis cuatro paquetes, sus pruebas y estos documentos, a partir de
instrucciones detalladas que yo le di (alcance, contrato propuesto, formato de errores,
programa de referencia). Lo que pasó en esa sesión, sin adornos:

- **Arranque.** El repositorio no estaba en el equipo; se clonó de la URL del enunciado
  (`sebastianzapatar/compi8am`, rama `evaluator`). Git no tenía nombre ni correo
  configurados, así que se configuraron solo para este repositorio.
- **Lexer: una prueba falló** en la primera ejecución:
  `cadena_sin_cerrar_al_final: se esperaba {EOF 1 6}, se obtuvo {EOF 1 7}`. La causa: al
  llegar al final de la entrada, `readChar` seguía sumando columnas cada vez que se llamaba.
  Se corrigió haciendo que `readChar` no haga nada si ya pasó del final, y la prueba pasó.
- **Parser:** las pruebas de las tres etapas (expresiones/declaraciones, control de flujo,
  opcionales) pasaron en la primera ejecución. Ojo: eso **no** prueba que todo esté bien,
  porque las pruebas y el código los escribió la misma herramienta; si hubiera un error de
  concepto podría estar en los dos. Por eso las posiciones esperadas (línea/columna) se
  contaron carácter por carácter, y los ejemplos de la guía de estudio se sacaron de la
  salida real del lexer y el parser (un programa temporal que luego se borró), no escritos
  a mano.
- **`gofmt`** pidió cambios tres veces: alinear comentarios en `New`, el orden de los
  `import` del archivo de pruebas, y reacomodó mal el comentario de `parseForStatement` (una
  lista justo después de un bloque de código); ese comentario se reescribió.
- Al revisar el código se encontró un comentario con un valor incorrecto
  (`POSTFIX = 12`; en realidad es 11) y se corrigió.
- **Robustez:** se agregó `FuzzParser` y se ejecutó 60 s (≈17 millones de entradas
  aleatorias): sin pánicos ni ciclos infinitos. También hay una prueba con 27 entradas rotas
  que deben dar al menos un error.
- **Lo que no se hizo:** no se hizo `push` porque el único remoto (`origin`) es el
  repositorio del docente; hay que agregar el remoto del grupo. No se tocaron `evaluator/`,
  `objects/`, `repl/`, `main.go` ni los ejemplos.

Cómo lo verifiqué yo:

- [COMPLETAR: qué archivos/funciones revisé línea por línea y qué entendí de cada una]
- [COMPLETAR: entradas que probé a mano en el REPL (cuando P3 lo adapte) o en las pruebas]
- [COMPLETAR: cambios que hice yo mismo después de revisar, si los hubo]
- [COMPLETAR: decisiones de la sección 2 que confirmé con Persona 2 y Persona 3]
