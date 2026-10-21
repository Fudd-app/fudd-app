# Checklist legal y de privacidad previo a producción

Este documento es operativo e interno. Las páginas públicas reducen ambigüedad, pero no sustituyen la revisión de un profesional ni los controles técnicos y contractuales.

## Bloqueantes antes de aceptar clientes pagos

- [ ] Completar en los Términos y la Política la razón social o nombre completo del titular, CUIT y domicilio legal/constituido. No publicar datos inventados.
- [ ] Confirmar que `supportmesaclick@gmail.com` existe, recibe mensajes y tiene responsables y tiempos de respuesta definidos.
- [ ] Obtener revisión de un abogado argentino con experiencia en protección de datos, comercio electrónico, consumo y SaaS B2B.
- [ ] Determinar y cumplir la inscripción o documentación exigible para las bases de datos personales ante la AAIP.
- [ ] Inventariar encargados y subencargados reales: Render, proveedor de correo, Google, Mercado Pago, monitoreo y cualquier analítica.
- [ ] Formalizar acuerdos de tratamiento y salvaguardas para transferencias internacionales, incluidas cláusulas contractuales modelo cuando correspondan.
- [ ] Definir plazos concretos por categoría de dato y automatizar eliminación, anonimización y vencimiento de respaldos.
- [ ] Probar el circuito de acceso, rectificación, supresión, exportación y oposición, con verificación de identidad y registro de plazos.
- [ ] Mantener evidencias de versión y aceptación de los Términos y la Política (usuario, fecha, versión, IP o evento equivalente).
- [ ] Realizar auditoría de cookies, almacenamiento local, SDK y telemetría cada vez que se agregue un proveedor.

## Seguridad y operación

- [ ] Guardar secretos de Mercado Pago y proveedores sólo en un gestor de secretos; cifrar credenciales persistidas y documentar rotación y revocación.
- [ ] Verificar aislamiento por negocio, permisos por rol, expiración/revocación de sesiones, rate limiting y protección CSRF/XSS/SQLi.
- [ ] Definir respuesta a incidentes, responsables, preservación de evidencia, evaluación de riesgo y comunicaciones exigibles.
- [ ] Hacer pruebas de restauración de backups y documentar continuidad para pedidos activos.
- [ ] Capacitar al personal y comercios para no volcar datos sensibles en notas libres.

## Google OAuth y publicación

- [ ] Verificar el dominio en Google Search Console con una cuenta autorizada del proyecto OAuth.
- [ ] Configurar exactamente la página principal, `/privacidad` y `/terminos` en OAuth Consent Screen.
- [ ] Mantener los enlaces legales visibles desde la página principal y usar sólo los scopes mínimos.
- [ ] Confirmar que el texto sobre Google coincide con los datos y scopes observados en producción.

## Comercio electrónico

- [ ] Publicar antes de contratar precio final, impuestos, moneda, frecuencia, renovación, límites y proceso de baja de cada plan.
- [ ] Implementar el mecanismo de revocación/baja exigible y conservar comprobante de la solicitud.
- [ ] Entregar o poner a disposición la versión aceptada del contrato de adhesión y no incluir cláusulas abusivas.
- [ ] Alinear facturación, identificación del proveedor y jurisdicción con la estructura legal efectivamente usada.
