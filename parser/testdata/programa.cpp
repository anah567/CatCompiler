#include <iostream>
using namespace std;

int factorial(int n) {
    if (n <= 1) { return 1; }
    return n * factorial(n - 1);
}

int main() {
    int edad = 20;
    double precio = 3.5;
    bool ok = edad >= 18 && precio < 10.0;
    string nombre = "Ana";
    int arr[3] = {1, 2, 3};
    // suma de los elementos
    int total = 0;
    for (int i = 0; i < 3; i++) {
        total += arr[i];
    }
    while (total > 0) { total--; }
    if (ok) { cout << "Hola " << nombre << " " << factorial(5) << "\n"; }
    else if (edad == 0) { cout << "cero" << endl; }
    else { cout << "no" << endl; }
    return 0;
}
