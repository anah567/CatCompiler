package repl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"polaobesahermosa/objects"
)

// TestRunFileCout comprueba que un archivo .cpp
// ejecuta main y escribe cout en la salida indicada.
func TestRunFileCout(t *testing.T) {

	// Creamos un programa C++ para la prueba.
	code := `
int main() {
    int numero = 10;
    int resultado = numero * 2;

    cout << "El resultado es: " << resultado << endl;

    return 0;
}
`

	// Creamos un archivo temporal para no modificar
	// los ejemplos originales del proyecto.
	path := filepath.Join(t.TempDir(), "prueba.cpp")

	if err := os.WriteFile(path, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}

	// Creamos un buffer para guardar lo que imprime cout.
	var output bytes.Buffer

	// Ejecutamos el archivo con un entorno nuevo.
	env := objects.NewEnvironment()

	err := RunFile(path, &output, env)
	if err != nil {
		t.Fatalf("error al ejecutar el archivo: %v", err)
	}

	// Comprobamos que cout haya producido el texto esperado.
	expected := "El resultado es: 20\n"

	if output.String() != expected {
		t.Fatalf(
			"se esperaba %q, se obtuvo %q",
			expected,
			output.String(),
		)
	}
}

// TestRunFileExtension comprueba que solo se acepten
// archivos con extensión .cpp.
func TestRunFileExtension(t *testing.T) {

	var output bytes.Buffer
	env := objects.NewEnvironment()

	err := RunFile("programa.txt", &output, env)

	if err == nil {
		t.Fatal("se esperaba un error por extensión incorrecta")
	}

	if !strings.Contains(err.Error(), ".cpp") {
		t.Fatalf("mensaje de error inesperado: %v", err)
	}
}

// TestRunFileProgramasCompletos comprueba que diferentes
// programas C++ funcionan desde el archivo hasta cout.
func TestRunFileProgramasCompletos(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name: "funcion con parametros",
			code: `
int sumar(int a, int b) {
    return a + b;
}

int main() {
    cout << sumar(5, 7) << endl;
    return 0;
}
`,
			expected: "12\n",
		},
		{
			name: "funcion recursiva",
			code: `
int factorial(int n) {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}

int main() {
    cout << factorial(5) << endl;
    return 0;
}
`,
			expected: "120\n",
		},
		{
			name: "ciclo for",
			code: `
int main() {
    int suma = 0;

    for (int i = 1; i <= 5; i++) {
        suma += i;
    }

    cout << suma << endl;
    return 0;
}
`,
			expected: "15\n",
		},
		{
			name: "arreglos",
			code: `
int main() {
    int numeros[3] = {10, 20, 30};

    cout << numeros[0] + numeros[1] + numeros[2] << endl;
    return 0;
}
`,
			expected: "60\n",
		},
		{
			name: "condicional",
			code: `
int main() {
    int edad = 20;

    if (edad >= 18) {
        cout << "Mayor de edad" << endl;
    } else {
        cout << "Menor de edad" << endl;
    }

    return 0;
}
`,
			expected: "Mayor de edad\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Creamos un archivo temporal para cada programa.
			path := filepath.Join(t.TempDir(), "programa.cpp")

			if err := os.WriteFile(path, []byte(tt.code), 0600); err != nil {
				t.Fatal(err)
			}

			// Capturamos la salida de cout.
			var output bytes.Buffer
			env := objects.NewEnvironment()

			// Ejecutamos el programa completo.
			if err := RunFile(path, &output, env); err != nil {
				t.Fatalf("error al ejecutar: %v", err)
			}

			// Comparamos la salida con el resultado esperado.
			if output.String() != tt.expected {
				t.Fatalf(
					"se esperaba %q, se obtuvo %q",
					tt.expected,
					output.String(),
				)
			}
		})
	}
}

// TestRunFileErrores comprueba que el intérprete
// detecta errores de sintaxis y de ejecución.
func TestRunFileErrores(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedError string
	}{
		{
			name: "error de sintaxis",
			code: `
int main() {
    int numero = 10
    return 0;
}
`,
			expectedError: "errores de sintaxis",
		},
		{
			name: "variable no declarada",
			code: `
int main() {
    cout << desconocida << endl;
    return 0;
}
`,
			expectedError: "desconocida",
		},
		{
			name: "cantidad incorrecta de argumentos",
			code: `
int sumar(int a, int b) {
    return a + b;
}

int main() {
    cout << sumar(5) << endl;
    return 0;
}
`,
			expectedError: "cantidad de argumentos incorrecta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Creamos un archivo temporal con el programa.
			path := filepath.Join(t.TempDir(), "error.cpp")

			if err := os.WriteFile(path, []byte(tt.code), 0600); err != nil {
				t.Fatal(err)
			}

			// Ejecutamos el archivo y capturamos su salida.
			var output bytes.Buffer
			env := objects.NewEnvironment()

			err := RunFile(path, &output, env)

			// En estos casos debe producirse un error.
			if err == nil {
				t.Fatal("se esperaba un error, pero el programa terminó correctamente")
			}

			// Comprobamos que el error corresponda al problema.
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Fatalf(
					"se esperaba un error con %q, se obtuvo %q",
					tt.expectedError,
					err.Error(),
				)
			}
		})
	}
}
