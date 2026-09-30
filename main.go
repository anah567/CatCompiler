// Programa principal de CatCompiler: muestra la bienvenida e inicia el REPL.
package main

import (
	"fmt"
	"os"
	"os/user"
	"polaobesahermosa/objects"
	"polaobesahermosa/repl"
)

func main() {
	// Con una ruta .ssg se ejecuta el archivo completo y se termina. Se usa
	// un solo entorno para todas sus sentencias, igual que en el REPL.
	if len(os.Args) > 1 {
		if len(os.Args) != 2 {
			fmt.Fprintln(os.Stderr, "uso: go run . [archivo.ssg]")
			os.Exit(1)
		}
		if err := repl.RunFile(os.Args[1], os.Stdout, objects.NewEnvironment()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// Se obtiene el usuario del sistema para saludarlo por su nombre.
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	fmt.Printf("********************************************\n")
	fmt.Printf("**                                        **\n")
	fmt.Printf("**  -- BIENVENIDO A CATCOMPILER REPL --   **\n")
	fmt.Printf("**                                        **\n")
	fmt.Printf("**      /\\_/\\     ¡Miau! Soy Pola,      **\n")
	fmt.Printf("**     ( o.o )    tu asistente felino.    **\n")
	fmt.Printf("**      > ^ <     ¡Dame esos tokens!      **\n")
	fmt.Printf("**                                        **\n")
	fmt.Printf("********************************************\n\n")
	fmt.Printf("¡Hola %s! Estás en CatCompiler.\n",
		user.Username)
	fmt.Printf("Escribe algún comando y deja que la magia ocurra...\n")

	// El REPL lee de la entrada estándar y escribe en la salida estándar.
	repl.Start(os.Stdin, os.Stdout)
}
