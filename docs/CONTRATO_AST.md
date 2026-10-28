# Contrato del AST — subconjunto de C++

Autor: Persona 1 (sintaxis: `token`, `lexer`, `parser`, `ast`).
Este documento fija **qué nodos produce el parser** y qué contiene cada uno, para que
Persona 2 (evaluador) y Persona 3 (REPL / ejemplos / pruebas) puedan trabajar sin
esperar a que el parser esté terminado.

**Estado: definitivo.** Las decisiones de este contrato (y las de `DECISIONES_P1.md` §2)
se tomaron sin poder consultarlas con el grupo; son el punto de partida acordado. Si
alguien necesita cambiar un nodo, se cambia primero este documento y se avisa al grupo.

## 0. Lo que NO cambia (firmas públicas)

```go
l := lexer.New(codigo)          // string → *lexer.Lexer
tok := l.NextToken()            // token.Token; termina con tok.Type == token.EOF
p := parser.New(l)              // *parser.Parser
program := p.ParseProgram()     // *ast.Program
errores := p.Errors()           // []string (ya traen tipo de error, línea y columna)
```

Las interfaces `ast.Node`, `ast.Statement` y `ast.Expression` siguen iguales.
Todos los nodos conservan su campo `Token token.Token` y su método `String()`.

## 1. Cambio en `token.Token` (aditivo)

```go
type Token struct {
    Type    TokenType
    Literal string
    Line    int // NUEVO: línea donde empieza el token (desde 1)
    Column  int // NUEVO: columna donde empieza el token (desde 1)
}
```

Para reportar un error de ejecución con posición, el evaluador puede usar el token
del nodo: `n.Token.Line`, `n.Token.Column`. Ejemplo para una división por cero en
`InfixExpression`: el token es el `/`.

El token `ILLEGAL` (error léxico) lleva en `Literal` el **mensaje** del error
(por ejemplo `carácter no válido '@'`). El parser lo convierte en un
"Error léxico [línea L, columna C]: ..." y lo agrega a `Errors()`.

## 2. Resumen del cambio de nodos

| Estado | Nodos |
|---|---|
| **Se mantienen** | `Program`, `ExpressionStatement`, `BlockStatement`, `ReturnStatement`, `Identifier`, `IntegerLiteral`, `StringLiteral`, `Boolean`, `PrefixExpression`, `InfixExpression`, `AssignExpression`, `IfExpression`, `WhileExpression`, `ForExpression`, `CallExpression`, `IndexExpression` |
| **Cambian** | `AssignExpression` gana `Operator string`. `PrefixExpression` acepta también `++` y `--`. `InfixExpression` acepta también `&&`, `\|\|`, `<<`, `>>`. `IfExpression`, `WhileExpression` y `ForExpression` **ahora son sentencias** (implementan `Statement`, ya no `Expression`) y van directo en la lista de sentencias, no dentro de un `ExpressionStatement`. `ReturnStatement.ReturnValue` puede ser `nil` (`return;`). `ForExpression.Init` puede ser `VarDeclaration`/`VarDeclarationList`. |
| **Se agregan** | `VarDeclaration`, `VarDeclarationList`, `FunctionDeclaration` (+ `Param`, que no es nodo), `DoubleLiteral`, `EndlLiteral`, `PostfixExpression`, `CoutStatement`, `MemberExpression`, `DoWhileStatement`, `BreakStatement`, `ContinueStatement` |
| **Se eliminan** | `LetStatement`, `FunctionLiteral`, `ArrayLiteral`, `HashLiteral` (y `HashPair`) |

Decisión sobre `endl`: es un nodo propio, **`EndlLiteral`**, no un `Identifier`.
Así el evaluador no tiene que buscar una variable llamada "endl": cuando ve un
`EndlLiteral` produce `"\n"`.

Decisión sobre arreglos: **no** hay nodo `ArrayInitializer`; la lista `{1, 2, 3}` va
en el campo `Elements` de `VarDeclaration` (solo se permite al declarar un arreglo).

Las directivas `#include <...>` (las descarta el lexer) y `using namespace std;`
(la parsea el parser pero **no genera nodo**) no aparecen en el AST.

## 3. Nodos, campos y ejemplos

Notación del árbol: `Nodo campo=valor`, y los hijos con su nombre de campo.

### Program
```go
type Program struct { Statements []Statement }
```
Raíz. Puede contener funciones, variables globales y sentencias sueltas (para el REPL).

