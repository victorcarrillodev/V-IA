---
name: systemic-issue-triage
description: Decidir si un fallo es puntual, sistémico o estructural antes de corregirlo. Úsala cuando el mismo tipo de error aparece en más de un sitio o tras varios parches.
license: "Apache-2.0"
metadata:
  author: "v-ia"
  version: "1.0"
---
# Triaje de fallos sistémicos

Antes de parchear, responde con evidencia (`ruta:línea`):

1. **¿Dónde más ocurre?** Busca el patrón (símbolos y TEXTO: al retirar un verbo, busca su cadena en todo el repo; el recuento por símbolo suele subestimar 2-5×).
2. **¿Cuál es la causa raíz?** Un contrato roto, una validación ausente en un límite compartido, un supuesto no documentado. Si no puedes nombrarla, aún no lo entiendes.
3. **Clasifica**: *Puntual* (1 sitio) → corrige en línea con una prueba. *Sistémico* (varios sitios, una causa) → UN arreglo en el punto compartido, cerrado contra pruebas con nombre, nunca contra promesas. *Estructural* (exige rediseño) → ficha de seguimiento y pregunta antes de ampliar el alcance. Si hay dos modos de fallo en un hilo, no lo cierres: comenta ambos.
4. **Prueba de sobreingeniería**: si el arreglo añade estado, verbo, flag, puerta o una representación paralela, rediséñalo; lo correcto suele BORRAR o RELAJAR.
5. **Presupuesto de bloqueo**: solo se bloquea al humano en consentimiento o una decisión terminal.
6. No ensanches el alcance en silencio: si descubres más sitios de los pedidos, informa. No ocultes el error (capturar y silenciar) ni dupliques el arreglo en cada sitio.
7. Informa por clase, con el delta neto de líneas.
