// Programa principal de CatCompiler: muestra la bienvenida e inicia el REPL.
package main

import (
	"fmt"
	"os"
	"os/user"
	"polaobesahermosa/repl"
)

func main() {
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
