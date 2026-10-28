#!/bin/bash

# Ejecutamos todas las pruebas de CatCompiler
set -u

cd "$(dirname "$0")/.." || exit 1

aprobadas=0
fallidas=0

# Compilamos una sola vez para ejecutar las pruebas
binario=$(mktemp "${TMPDIR:-/tmp}/catcompiler.XXXXXX") || exit 1
trap 'rm -f "$binario"' EXIT

if ! go build -o "$binario" .; then
    echo "ERROR: No se pudo compilar CatCompiler"
    exit 1
fi

echo "================================"
echo "   PRUEBAS DE CATCOMPILER"
echo "================================"

# Probamos programas que deben ejecutarse correctamente
for archivo in tests/valid/*.cpp; do
    nombre=$(basename "$archivo" .cpp)
    esperado="tests/expected/$nombre.txt"
    salida=$(mktemp)
    
    if "$binario" "$archivo" > "$salida" 2>&1 &&
       diff -u "$esperado" "$salida"; then
        echo "PASS: $nombre"
        aprobadas=$((aprobadas + 1))
    else
        echo "FAIL: $nombre"
        cat "$salida"
        fallidas=$((fallidas + 1))
    fi

    rm -f "$salida"
done

# Probamos programas que deben producir errores
for archivo in tests/invalid/*.cpp; do
    nombre=$(basename "$archivo" .cpp)
    esperado="tests/expected/$nombre.txt"
    salida=$(mktemp)

    if "$binario" "$archivo" > "$salida" 2>&1; then
        echo "FAIL: $nombre (no produjo error)"
        fallidas=$((fallidas + 1))
    elif grep -Fq -f "$esperado" "$salida"; then
        echo "PASS: $nombre (error detectado)"
        aprobadas=$((aprobadas + 1))
    else
        echo "FAIL: $nombre (error diferente)"
        cat "$salida"
        fallidas=$((fallidas + 1))
    fi

    rm -f "$salida"
done

echo "================================"
echo "Aprobadas: $aprobadas"
echo "Fallidas:  $fallidas"
echo "================================"

if [ "$fallidas" -gt 0 ]; then
    exit 1
fi