### VarDeclaration (sentencia)
```go
type VarDeclaration struct {
    Token     token.Token  // token del tipo ('int')
    Type      string       // "int" | "double" | "bool" | "string"
    Const     bool         // const int N = 5;
    Name      *Identifier
    IsArray   bool         // se escribieron corchetes
    ArraySize Expression   // nil si no hay tamaño:  int a[] = {...}
    Value     Expression   // nil si no hay "= valor" (o si es arreglo)
    Elements  []Expression // nil si no hay "= {...}"
}
```
```cpp
int edad = 20;
```
```
VarDeclaration Type="int" Name=edad
└── Value: IntegerLiteral 20
```
```cpp
int x;              // Value=nil
const int N = 5;    // Const=true, Value=IntegerLiteral 5
int a[5];           // IsArray=true, ArraySize=IntegerLiteral 5, Value=nil, Elements=nil
int arr[3] = {1, 2, 3};
```
```
VarDeclaration Type="int" Name=arr IsArray=true
├── ArraySize: IntegerLiteral 3
└── Elements: [IntegerLiteral 1, IntegerLiteral 2, IntegerLiteral 3]
```
Reglas que ya verifica el parser: `void x;` es error; `const` sin valor es error;
`int a[];` sin lista es error; un arreglo solo se inicializa con `{...}`.

### VarDeclarationList (sentencia)
```go
type VarDeclarationList struct {
    Token        token.Token        // token del tipo
    Declarations []*VarDeclaration  // 2 o más, en orden
}
```
```cpp
int a = 1, b;
```
```
VarDeclarationList
├── VarDeclaration Type="int" Name=a  Value=IntegerLiteral 1
└── VarDeclaration Type="int" Name=b  Value=nil
```
Con una sola variable el parser devuelve un `VarDeclaration`, nunca una lista de 1.

### FunctionDeclaration (sentencia) y Param
```go
type Param struct {
    Type    string      // "int" | "double" | "bool" | "string"
    Name    *Identifier // nil solo en prototipos:  int f(int);
    IsArray bool        // int v[]
}
type FunctionDeclaration struct {
    Token      token.Token     // token del tipo de retorno
    ReturnType string          // "int" | "double" | "bool" | "string" | "void"
    Name       *Identifier
    Params     []Param         // vacío (no nil) si no hay parámetros
    Body       *BlockStatement // nil si es un prototipo
}
```
```cpp
int suma(int a, int b) { return a + b; }
```
```
FunctionDeclaration ReturnType="int" Name=suma Params=[int a, int b]
└── Body: BlockStatement
    └── ReturnStatement
        └── ReturnValue: InfixExpression "+"
            ├── Left: Identifier a
            └── Right: Identifier b
```
```cpp
int f(int);   // prototipo → Body=nil, Params=[{Type:"int", Name:nil}]
```
Solo se permiten funciones en el nivel superior (no dentro de un bloque).
Si una función tiene cuerpo, todos sus parámetros tienen nombre (el parser lo exige).

### BlockStatement (sentencia)
```go
type BlockStatement struct { Token token.Token; Statements []Statement }
```
Cuerpo de funciones y estructuras de control; también `{ ... }` suelto (nuevo ámbito).
Si un `if/else/while/for/do` tiene **una sola sentencia sin llaves**, el parser la
envuelve en un `BlockStatement`. El evaluador siempre recibe un bloque.

### ExpressionStatement (sentencia)
```go
type ExpressionStatement struct { Token token.Token; Expression Expression }
```
```cpp
i++;
```
```
ExpressionStatement
└── PostfixExpression "++"
    └── Left: Identifier i
```

### ReturnStatement (sentencia)
```go
type ReturnStatement struct { Token token.Token; ReturnValue Expression } // nil en "return;"
```

### IfExpression (ahora sentencia)
```go
type IfExpression struct {
    Token       token.Token
    Condition   Expression
    Consequence *BlockStatement
    Alternative *BlockStatement // nil si no hay else
}
```
```cpp
if (a) { x = 1; } else if (b) { x = 2; } else { x = 3; }
```
```
IfExpression
├── Condition: Identifier a
├── Consequence: BlockStatement [ x = 1; ]
└── Alternative: BlockStatement
    └── IfExpression                 ← el "else if" es un if dentro del else
        ├── Condition: Identifier b
        ├── Consequence: BlockStatement [ x = 2; ]
        └── Alternative: BlockStatement [ x = 3; ]
```

### WhileExpression (ahora sentencia)
```go
type WhileExpression struct { Token token.Token; Condition Expression; Body *BlockStatement }
```
```cpp
while (total > 0) { total--; }
```
```
WhileExpression
├── Condition: InfixExpression ">" (total, 0)
└── Body: BlockStatement
    └── ExpressionStatement → PostfixExpression "--" (total)
```

### DoWhileStatement (sentencia, opcional)
```go
type DoWhileStatement struct { Token token.Token; Body *BlockStatement; Condition Expression }
```
```cpp
do { x--; } while (x > 0);
```

