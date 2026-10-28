# CatCompiler

CatCompiler es un intérprete educativo desarrollado en Go que permite analizar y ejecutar un subconjunto del lenguaje C++.

El proyecto incluye un analizador léxico (Lexer), un analizador sintáctico (Parser), un árbol de sintaxis abstracta (AST), un evaluador (Evaluator) y una interfaz interactiva de consola (REPL).

## 1. Requisitos

- Go, en una versión compatible con `go.mod`.
- Una terminal para ejecutar los comandos.
- No se requieren dependencias externas adicionales para las funcionalidades descritas.

## 2. Ejecución

Desde la carpeta principal del proyecto:

```bash
go run .
```

Este comando inicia el REPL de CatCompiler, que permite seleccionar entre tres modos:

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

Entre las funcionalidades implementadas y comprobadas se encuentran:

- Declaraciones de variables tipadas.
- Tipos `int`, `double`, `bool` y `string`.
- Operaciones aritméticas y comparaciones.
- Operadores lógicos con evaluación de cortocircuito.
- Condicionales `if`, `else if` y `else`.
- Ciclos `for` y `while`.
- Funciones con parámetros y tipos de retorno.
- Funciones recursivas.
- Arreglos e indexación.
- Operadores de asignación e incremento.
- Salida por pantalla mediante `cout` y `endl`.
- Detección de errores de sintaxis y de ejecución.
- Ejecución automática de `main()` al cargar archivos `.cpp`.

El proyecto implementa un subconjunto educativo de C++, no el lenguaje C++ completo.

## 6. Ejemplos

La carpeta `examples/` contiene programas para demostrar las funcionalidades del intérprete.

### Variables

```bash
go run . examples/variables.cpp
```

Muestra los valores de variables de diferentes tipos.

### Ciclos

```bash
go run . examples/ciclos.cpp
```

Salida:

```text
Suma: 15
Contador: 0
Contador: 1
Contador: 2
```

### Funciones

```bash
go run . examples/funciones.cpp
```

Salida:

```text
Factorial de 5: 120
```

## 7. Estructura del proyecto

| Ruta | Responsabilidad |
|---|---|
| `main.go` | Inicia el REPL o ejecuta archivos `.cpp` |
| `token/` | Define los tipos de tokens |
| `lexer/` | Analiza el código fuente y genera tokens |
| `parser/` | Construye el AST |
| `ast/` | Define los nodos del árbol sintáctico |
| `objects/` | Define los objetos de ejecución y los entornos |
| `evaluator/` | Ejecuta las instrucciones representadas en el AST |
| `repl/` | Implementa los modos interactivos y la ejecución de archivos |
| `examples/` | Contiene programas `.cpp` de demostración |
| `docs/` | Contiene documentación técnica y la gramática del proyecto |

## 8. Pruebas

Para ejecutar todas las pruebas automáticas:

```bash
go test ./...
```

Para ver los resultados detallados de las pruebas de integración:

```bash
go test ./repl -v
```

Las pruebas verifican funcionalidades del lexer, parser y evaluador, además de la ejecución de programas completos.

Entre los casos de integración comprobados están:

- Ejecución de `main()`.
- Captura de la salida de `cout`.
- Funciones con parámetros.
- Recursividad.
- Ciclos `for`.
- Arreglos.
- Condicionales.
- Errores de sintaxis.
- Variables no declaradas.
- Cantidad incorrecta de argumentos.
- Validación de archivos `.cpp`.

Para compilar los paquetes:

```bash
go build ./...
```

## 9. Documentación adicional

La carpeta `docs/` contiene documentación sobre el AST, decisiones de implementación, la gramática del lenguaje y material de estudio.

## 10. Alcance

CatCompiler es un proyecto académico cuyo propósito es demostrar las fases principales de un intérprete: análisis léxico, análisis sintáctico y evaluación.

No pretende implementar todas las características del estándar C++.