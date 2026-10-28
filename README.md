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
- Operaciones aritméticas y comparaciones.
- Operadores lógicos con evaluación de cortocircuito.
- Condicionales `if`, `else if` y `else`.
- Ciclos `for` y `while`.
- Funciones con parámetros y tipos de retorno.
- Funciones recursivas.
- Variables globales y locales.
- Arreglos e indexación.
- Operadores de asignación e incremento.
- Salida por pantalla mediante `cout` y `endl`.
- Detección de errores de sintaxis y ejecución.
- Ejecución automática de `main()` al cargar archivos `.cpp`.

El proyecto implementa un subconjunto educativo de C++, no el lenguaje completo.

## 6. Ejemplos

La carpeta `examples/` contiene siete programas que demuestran las funcionalidades del intérprete.

| Archivo | Qué demuestra |
|---|---|
| `variables.cpp` | Variables y tipos de datos |
| `ciclos.cpp` | Ciclos `for` y `while` |
| `funciones.cpp` | Funciones y cálculo factorial |
| `condicionales.cpp` | Condicionales y decisiones |
| `arreglos.cpp` | Declaración, acceso y modificación de arreglos |
| `recursion.cpp` | Funciones recursivas |
| `ambitos.cpp` | Diferencias entre variables globales y locales |

Para ejecutar cualquiera de ellos:

```bash
go run . examples/variables.cpp
go run . examples/ciclos.cpp
go run . examples/funciones.cpp
go run . examples/condicionales.cpp
go run . examples/arreglos.cpp
go run . examples/recursion.cpp
go run . examples/ambitos.cpp
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

El script ejecuta ocho programas válidos y tres programas inválidos, compara los resultados con los esperados y muestra cuáles pruebas fueron aprobadas.

En la última ejecución se obtuvo:

```text
Aprobadas: 11
Fallidas:  0
```

Los casos comprobados incluyen variables, operaciones, condicionales, ciclos, funciones, recursividad, ámbitos, arreglos y detección de errores.

### Compilación

Para comprobar que el proyecto compila correctamente:

```bash
go build ./...
```

## 9. Documento de diseño

El archivo `DISENO_CATCOMPILER.md` presenta la arquitectura del intérprete, una gramática EBNF resumida, las principales decisiones de implementación, las limitaciones del lenguaje y el uso de inteligencia artificial durante el desarrollo.

## 10. Alcance

CatCompiler es un proyecto académico cuyo propósito es demostrar las principales fases de un intérprete: análisis léxico, análisis sintáctico y evaluación.

No pretende implementar todas las características del estándar C++.