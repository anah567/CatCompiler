# Guía de estudio — Persona 1 (lexer, parser, AST)

Guía para el examen escrito. Está pensada para alguien que nunca vio compiladores:
primero la idea general, luego cada etapa con ejemplos, y al final ejercicios tipo examen.
Todos los tokens, árboles y mensajes de error de esta guía se sacaron de la **salida real**
del lexer y el parser del proyecto.

---

## 1. La idea general en una imagen

```
 texto            tokens                 AST                        resultado
"int x = 2+3;" ─▶ INT_TYPE IDENT = ...  ─▶ VarDeclaration x       ─▶  x vale 5
                  (lexer)                  └── (+) 2 3              (evaluador)
                                           (parser)
```

| Etapa | Archivo | Entrada → salida | Qué errores detecta |
|---|---|---|---|
| Lexer (análisis léxico) | `lexer/lexer.go` | texto → tokens | **Léxicos**: caracteres que no forman ningún token (`@`, cadena sin cerrar, `3.4.5`...) |
| Parser (análisis sintáctico) | `parser/parser.go` | tokens → AST | **Sintácticos**: tokens en un orden que la gramática no permite (falta `;`, falta `)`...) |
| Evaluador (Persona 2) | `evaluator/` | AST → resultado | **De ejecución**: división por cero, variable no declarada, índice fuera de rango... |

Analogía con el español: el lexer separa las **palabras** ("el", "perro", "come");
el parser revisa la **gramática** ("perro el come" está mal ordenado); el evaluador
entiende el **significado** ("el perro come piedras" es gramatical pero raro).

---

## 2. El lexer: cómo se parte un texto en tokens

Un **token** es una "palabra" del lenguaje con 4 datos (`token/token.go`):

```go
type Token struct {
    Type    TokenType // la clase: INT_TYPE, IDENT, INT, "+", ";" ...
    Literal string    // el texto: "int", "edad", "20", "+", ";"
    Line    int       // línea donde empieza (desde 1)
    Column  int       // columna donde empieza (desde 1)
}
```

### 2.1 Cómo avanza el lexer

El lexer guarda: `input` (el texto), `position` (índice del carácter actual), `ch` (el
carácter actual), `readPosition` (el siguiente), `line` y `column`.

- `readChar()` avanza un carácter. Si el carácter que deja atrás es `'\n'`, suma una línea
  y pone la columna en 1; si no, suma una columna. Al llegar al final `ch = 0` (fin).
- `peekChar()` mira el siguiente carácter **sin avanzar**. Sirve para decidir entre `=` y
  `==`, `+`, `++` y `+=`, `<`, `<=` y `<<`, etc.
- `NextToken()` hace siempre lo mismo:
  1. salta espacios (`skipWhitespace`);
  2. **anota la línea y columna** donde empieza el token;
  3. mira `ch` en un `switch` y decide qué token es;
  4. avanza para dejar listo el siguiente.

### 2.2 Ejemplo paso a paso: `int edad = 20;`

| `ch` al empezar | Qué hace `NextToken` | Token |
|---|---|---|
| `i` (col 1) | es letra → `readIdentifier` lee `int` → `LookupIdent("int")` está en `keywords` | `INT_TYPE "int" 1:1` |
| espacio → `e` (col 5) | letra → lee `edad` → no es reservada | `IDENT "edad" 1:5` |
| espacio → `=` (col 10) | `peekChar()` es espacio, no `=` → es asignación | `= "=" 1:10` |
| espacio → `2` (col 12) | dígito → `readNumber` lee `20`; no sigue `.` → entero | `INT "20" 1:12` |
| `;` (col 14) | delimitador | `; ";" 1:14` |
| fin (col 15) | `ch == 0` | `EOF "" 1:15` |

