# CatCompiler

CatCompiler es un intérprete educativo desarrollado en Go que permite analizar y ejecutar un subconjunto del lenguaje C++.

El proyecto incluye un analizador léxico (Lexer), un analizador sintáctico (Parser), un árbol de sintaxis abstracta (AST), un evaluador (Evaluator) y una interfaz interactiva de consola (REPL).

## Integrantes

- Ana Sofia Henao
- Miguel Angel Quintero
- Samuel Tabares

## 1. Requisitos

- Go en una versión compatible con la indicada en `go.mod`.
- Una terminal para ejecutar los comandos.
- No se requieren dependencias externas adicionales.

## 2. Ejecución

Desde la carpeta principal del proyecto:

```bash
go run .
```

Este comando inicia el REPL de CatCompiler, que permite trabajar en tres modos:

1. **Lexer:** muestra los tokens del código.
2. **Parser:** muestra la estructura del AST.
3. **Evaluator:** ejecuta las instrucciones.

También se puede ejecutar directamente un archivo `.cpp`:

```bash
go run . examples/variables.cpp
```

Cuando el archivo contiene una función `main()`, el intérprete la ejecuta automáticamente.

## 3. Modos del REPL

### Lexer

Convierte el código fuente en tokens, identificando su tipo, contenido, línea y columna.

Ejemplo:

```cpp
int edad = 20;
```

El resultado incluye los tokens correspondientes al tipo `int`, el identificador `edad`, el operador `=`, el número `20` y el punto y coma.

### Parser

Recibe los tokens del lexer y construye el árbol de sintaxis abstracta (AST).

Ejemplo:

```cpp
int edad = 20;
```

Representación simplificada:

```text
Program
└── VarDeclaration int edad
    └── Valor: IntegerLiteral 20
```

### Evaluator

Recorre el AST y ejecuta las instrucciones.

Ejemplo:

```cpp
int numero = 10;
cout << numero * 2 << endl;
```

Salida:

```text
20
```

El REPL puede mostrar `null` después de instrucciones que no producen un valor de resultado.

## 4. Comandos del REPL

| Comando | Función |
|---|---|
| `:lexer` | Cambia al modo Lexer |
| `:parser` | Cambia al modo Parser |
| `:evaluator` | Cambia al modo Evaluator |
| `:cargar archivo.cpp` | Ejecuta un archivo `.cpp` en el entorno actual |
| `:ayuda` | Muestra los comandos disponibles |
| `:salir` | Finaliza el REPL |

Ejemplo:

```text
:cargar examples/funciones.cpp
```

## 5. Funcionalidades implementadas

CatCompiler permite trabajar con:

- Declaraciones de variables tipadas.
- Tipos `int`, `double`, `bool` y `string`.
- Operaciones aritméticas, comparaciones y precedencia de operadores.
- Operadores lógicos con evaluación de cortocircuito.
- Condicionales `if`, `else if` y `else`.
- Ciclos `for`, `while` y `do-while`.
- Instrucciones `break` y `continue`.
- Funciones con parámetros y tipos de retorno.
- Funciones recursivas, con protección frente a recursión excesiva.
- Variables globales, locales y *shadowing*.
- Arreglos e indexación con detección de índices fuera de rango.
- Operadores de asignación, incremento y asignación compuesta, como `++` y `+=`.
- Comentarios en el código fuente.
- Salida por pantalla mediante `cout` y `endl`.
- Funciones integradas y métodos básicos para cadenas y arreglos.
- Detección de errores léxicos, sintácticos y de ejecución.
- Información de línea y columna en errores léxicos y sintácticos.
- Ejecución automática de `main()` al cargar archivos `.cpp`.

### 5.1. Funciones integradas

El intérprete incluye las siguientes funciones:

| Función | Descripción |
|---|---|
| `to_string()` | Convierte un valor a una cadena de texto |
| `stoi()` | Convierte una cadena a un número entero |
| `stod()` | Convierte una cadena a un número decimal |
| `abs()` | Obtiene el valor absoluto |
| `max()` | Obtiene el mayor valor |
| `min()` | Obtiene el menor valor |
| `pow()` | Calcula una potencia |
| `sqrt()` | Calcula una raíz cuadrada |

### 5.2. Métodos integrados

| Método | Descripción |
|---|---|
| `.size()` | Obtiene el tamaño de una cadena o arreglo |
| `.length()` | Obtiene la longitud de una cadena |
| `.push_back()` | Agrega un elemento al final de un arreglo |

