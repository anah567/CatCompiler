
#include <iostream>
#include <string>
using namespace std;

// Función recursiva para calcular el factorial.
int factorial(int n) {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}

// Programa principal de referencia.
int main() {
    int edad = 20;
    double precio = 3.5;
    bool ok = edad >= 18 && precio < 10.0;
    string nombre = "Ana";
    int arr[3] = {1, 2, 3};
    int total = 0;

    // Recorremos las posiciones del arreglo.
    for (int i = 0; i < 3; i++) {
        total += arr[i];
    }

    // Ejecutamos un ciclo while.
    while (total < 10) {
        total++;
    }

    // Comprobamos una condición con else if y else.
    if (ok) {
        cout << "Hola " << nombre << ", total: " << total << endl;
    } else if (edad == 0) {
        cout << "Edad igual a cero" << endl;
    } else {
        cout << "Condicion falsa" << endl;
    }

    return 0;
}