Reglas importantes:
- **Palabras reservadas**: se lee la palabra completa y se busca en el mapa `keywords`
  (`int`, `double`, `bool`, `string`, `void`, `true`, `false`, `if`, `else`, `while`, `do`,
  `for`, `return`, `break`, `continue`, `const`, `cout`, `endl`, `using`, `namespace`,
  `std`). Si no está, es `IDENT`. Por eso `integer` es `IDENT` aunque empiece con `int`.
- **El operador más largo gana**: `a+++b` se parte `a` `++` `+` `b`.
- **Números**: `3.5` es `DOUBLE` solo si hay dígito después del punto; `3.x` es `INT 3`,
  `.`, `IDENT x`. `3.4.5` es error léxico.
- **Cadenas**: el `Literal` es el contenido **sin comillas y con los escapes convertidos**:
  `"a\n"` → `a` + salto de línea real. Escapes válidos: `\n \t \" \\`.
- **Lo que se ignora**: espacios, `// comentario`, `/* comentario */` y las líneas `#...`
  (`#include <iostream>`). No producen tokens; `NextToken` se llama a sí misma para
  devolver el token que sigue.

### 2.3 Errores léxicos

El lexer no se detiene: devuelve un token `ILLEGAL` cuyo `Literal` es el mensaje.

| Entrada | Token ILLEGAL |
|---|---|
| `x @ y` | `carácter no válido '@'` en 1:3 |
| `"hola` | `cadena sin cerrar: falta '"'` en 1:1 |
| `"a\qb"` | `secuencia de escape no válida '\q'` en 1:1 |
| `/* abierto` | `comentario /* sin cerrar` en 1:1 |
| `3.4.5` | `número mal formado '3.4.5'` en 1:1 |
| `a & b` | `carácter no válido '&' (¿quisiste escribir '&&'?)` en 1:3 |

El **parser** convierte cada `ILLEGAL` en `Error léxico [línea L, columna C]: mensaje`
(en `nextToken`) y lo descarta, para seguir revisando la sintaxis.

---

## 3. El parser: de tokens a árbol

### 3.1 Qué es el AST

Un árbol donde cada nodo es una construcción (declaración, suma, if...) y sus hijos son
sus partes. Es "abstracto" porque pierde lo que no importa: `;`, paréntesis, espacios.
`(1 + 2)` y `1 + 2` dan el mismo árbol.

Dos familias (`ast/ast.go`):
- **Statement** (sentencia, no produce valor): `VarDeclaration`, `FunctionDeclaration`,
  `ReturnStatement`, `IfExpression`, `WhileExpression`, `ForExpression`, `CoutStatement`...
- **Expression** (produce valor): `Identifier`, `IntegerLiteral`, `InfixExpression`,
  `CallExpression`, `PostfixExpression`...

### 3.2 curToken y peekToken

El parser mira **dos** tokens: el actual (`curToken`) y el siguiente (`peekToken`).
- `nextToken()` avanza uno.
- `expectPeek(t)`: "el siguiente TIENE que ser `t`". Si lo es, avanza; si no, registra
  `se esperaba t pero se encontró ...` y devuelve `false`.
- Convención: cada función `parseX` empieza con `curToken` en el **primer** token de su
  construcción y termina con `curToken` en el **último** (el `;` o la `}`).

### 3.3 Sentencias: descenso recursivo

`parseStatement` mira el primer token y decide:

| Primer token | Función |
|---|---|
| tipo (`int`...) o `const` | `parseDeclaration` → variable(s) o función |
| `return` | `parseReturnStatement` |
| `if` / `while` / `for` / `do` | `parseIfStatement` / `parseWhileStatement` / `parseForStatement` / `parseDoWhileStatement` |
| `cout` | `parseCoutStatement` |
| `break` / `continue` | `parseJumpStatement` |
| `{` | `parseBlockStatement` |
| `using` | `parseUsingDirective` (no genera nodo) |
| `;` | sentencia vacía (nada) |
| otro | `parseExpressionStatement` (expresión + `;`) |

