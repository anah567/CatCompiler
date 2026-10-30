# CatCompiler — Documento de diseño

**Proyecto:** Intérprete de un subconjunto de C++  
**Asignatura:** Lenguajes y Compiladores  
**Integrantes:** Ana Sofia Henao, Miguel Angel Quintero y Samuel Tabares

## 1. Descripción general y arquitectura

CatCompiler es un intérprete educativo desarrollado en Go que analiza y ejecuta programas escritos en un subconjunto de C++.

Su funcionamiento sigue estas etapas:

**Código fuente → Lexer → Tokens → Parser → AST → Evaluator → Resultado**

- **Token:** define los tipos de tokens, operadores y palabras reservadas.
- **Lexer:** convierte el código en tokens e identifica su ubicación.
- **Parser:** comprueba la sintaxis y construye el AST.
- **AST:** representa las declaraciones, expresiones y estructuras del programa.
- **Objects:** define los valores, funciones integradas y entornos de ejecución.
- **Evaluator:** recorre el AST y ejecuta las instrucciones.
- **REPL:** permite analizar tokens, visualizar el AST y ejecutar código o archivos `.cpp`.

El intérprete se inicia con `go run .` y permite ejecutar archivos mediante `go run . archivo.cpp`. Si existe una función `main()`, se ejecuta automáticamente.

## 2. Gramática del lenguaje

La siguiente gramática EBNF resume las principales construcciones implementadas. No incluye todas las reglas internas del parser.

```ebnf
programa = { funcion | sentencia } ;

sentencia = declaracion | bloque
          | condicional | ciclo_while
          | ciclo_do | ciclo_for
          | retorno | ruptura | continuacion
          | salida | expresion, ";" ;

bloque         = "{", { sentencia }, "}" ;

tipo           = "int" | "double" | "bool" | "string" ;
tipo_retorno   = tipo | "void" ;

declaracion    = [ "const" ], tipo, identificador,
                 [ "[", expresion, "]" ],
                 [ "=", expresion ], ";" ;

funcion        = tipo_retorno, identificador, "(",
                 [ parametros ], ")", bloque ;
parametros     = tipo, identificador,
                 { ",", tipo, identificador } ;

condicional    = "if", "(", expresion, ")", sentencia,
                 { "else", "if", "(", expresion, ")",
                   sentencia },
                 [ "else", sentencia ] ;

ciclo_while    = "while", "(", expresion, ")", sentencia ;
ciclo_do       = "do", sentencia,
                 "while", "(", expresion, ")", ";" ;
ciclo_for      = "for", "(", [ inicio ], ";",
                 [ expresion ], ";",
                 [ expresion ], ")", sentencia ;

retorno        = "return", [ expresion ], ";" ;
ruptura        = "break", ";" ;
continuacion   = "continue", ";" ;

salida         = "cout", "<<", expresion,
                 { "<<", expresion }, ";" ;

expresion      = asignacion ;
asignacion     = logica_o,
                 [ ( "=" | "+=" | "-=" | "*=" | "/=" | "%=" ), asignacion ] ;
logica_o       = logica_y, { "||", logica_y } ;
logica_y       = igualdad, { "&&", igualdad } ;
igualdad       = comparacion,
                 { ( "==" | "!=" ), comparacion } ;
comparacion    = suma,
                 { ( "<" | ">" | "<=" | ">=" ), suma } ;
suma           = producto,
                 { ( "+" | "-" ), producto } ;
producto       = unaria,
                 { ( "*" | "/" | "%" ), unaria } ;
unaria         = [ "!" | "-" | "+" | "++" | "--" ],
                 postfija ;

postfija       = primaria, { "++" | "--" } ;

primaria       = literal | identificador
               | "(", expresion, ")" ;
```

Las llamadas a funciones, el acceso a arreglos, los métodos integrados y otras operaciones se procesan mediante reglas adicionales. La precedencia de operadores se implementa con un parser de Pratt.

## 3. Decisiones de implementación

**Lenguaje Go:** se utilizó para organizar el proyecto en paquetes independientes y facilitar las pruebas.

**Variables y ámbitos:** se utilizan entornos relacionados entre sí para manejar variables globales, locales y *shadowing*, permitiendo declarar una variable local con el mismo nombre de otra externa.

**Funciones:** almacenan sus parámetros, cuerpo, tipo de retorno y entorno. Cada llamada crea un entorno local. Se permite la recursión y se controla la profundidad de llamadas para evitar un desbordamiento de pila por recursión excesiva.

**Arreglos:** se almacenan como colecciones de valores. Permiten consultar y modificar elementos, con detección de índices fuera de rango.

**Control de flujo:** se implementan `if`, `else if`, `else`, `for`, `while`, `do-while`, `break` y `continue`. Los operadores `&&` y `||` utilizan evaluación de cortocircuito.

**Funciones integradas:** se implementaron `to_string`, `stoi`, `stod`, `abs`, `max`, `min`, `pow` y `sqrt`, además de los métodos `.size()`, `.length()` y `.push_back()`.

**Salida:** `cout` utiliza un `io.Writer` para mostrar resultados y permitir capturarlos en las pruebas. Los booleanos se imprimen como `1` y `0`.

**Errores:** se detectan errores léxicos y sintácticos con línea y columna, así como errores de ejecución por variables no declaradas, argumentos incorrectos, índices inválidos y recursión excesiva.

## 4. Limitaciones

CatCompiler no implementa el estándar completo de C++. Sus principales limitaciones son:

- No admite `vector`, entrada mediante `cin` ni matrices multidimensionales como `int m[2][2]`.
- No implementa de forma general clases, herencia, plantillas ni punteros.
- Solo incorpora un conjunto reducido de funciones de la biblioteca estándar.
- Los errores de ejecución no siempre incluyen línea y columna.
- La protección frente a recursión excesiva utiliza un límite interno de llamadas.

Por estas razones, un programa válido en C++ convencional no necesariamente puede ejecutarse en CatCompiler.

## 5. Uso de inteligencia artificial

Durante el desarrollo se utilizó inteligencia artificial como apoyo para comprender conceptos, analizar errores, proponer casos de prueba, revisar código y mejorar la documentación.

Las sugerencias se contrastaron con la implementación y se verificaron mediante pruebas automáticas y ejecución de programas. Las decisiones técnicas y la validación de resultados se realizaron como parte del trabajo del equipo.

## 6. Verificación

El proyecto incluye pruebas automáticas de Go y pruebas de integración con programas `.cpp` válidos e inválidos.

La carpeta `tests/` contiene los programas, los resultados esperados y el script `run_tests.sh`.

En la última ejecución se obtuvieron **16 pruebas aprobadas y 0 fallidas**: diez casos válidos y seis casos de error. Se comprobaron variables, operaciones, condicionales, ciclos, funciones, recursión, ámbitos, arreglos, funciones integradas y manejo de errores.

Comandos de verificación:

```bash
go test ./...
go build ./...
bash tests/run_tests.sh
```

Las pruebas de Go, la compilación y las 16 pruebas de integración finalizaron correctamente.