Estas funciones y métodos forman parte de la implementación educativa de CatCompiler y no representan toda la biblioteca estándar de C++.

## 6. Ejemplos

La carpeta `examples/` contiene ocho programas que demuestran las funcionalidades del intérprete.

| Archivo | Qué demuestra |
|---|---|
| `variables.cpp` | Variables y tipos de datos |
| `ciclos.cpp` | Estructuras repetitivas |
| `funciones.cpp` | Funciones y cálculo factorial |
| `condicionales.cpp` | Condicionales y decisiones |
| `arreglos.cpp` | Declaración, acceso y modificación de arreglos |
| `recursion.cpp` | Funciones recursivas |
| `ambitos.cpp` | Diferencias entre variables globales y locales |
| `integradas.cpp` | Uso de funciones y métodos integrados |

Para ejecutar cualquiera de ellos:

```bash
go run . examples/variables.cpp
go run . examples/ciclos.cpp
go run . examples/funciones.cpp
go run . examples/condicionales.cpp
go run . examples/arreglos.cpp
go run . examples/recursion.cpp
go run . examples/ambitos.cpp
go run . examples/integradas.cpp
```

Por ejemplo, `examples/funciones.cpp` produce:

```text
Factorial de 5: 120
```

## 7. Estructura del proyecto

| Ruta | Responsabilidad |
|---|---|
| `main.go` | Inicia el REPL o ejecuta archivos `.cpp` |
| `token/` | Define los tipos de tokens |
| `lexer/` | Convierte el código fuente en tokens |
| `parser/` | Construye el AST |
| `ast/` | Define los nodos del árbol sintáctico |
| `objects/` | Define los objetos y entornos de ejecución |
| `evaluator/` | Ejecuta las instrucciones representadas en el AST |
| `evaluator/builtins.go` | Implementa las funciones y métodos integrados |
| `repl/` | Implementa los modos interactivos y la ejecución de archivos |
| `examples/` | Contiene los programas `.cpp` de demostración |
| `tests/valid/` | Contiene programas que deben ejecutarse correctamente |
| `tests/invalid/` | Contiene programas que deben producir errores |
| `tests/expected/` | Guarda los resultados esperados de las pruebas |
| `tests/run_tests.sh` | Ejecuta las pruebas de integración |
| `DISENO_CATCOMPILER.md` | Documento de diseño y arquitectura del intérprete |

## 8. Pruebas

El proyecto cuenta con pruebas automáticas de Go y pruebas de integración basadas en programas `.cpp`.

### Pruebas de Go

Para ejecutar todas las pruebas:

```bash
go test ./...
```

Para ver las pruebas del paquete REPL:

```bash
go test ./repl -v
```

### Pruebas de integración

Para ejecutar todos los programas de prueba con un solo comando:

```bash
bash tests/run_tests.sh
```

El script ejecuta nueve programas válidos y seis programas inválidos, compara los resultados con los esperados y muestra cuáles pruebas fueron aprobadas.

En la última ejecución se obtuvo:

```text
Aprobadas: 15
Fallidas:  0
```

Los casos comprobados incluyen variables, operaciones, condicionales, ciclos, funciones, recursividad, ámbitos, arreglos y funciones integradas.

También se comprueba la detección de errores por argumentos incorrectos, errores léxicos, errores sintácticos, índices fuera de rango, recursión infinita y variables no declaradas.

### Compilación

Para comprobar que el proyecto compila correctamente:

```bash
go build ./...
```

## 9. Documento de diseño

El archivo `DISENO_CATCOMPILER.md` presenta la arquitectura del intérprete, una gramática EBNF resumida, las principales decisiones de implementación, las limitaciones del lenguaje y el uso de inteligencia artificial durante el desarrollo.

## 10. Alcance y limitaciones

CatCompiler es un proyecto académico cuyo propósito es demostrar las principales fases de un intérprete: análisis léxico, análisis sintáctico y evaluación.

No pretende implementar todas las características del estándar C++.

Entre sus limitaciones conocidas se encuentran:

- No implementa `vector` de la biblioteca estándar de C++.
- No implementa entrada de datos mediante `cin`.
- No admite matrices multidimensionales como `int m[2][2]`.
- Los errores de ejecución no siempre incluyen línea y columna.
- Las funciones integradas representan únicamente un conjunto reducido de operaciones.
- La protección contra recursión excesiva utiliza un límite interno de llamadas, no un manejo completo de la pila de ejecución de C++.

Estas limitaciones permiten mantener el alcance del proyecto centrado en el funcionamiento del intérprete.