¿Variable o función? Las dos empiezan `int nombre`. `parseDeclaration` lee el tipo y el
nombre y mira `peekToken`: si es `(` es función; si no, variable.

### 3.4 Expresiones: el parser de Pratt, paso a paso

Cada token puede tener una función **prefija** (si empieza una expresión: `5`, `x`, `-`,
`(`, `++`) y una **infija** (si va después de una expresión: `+`, `*`, `=`, `(` de llamada,
`[`, `++` postfijo). Cada operador infijo tiene una **precedencia**:

| Nivel (número) | Operadores | Asociatividad |
|---|---|---|
| LOWEST (1) | — | — |
| ASSIGN (2) | `= += -= *= /= %=` | derecha |
| OR (3) | `\|\|` | izquierda |
| AND (4) | `&&` | izquierda |
| EQUALS (5) | `== !=` | izquierda |
| LESSGREATER (6) | `< > <= >=` | izquierda |
| SHIFT (7) | `<< >>` | izquierda |
| SUM (8) | `+ -` | izquierda |
| PRODUCT (9) | `* / %` | izquierda |
| PREFIX (10) | `-x !x ++x --x` | derecha |
| POSTFIX (11) | `x++ x-- f() a[i] s.m` | izquierda |

`parseExpression(prec)`: 1) usa la función prefija del token actual para armar la
izquierda; 2) **mientras** el siguiente operador tenga precedencia **mayor** que `prec`,
avanza y llama a su función infija con la izquierda; 3) devuelve.

**Traza de `x = 1 + 2 * 3;`**

```
parseExpression(LOWEST=1)            cur = x
  prefija IDENT → izquierda = x
  peek '=' (2) > 1 → infija '=' : parseAssignExpression(x)
     ¿x es asignable? sí (Identifier)
     valor = parseExpression(ASSIGN-1 = 1)        cur = 1
        prefija INT → izquierda = 1
        peek '+' (8) > 1 → infija '+': derecha = parseExpression(8)   cur = 2
            prefija INT → izquierda = 2
            peek '*' (9) > 8 → infija '*': derecha = parseExpression(9)   cur = 3
                prefija INT → 3 ; peek ';' (1) > 9? no → devuelve 3
            izquierda = (2 * 3) ; peek ';' (1) > 8? no → devuelve (2 * 3)
        izquierda = (1 + (2 * 3)) ; peek ';' → para
     → (x = (1 + (2 * 3)))
  peek ';' → para
parseExpressionStatement: expectPeek(';') ✔
```

**¿Por qué izquierda en `a - b - c`?** El primer `-` parsea su derecha con su misma
precedencia (8). Al ver el segundo `-` (8), `8 > 8` es falso: no lo absorbe. Queda
`((a - b) - c)`.

**¿Por qué derecha en `a = b = 3`?** `parseAssignExpression` parsea el valor con
`ASSIGN - 1`. El segundo `=` (2) sí cumple `2 > 1`, así que queda **dentro** del valor:
`(a = (b = 3))`.

**`i++ + 1`**: tras `i`, el `++` (POSTFIX 11) se absorbe como postfijo → `(i++)`; luego `+`
→ `((i++) + 1)`.

**`-a + b`**: el `-` prefijo parsea su operando con PREFIX (10); `+` (8) no es mayor,
así que el `-` solo toma `a` → `((-a) + b)`.

**`cout << a + b << endl;`**: cada valor se parsea con `parseExpression(SHIFT)`. Absorbe `+`
(8 > 7) pero se detiene en el siguiente `<<` (7 > 7 es falso). Valores: `(a + b)`, `endl`.

---

## 4. Cinco entradas válidas: tokens y AST

### Ejemplo 1: `int edad = 20;`

```
INT_TYPE "int" 1:1 | IDENT "edad" 1:5 | = "=" 1:10 | INT "20" 1:12 | ; ";" 1:14 | EOF 1:15
```
```
Program
└── VarDeclaration  Type="int"  Name=edad
    └── Value: IntegerLiteral 20
```