### ForExpression (ahora sentencia)
```go
type ForExpression struct {
    Token     token.Token
    Init      Statement  // *VarDeclaration, *VarDeclarationList o *ExpressionStatement; nil si se omite
    Condition Expression // nil si se omite
    Update    Expression // nil si se omite
    Body      *BlockStatement
}
```
```cpp
for (int i = 0; i < 3; i++) { total += arr[i]; }
```
```
ForExpression
├── Init: VarDeclaration Type="int" Name=i Value=IntegerLiteral 0
├── Condition: InfixExpression "<" (i, 3)
├── Update: PostfixExpression "++" (i)
└── Body: BlockStatement
    └── ExpressionStatement
        └── AssignExpression Operator="+="
            ├── Left: Identifier total
            └── Value: IndexExpression (arr, i)
```
Ámbito sugerido: la variable de `Init` vive solo dentro del `for`.

### CoutStatement (sentencia)
```go
type CoutStatement struct { Token token.Token; Values []Expression }
```
```cpp
cout << "Hola " << nombre << endl;
```
```
CoutStatement
├── Values[0]: StringLiteral "Hola "
├── Values[1]: Identifier nombre
└── Values[2]: EndlLiteral
```
Cada valor se parsea con precedencia mayor que `<<`, así `cout << a + b << endl;`
imprime `a + b`. Para imprimir una comparación hay que usar paréntesis, como en C++:
`cout << (a < b);`.

### BreakStatement / ContinueStatement (sentencias, opcionales)
```go
type BreakStatement struct { Token token.Token }
type ContinueStatement struct { Token token.Token }
```
El parser no verifica que estén dentro de un ciclo; eso lo decide el evaluador.

### Literales e identificadores (expresiones)
```go
type Identifier     struct { Token token.Token; Value string }
type IntegerLiteral struct { Token token.Token; Value int64 }
type DoubleLiteral  struct { Token token.Token; Value float64 }  // NUEVO: 3.5, 10.0
type StringLiteral  struct { Token token.Token; Value string }   // escapes ya convertidos
type Boolean        struct { Token token.Token; Value bool }
type EndlLiteral    struct { Token token.Token }                 // NUEVO: endl
```
`StringLiteral.Value` de `"a\n"` es `a` + salto de línea real (2 caracteres).

### PrefixExpression
```go
type PrefixExpression struct { Token token.Token; Operator string; Right Expression }
```
Operadores: `-`, `!`, `++`, `--`. Con `++`/`--` el operando es `Identifier` o
`IndexExpression`. `++i` produce el valor **nuevo**.

### PostfixExpression (NUEVO)
```go
type PostfixExpression struct { Token token.Token; Left Expression; Operator string }
```
Operadores: `++`, `--`. `Left` es `Identifier` o `IndexExpression`. `i++` produce el
valor **anterior** y luego incrementa.

### InfixExpression
```go
type InfixExpression struct { Token token.Token; Left Expression; Operator string; Right Expression }
```
Operadores: `+ - * / % == != < > <= >= && || << >>`. Se recomienda que `&&` y `||`
evalúen en cortocircuito (extensión opcional del enunciado).
```cpp
edad >= 18 && precio < 10.0
```
```
InfixExpression "&&"
├── Left: InfixExpression ">=" (edad, 18)
└── Right: InfixExpression "<" (precio, DoubleLiteral 10.0)
```

### AssignExpression (cambia: nuevo campo Operator)
```go
type AssignExpression struct { Token token.Token; Left Expression; Operator string; Value Expression }
```
`Operator` ∈ `=`, `+=`, `-=`, `*=`, `/=`, `%=`. `Left` es `Identifier` o
`IndexExpression` (lo verifica el parser). Asociativa a la derecha:
`a = b = 3` → `AssignExpression(a, =, AssignExpression(b, =, 3))`.

### CallExpression
```go
type CallExpression struct { Token token.Token; Function Expression; Arguments []Expression }
```
`Function` es `Identifier` (`factorial(5)`) o `MemberExpression` (`s.size()`).

### IndexExpression
```go
type IndexExpression struct { Token token.Token; Left Expression; Index Expression }
```
Lectura `arr[i]`; escritura `arr[i] = x;` (es el `Left` de un `AssignExpression`).

### MemberExpression (NUEVO, opcional)
```go
type MemberExpression struct { Token token.Token; Object Expression; Member *Identifier }
```
```cpp
s.size()
```
```
CallExpression
└── Function: MemberExpression (Object=Identifier s, Member=size)
```

## 4. Precedencia que produce el parser (menor → mayor)

| Nivel | Operadores | Asociatividad |
|---|---|---|
| ASSIGN | `= += -= *= /= %=` | derecha |
| OR | `\|\|` | izquierda |
| AND | `&&` | izquierda |
| EQUALS | `== !=` | izquierda |
| LESSGREATER | `< > <= >=` | izquierda |
| SHIFT | `<< >>` | izquierda |
| SUM | `+ -` | izquierda |
| PRODUCT | `* / %` | izquierda |
| PREFIX | `-x !x ++x --x` | derecha |
| POSTFIX | `x++ x-- f() a[i] s.m` | izquierda |

