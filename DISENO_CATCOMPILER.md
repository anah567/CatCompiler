# CatCompiler — Documento de diseño

**Proyecto:** Intérprete de un subconjunto de C++  
**Asignatura:** Lenguajes y Compiladores  
**Integrantes:** Ana Sofia Henao, Miguel Angel Quintero y Samuel Tabares

## 1. Descripción general y arquitectura

CatCompiler es un intérprete educativo desarrollado en Go que permite analizar y ejecutar programas escritos en un subconjunto del lenguaje C++.

El proyecto está dividido en componentes que trabajan de manera consecutiva:

**Código fuente → Lexer → Tokens → Parser → AST → Evaluator → Resultado**

- **Token:** define los tipos de tokens y las palabras reservadas del lenguaje.
- **Lexer:** lee el código fuente y lo divide en tokens, como identificadores, números, operadores y palabras reservadas.
- **Parser:** recibe los tokens y comprueba que estén organizados según las reglas del lenguaje.
- **AST:** representa el programa mediante un árbol de sintaxis abstracta que contiene declaraciones, expresiones y estructuras de control.
- **Objects:** define los valores que utiliza el intérprete durante la ejecución y los entornos donde se almacenan las variables.
- **Evaluator:** recorre el AST, realiza las operaciones y ejecuta las instrucciones.
- **REPL:** permite analizar tokens, visualizar el AST, ejecutar instrucciones y cargar archivos `.cpp`.

El programa puede utilizarse desde la terminal en modo interactivo mediante `go run .` o ejecutar un archivo completo mediante `go run . archivo.cpp`. Cuando el archivo contiene una función `main()`, esta se ejecuta automáticamente.

## 2. Gramática del lenguaje

El parser utiliza un subconjunto de C++ que incluye declaraciones tipadas, expresiones, condicionales, ciclos, funciones y arreglos.

A continuación, se presenta una versión resumida de sus principales reglas en notación EBNF:

```ebnf
programa       = { sentencia } ;

sentencia      = declaracion
               | asignacion, ";"
               | condicional
               | ciclo_while
               | ciclo_for
               | retorno
               | salida
               | expresion, ";" ;

declaracion    = tipo, identificador,
                 [ "=", expresion ], ";" ;

tipo           = "int" | "double" | "bool"
               | "string" ;

condicional    = "if", "(", expresion, ")",
                 bloque, [ "else", bloque ] ;

ciclo_while    = "while", "(", expresion, ")",
                 bloque ;

ciclo_for      = "for", "(", inicio, ";",
                 condicion, ";", actualizacion, ")",
                 bloque ;

funcion        = tipo_retorno, identificador,
                 "(", [ parametros ], ")", bloque ;

retorno        = "return", [ expresion ], ";" ;

salida         = "cout", "<<", expresion,
                 { "<<", expresion }, ";" ;

bloque         = "{", { sentencia }, "}" ;
```

Esta gramática es un resumen explicativo y no representa todas las reglas internas del parser. Las expresiones y la precedencia de los operadores se procesan mediante un parser de Pratt.

## 3. Decisiones de implementación

**Lenguaje Go:** se eligió Go para separar el intérprete en paquetes, manejar estructuras de datos y desarrollar pruebas automáticas.

**Variables y ámbitos:** las variables se almacenan en entornos de ejecución. Se utilizan entornos relacionados entre sí para permitir variables globales y locales. Esto evita que una variable local modifique accidentalmente otra variable con el mismo nombre en un ámbito externo.

**Funciones:** las funciones almacenan sus parámetros, tipo de retorno, cuerpo y entorno. Cuando se llaman, se crea un entorno para sus variables locales. Este mecanismo también permite ejecutar funciones recursivas.

**Arreglos:** se representan como colecciones de valores. El evaluador permite acceder a sus elementos mediante índices y modificar posiciones válidas.

**Control de flujo:** las estructuras `if`, `while` y `for` se ejecutan evaluando sus condiciones y recorriendo las instrucciones correspondientes. Las operaciones lógicas `&&` y `||` utilizan evaluación de cortocircuito.

**Salida:** la instrucción `cout` permite mostrar resultados. La salida se dirige mediante un `io.Writer`, lo que facilita capturarla durante las pruebas automáticas.

**Errores:** el intérprete identifica problemas sintácticos y errores de ejecución, como el uso de variables no declaradas o llamadas a funciones con una cantidad incorrecta de argumentos.

## 4. Limitaciones

CatCompiler no implementa el estándar completo de C++. Su alcance se limita a las construcciones desarrolladas para el proyecto académico.

Entre sus limitaciones se encuentran la ausencia de soporte general para clases, herencia, plantillas, punteros y bibliotecas estándar completas. Algunas construcciones pueden ser reconocidas durante el análisis sintáctico sin contar con una implementación completa en el evaluador.

Por lo tanto, un programa válido para un compilador C++ convencional no necesariamente puede ejecutarse en CatCompiler.

## 5. Uso de inteligencia artificial

Durante el desarrollo se utilizó inteligencia artificial como herramienta de apoyo para comprender conceptos, analizar errores, proponer casos de prueba, revisar código y mejorar la documentación.

Las sugerencias se contrastaron con la implementación y se verificaron mediante pruebas automáticas y ejecuciones de programas. La herramienta no reemplazó la necesidad de comprender las decisiones técnicas ni de comprobar el comportamiento del intérprete.

## 6. Verificación

El proyecto cuenta con pruebas unitarias e integrales para comprobar sus componentes.

La carpeta `tests/` contiene programas válidos e inválidos, archivos con resultados esperados y el script `run_tests.sh`, que permite ejecutar las pruebas mediante un solo comando.

En la última ejecución se obtuvieron **11 pruebas aprobadas y 0 fallidas**, incluyendo variables, operaciones, condicionales, ciclos, funciones, recursividad, ámbitos, arreglos y detección de errores.

Adicionalmente, las pruebas automatizadas de Go se ejecutan mediante:

```bash
go test ./...
```

El proyecto puede compilarse mediante:

```bash
go build ./...
```