### Ejemplo 2: `double total = precio * 2 + 1.5;`

```
DOUBLE_TYPE "double" 1:1 | IDENT "total" 1:8 | = 1:14 | IDENT "precio" 1:16 | * 1:23
| INT "2" 1:25 | + 1:27 | DOUBLE "1.5" 1:29 | ; 1:32 | EOF 1:33
```
```
Program
└── VarDeclaration  Type="double"  Name=total
    └── Value: InfixExpression +
        ├── Left: InfixExpression *
        │   ├── Left: Identifier precio
        │   └── Right: IntegerLiteral 2
        └── Right: DoubleLiteral 1.5
```
`String()`: `double total = ((precio * 2) + 1.5);` — el `*` quedó más abajo porque tiene
más precedencia (se calcula primero).

### Ejemplo 3: `if (edad >= 18 && ok) { cout << "mayor" << endl; } else { edad++; }`

```
IF 1:1 | ( 1:4 | IDENT edad 1:5 | >= 1:10 | INT 18 1:13 | && 1:16 | IDENT ok 1:19 | ) 1:21
| { 1:23 | COUT 1:25 | << 1:30 | STRING "mayor" 1:33 | << 1:41 | ENDL 1:44 | ; 1:48 | } 1:50
| ELSE 1:52 | { 1:57 | IDENT edad 1:59 | ++ 1:63 | ; 1:65 | } 1:67 | EOF 1:68
```
```
Program
└── IfExpression
    ├── Condition: InfixExpression &&
    │   ├── Left: InfixExpression >=
    │   │   ├── Left: Identifier edad
    │   │   └── Right: IntegerLiteral 18
    │   └── Right: Identifier ok
    ├── Consequence: BlockStatement
    │   └── CoutStatement
    │       ├── Values[0]: StringLiteral "mayor"
    │       └── Values[1]: EndlLiteral
    └── Alternative: BlockStatement
        └── ExpressionStatement
            └── PostfixExpression ++
                └── Left: Identifier edad
```
`&&` (4) tiene menos precedencia que `>=` (6): primero se agrupa la comparación.
Si fuera `else if (...)`, el `Alternative` sería un bloque cuya única sentencia es otro
`IfExpression`.

### Ejemplo 4: `for (int i = 0; i < 3; i++) { total += arr[i]; }`

```
FOR 1:1 | ( 1:5 | INT_TYPE 1:6 | IDENT i 1:10 | = 1:12 | INT 0 1:14 | ; 1:15 | IDENT i 1:17
| < 1:19 | INT 3 1:21 | ; 1:22 | IDENT i 1:24 | ++ 1:25 | ) 1:27 | { 1:29 | IDENT total 1:31
| += 1:37 | IDENT arr 1:40 | [ 1:43 | IDENT i 1:44 | ] 1:45 | ; 1:46 | } 1:48 | EOF 1:49
```
```
Program
└── ForExpression
    ├── Init: VarDeclaration  Type="int"  Name=i
    │   └── Value: IntegerLiteral 0
    ├── Condition: InfixExpression <
    │   ├── Left: Identifier i
    │   └── Right: IntegerLiteral 3
    ├── Update: PostfixExpression ++
    │   └── Left: Identifier i
    └── Body: BlockStatement
        └── ExpressionStatement
            └── AssignExpression  Operator="+="
                ├── Left: Identifier total
                └── Value: IndexExpression
                    ├── Left: Identifier arr
                    └── Index: Identifier i
```

### Ejemplo 5: `int suma(int a, int b) { return a + b; }`

