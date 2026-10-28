int numero = 10;

int calcular() {
    int numero = 20;
    return numero;
}

int main() {
    cout << "Local: " << calcular() << endl;
    cout << "Global: " << numero << endl;
    return 0;
}
