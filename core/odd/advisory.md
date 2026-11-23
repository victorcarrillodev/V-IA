
En este agente NADA te bloquea: cumple estas reglas por tu cuenta. Si tienes terminal y `v-ia` está en PATH, úsalo como control:
- Antes de una consulta o propuesta: `v-ia guard start`. Al terminar: `v-ia guard check` debe decir "sin cambios".
- En Seguimiento: `v-ia task new <feature> --request "<petición literal>"`; antes de cerrar, `v-ia task check`.
- Antes de reportar terminado o entregar: `v-ia review check` (build/tests, riesgo y recibo). Un recibo no autoriza commit ni push.
Si `v-ia` no está disponible, aplica igualmente las reglas a mano.