```
INT_TYPE 1:1 | IDENT suma 1:5 | ( 1:9 | INT_TYPE 1:10 | IDENT a 1:14 | , 1:15 | INT_TYPE 1:17
| IDENT b 1:21 | ) 1:22 | { 1:24 | RETURN 1:26 | IDENT a 1:33 | + 1:35 | IDENT b 1:37 | ; 1:38
| } 1:40 | EOF 1:41
```
```
Program
└── FunctionDeclaration  ReturnType="int"  Name=suma  Params=[int a, int b]
    └── Body: BlockStatement
        └── ReturnStatement
            └── ReturnValue: InfixExpression +
                ├── Left: Identifier a
                └── Right: Identifier b
```
`parseDeclaration` leyó `int suma`, vio `(` en `peekToken` y llamó a
`parseFunctionDeclaration`.

---

## 5. Cinco entradas inválidas: qué error y en qué etapa

| # | Entrada | Mensaje exacto | Etapa |
|---|---|---|---|
| 1 | `int x = 5 @;` | `Error léxico [línea 1, columna 11]: carácter no válido '@'` | **Lexer** (`@` no inicia ningún token). El parser lo descarta y la declaración queda bien: un solo error. |
| 2 | `string s = "hola;` | `Error léxico [línea 1, columna 12]: cadena sin cerrar: falta '"'` y luego `Error sintáctico [línea 1, columna 18]: se esperaba una expresión pero se encontró el fin del archivo` | **Lexer** primero; como el token roto se descarta, al parser le falta el valor después de `=` → también **sintáctico**. |
| 3 | `int x = 5` | `Error sintáctico [línea 1, columna 10]: se esperaba ';' pero se encontró el fin del archivo` | **Parser** (`expectPeek(';')` en `parseDeclaration`). Los tokens son todos válidos. |
| 4 | `if (x > 1 { y = 2; }` | `Error sintáctico [línea 1, columna 11]: se esperaba ')' pero se encontró '{'` | **Parser** (`parseCondition`). `synchronize` salta el bloque para no dar más errores. |
| 5 | `5 = x;` | `Error sintáctico [línea 1, columna 3]: no se puede asignar a 5: se esperaba una variable o un elemento de arreglo` | **Parser** (`parseAssignExpression` → `isAssignable`). |

Dos casos para comparar:
- `cout << a < b;` → `Error sintáctico [línea 1, columna 11]: se esperaba ';' pero se
  encontró '<'`. El `<` (6) tiene **menos** precedencia que `<<` (7), así que el valor de
  `cout` termina en `a`. Igual que en C++: hay que escribir `cout << (a < b);`.
- `int x = 10 / 0;` → **no hay error** en lexer ni parser: el AST
  `int x = (10 / 0);` es correcto. El error sale en el **evaluador** (de ejecución).
  Lo mismo pasa con una variable no declarada o con `int x = "hola";` (tipos).

**Errores en cascada.** En `int x = ;` + `int y = 2;` + `y = ;` salen exactamente 2
errores (líneas 1 y 3), no uno por cada token raro: después de cada error,
`synchronize` salta hasta el `;`.

---

## 6. Cómo se agrega una construcción nueva

Receta general: **token** (¿necesita un tipo de token nuevo?) → **lexer** (¿cómo se
reconoce?) → **ast** (¿qué nodo lo representa?) → **parser** (¿qué función lo parsea y
dónde se registra?) → **pruebas** → avisar al evaluador y al REPL.

### 6.1 Así se agregó `do { ... } while (cond);`

1. **token/token.go**: constante `DO = "DO"` y `"do": DO` en `keywords`. (`WHILE` ya existía.)
2. **lexer**: nada. `do` es una palabra; `readIdentifier` + `LookupIdent` ya la reconocen.
3. **ast/ast.go**: nodo `DoWhileStatement{Token, Body *BlockStatement, Condition Expression}`
   con `statementNode()`, `TokenLiteral()` y `String()`.
