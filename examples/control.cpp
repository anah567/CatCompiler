
int main() {
    // Condicionales con else if
    int nota = 4;

    if (nota > 4) {
        cout << "Excelente" << endl;
    } else if (nota >= 3) {
        cout << "Aprobado" << endl;
    } else {
        cout << "Reprobado" << endl;
    }

    // Ciclo do-while
    int i = 0;
    do {
        i++;
    } while (i < 3);

    cout << "do-while: " << i << endl;

    // Ciclo for con continue y break
    for (int j = 0; j < 10; j++) {
        if (j == 1) {
            continue;
        }

        if (j == 4) {
            break;
        }

        cout << "j = " << j << endl;
    }

    /* Evaluacion en cortocircuito */
    cout << "corto: " << (true || 1 / 0 == 0) << endl;

    // Operadores de asignacion y modulo
    int n = 10;
    n *= 3;

    cout << "n % 7 = " << n % 7 << endl;

    return 0;
}