## 5. Qué nodo debe manejar el evaluador (Persona 2)

| Nodo | Qué hacer |
|---|---|
| `Program` | Registrar funciones y globales en orden; si existe `main`, llamarla al final (en el REPL, sin `main`, se evalúan las sentencias sueltas). |
| `VarDeclaration` | Crear la variable en el **ámbito actual**. Sin valor: valor por defecto (decisión: 0 / 0.0 / false / ""). Arreglo: crear `ArraySize` elementos o usar `Elements`. Respetar `Const`. Convertir según `Type` (p. ej. `double d = 3;` guarda 3.0). |
| `VarDeclarationList` | Evaluar cada `VarDeclaration` en orden, en el mismo ámbito. |
| `FunctionDeclaration` | Guardar la función por nombre (`Params[i].Name.Value`). Si `Body == nil` es prototipo: no hacer nada. |
| `BlockStatement` | Ámbito nuevo (local) que ve el exterior. |
| `ExpressionStatement` | Evaluar la expresión. |
| `ReturnStatement` | `ReturnValue` puede ser `nil` → retorno sin valor. |
| `IfExpression` / `WhileExpression` / `ForExpression` / `DoWhileStatement` | Ahora llegan como sentencias. En `for`, el `Init` vive en un ámbito propio del ciclo. |
| `CoutStatement` | Imprimir cada valor de `Values` sin separador; `EndlLiteral` → `"\n"`. |
| `BreakStatement` / `ContinueStatement` | Señales para el ciclo más interno (opcional). |
| `Identifier`, `IntegerLiteral`, `DoubleLiteral`, `StringLiteral`, `Boolean`, `EndlLiteral` | Valores. `DoubleLiteral` necesita un objeto `Double`/`Float` nuevo. |
| `PrefixExpression` | `-`, `!`, y `++`/`--` (modifican la variable, devuelven el valor nuevo). |
| `PostfixExpression` | `++`/`--`: modifican la variable, devuelven el valor anterior. |
| `InfixExpression` | Aritmética int/double, comparaciones, `&&`/`\|\|` (cortocircuito), `<<`/`>>` entre enteros (desplazamiento) si se quiere. |
| `AssignExpression` | Usar `Operator`: `x += v` ≡ `x = x + v`. Destino `Identifier` o `IndexExpression`. |
| `CallExpression` | Funciones de usuario e integradas; `MemberExpression` para `s.size()`/`s.length()` (opcional). |
| `IndexExpression` | Lectura con verificación de rango (error con `n.Token.Line/Column`). |
| Eliminados | Quitar los `case` de `LetStatement`, `FunctionLiteral`, `ArrayLiteral`, `HashLiteral` (y `evalHash`). `objects.Function` debe pasar a usar `[]ast.Param` o los nombres de los parámetros. |

## 6. Qué nodo debe imprimir el REPL (Persona 3)

En `repl/repl.go`, `describirNodo` necesita un `case` por nodo:

| Nodo | Texto sugerido | Hijos |
|---|---|---|
| `VarDeclaration` | `VarDeclaration int edad` (+ `const`, `[]`) | `Tamaño` (ArraySize), `Valor` (Value), `[i]` por cada Elements |
| `VarDeclarationList` | `VarDeclarationList int` | cada `VarDeclaration` |
| `FunctionDeclaration` | `FunctionDeclaration int suma(int a, int b)` (usar `Param.String()`) | `Cuerpo` (si no es nil) |
| `CoutStatement` | `CoutStatement` | `<<` por cada valor |
| `DoWhileStatement` | `DoWhileStatement` | `Cuerpo`, `Condición` |
| `BreakStatement` / `ContinueStatement` | nombre del nodo | — |
| `DoubleLiteral` | `DoubleLiteral 3.5` | — |
| `EndlLiteral` | `EndlLiteral` | — |
| `PostfixExpression` | `PostfixExpression ++` | `Izquierda` |
| `MemberExpression` | `MemberExpression .size` | `Objeto` |
| `AssignExpression` | `AssignExpression ` + `n.Operator` | igual que antes |
| `ReturnStatement` | igual | `Valor` puede ser nil → ya se muestra `<vacío>` |
| Eliminados | Quitar `LetStatement`, `FunctionLiteral`, `ArrayLiteral`, `HashLiteral` | |

Además: el modo lexer puede mostrar `tok.Line`/`tok.Column`; los `;` ya son
obligatorios (en el REPL `5 + 5` sin `;` da error sintáctico); y la extensión de
archivos `.ssg` debería pasar a `.cpp`.