4. **parser/parser.go**:
   - en `parseStatement`: `case token.DO: return p.parseDoWhileStatement()`;
   - `parseDoWhileStatement`: `parseBody()` para el cuerpo, `expectPeek(WHILE)`,
     `parseCondition()` para `( cond )` y `expectPeek(SEMICOLON)`.
5. **pruebas**: `TestOpcionales/do-while` y los errores `do-while sin ';'` y `do sin while`.
6. **Persona 2** agrega un `case *ast.DoWhileStatement` (ejecutar el cuerpo y luego evaluar
   la condición); **Persona 3** agrega su `case` en `describirNodo`.

### 6.2 Así se agregó `+=` (asignación compuesta)

1. **token**: `PLUS_ASSIGN = "+="`.
2. **lexer**: en el `case '+'` de `NextToken`, si `peekChar()` es `'='` →
   `makeTwoCharToken(token.PLUS_ASSIGN)`. (Se revisa antes `'+'` para `++`.)
3. **ast**: no hizo falta un nodo nuevo; `AssignExpression` ganó el campo `Operator`.
4. **parser**: `token.PLUS_ASSIGN: ASSIGN` en la tabla `precedences` y
   `registerInfix(token.PLUS_ASSIGN, p.parseAssignExpression)`. La misma función de `=`
   guarda `Operator: p.curToken.Literal`.
5. **evaluador**: `x += v` se calcula como `x = x + v`.

### 6.3 Ejercicio: ¿cómo agregarías el operador ternario `c ? a : b`?

Tokens `QUESTION "?"` y `COLON ":"`; en el lexer, dos `case` de un carácter; nodo
`ConditionalExpression{Token, Condition, Consequence, Alternative}`; en el parser, un
nivel de precedencia nuevo entre ASSIGN y OR, `registerInfix(QUESTION, ...)` que parsea
la expresión del medio, exige `expectPeek(COLON)` y parsea la última (asociativo a la
derecha, como la asignación).

---

## 7. Preguntas tipo examen (respuesta corta)

1. **¿Por qué el lexer no detecta que falta un `;`?** Porque no sabe de orden: solo
   produce tokens. El orden lo revisa el parser.
2. **¿De dónde salen la línea y la columna de un error sintáctico?** Del token
   encontrado (`peekToken` o `curToken`): el lexer las guardó en `Line`/`Column`.
3. **¿Para qué sirve `peekChar` y para qué `peekToken`?** Los dos "miran adelante sin
   avanzar": `peekChar` un carácter (lexer: `=` vs `==`), `peekToken` un token (parser:
   `int f(` es función, `int f =` es variable).
4. **¿Por qué `else if` no tiene nodo propio?** Porque el cuerpo del `else` puede ser
   cualquier sentencia, y un `if` es una sentencia. `parseBody` lo envuelve en un bloque.
5. **En `if (a) if (b) x = 1; else x = 2;` ¿de quién es el `else`?** Del `if (b)`: el `if`
   de adentro revisa primero si viene `else`.
6. **¿Qué devuelve el lexer para `"a\n"`?** `STRING` con literal de 2 caracteres: `a` y un
   salto de línea real.
7. **¿Qué hace `synchronize`?** Después de un error, salta tokens hasta `;` o hasta cerrar
   el bloque, para no reportar errores falsos que solo son consecuencia del primero.
8. **¿Por qué las funciones de sentencias devuelven `ast.Statement` y no un puntero?**
   En Go, un puntero `nil` guardado en una interfaz **no** es `nil`; el `if stmt != nil`
   de `ParseProgram` lo dejaría pasar. Devolviendo la interfaz con `return nil` se evita.
9. **¿Qué AST da `a = b = 3;`? ¿Y `1 - 2 - 3;`?** `(a = (b = 3))` (derecha) y
   `((1 - 2) - 3)` (izquierda).
10. **¿En qué etapa falla `int x = "hola";`?** En ninguna de las mías: es sintácticamente
    correcto. El error de tipos lo debe dar el evaluador.
