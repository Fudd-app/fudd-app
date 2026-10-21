import type { Metadata } from "next";
import LegalDocument, { legalLinkClass, type LegalSection } from "@/components/legal/LegalDocument";
import { LEGAL } from "@/lib/legal";
import { formatearPrecioPlan, PLAN_MONEDA, PLAN_PRUEBA_MESES } from "@/lib/planes";

export const metadata: Metadata = {
  title: "Términos del Servicio | Zelko",
  description: "Condiciones de uso de la plataforma Zelko para comercios y comensales.",
  alternates: { canonical: "/terminos" },
};

const sections: LegalSection[] = [
  {
    id: "identificacion-y-aceptacion",
    title: "Identificación, alcance y aceptación",
    content: (
      <>
        <p>
          Estos Términos regulan el acceso y uso de {LEGAL.brand}, disponible en{" "}
          <a className={legalLinkClass} href={LEGAL.website}>{LEGAL.website}</a>, por parte de comercios,
          integrantes de sus equipos y comensales. El servicio es operado bajo la marca <strong>{LEGAL.operator}</strong>
          {" "}en {LEGAL.country}. Consultas: <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a>.
        </p>
        <p>
          Al crear un negocio, aceptar expresamente estos Términos o usar funciones que requieren una cuenta,
          confirmás que los leíste y que tenés capacidad y facultades suficientes para obligarte o representar al
          comercio. La simple consulta de una carta pública no implica contratar un plan de Zelko.
        </p>
        <p>
          Si actuás como consumidor, conservás todos los derechos irrenunciables que reconozcan las normas de
          defensa del consumidor. Ninguna disposición de este documento los limita.
        </p>
      </>
    ),
  },
  {
    id: "roles",
    title: "Qué hace Zelko y qué hace el comercio",
    content: (
      <>
        <p>
          Zelko provee herramientas para publicar una carta digital, administrar sucursales y mesas, recibir
          pedidos, organizar cocina, mostrar estados, obtener métricas y conectar servicios de terceros. El comercio
          conserva la dirección y responsabilidad de su actividad gastronómica.
        </p>
        <p>
          El comercio es quien ofrece y vende sus productos, fija precios, impuestos, stock, ingredientes,
          disponibilidad y condiciones de atención; confirma, prepara, entrega, factura, cancela y reintegra pedidos;
          y debe cumplir las normas sanitarias, bromatológicas, fiscales, laborales y de defensa del consumidor que le
          correspondan. Zelko no cocina, manipula ni entrega alimentos y no garantiza su composición o aptitud.
        </p>
      </>
    ),
  },
  {
    id: "cuentas-y-seguridad",
    title: "Cuentas, accesos y seguridad",
    content: (
      <>
        <p>
          Los accesos administrativos son personales. Debés usar información exacta, mantener actualizados los datos
          y asignar a cada integrante sólo el rol necesario. No está permitido compartir enlaces mágicos, sesiones o
          credenciales, eludir controles ni acceder a otro negocio sin autorización.
        </p>
        <p>
          El ingreso administrativo puede realizarse mediante correo y contraseña de Zelko o mediante Google. Los
          enlaces de un solo uso pueden utilizarse para invitaciones de equipo. Google verifica la identidad según sus
          propias condiciones; Zelko no solicita la contraseña de Google. Debés avisar de inmediato a{" "}
          <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a> ante un acceso
          sospechoso, pérdida de control del correo o cambio no autorizado.
        </p>
      </>
    ),
  },
  {
    id: "pedidos",
    title: "Carta, pedidos y relación con el comensal",
    content: (
      <>
        <p>
          La información de la carta proviene del comercio. Un pedido enviado mediante la plataforma es una solicitud
          dirigida al comercio y queda sujeto a su confirmación, disponibilidad real y reglas informadas al comensal.
          El comercio debe corregir errores de precio o descripción antes de aceptar y comunicar cualquier sustitución.
        </p>
        <p>
          Reclamos sobre calidad, alergias, demoras, entrega, facturación, cancelación o devolución deben ser atendidos
          por el comercio, sin perjuicio del soporte técnico que Zelko pueda brindar. Las notas del pedido no
          reemplazan la conversación directa con el establecimiento ante una alergia o riesgo de salud.
        </p>
      </>
    ),
  },
  {
    id: "pagos",
    title: "Pagos con Mercado Pago",
    content: (
      <>
        <p>
          Cuando el comercio habilita Mercado Pago, el pago se procesa en el entorno de ese proveedor y se rige también
          por sus términos y políticas. El comercio es el vendedor y titular de su cuenta de cobro. Zelko recibe
          identificadores, importes y estados necesarios para reflejar la operación, pero no almacena números completos
          de tarjeta ni códigos de seguridad.
        </p>
        <p>
          La aprobación, rechazo, contracargo, acreditación o devolución depende de Mercado Pago, del medio de pago y
          del comercio. Un estado técnico no reemplaza el comprobante del proveedor. El comercio debe proteger sus
          credenciales de integración y revocarlas si sospecha una exposición.
        </p>
      </>
    ),
  },
  {
    id: "planes-y-contratacion",
    title: "Planes, precios y contratación",
    content: (
      <>
        <p>
          Cada negocio nuevo recibe acceso al plan Pro sin cargo durante {PLAN_PRUEBA_MESES} meses desde su creación. Al terminar la
          prueba, puede continuar con el plan Free sujeto a sus límites o contratar el plan Pro por {formatearPrecioPlan()} {PLAN_MONEDA} mensuales.
          Los límites visibles del plan —por ejemplo, cantidad de mesas, productos o sucursales— pueden impedir nuevas
          altas sin eliminar los datos ya existentes.
        </p>
        <p>
          No se efectuará un cobro nuevo ni una renovación paga sin una base contractual válida e información previa.
          Cuando corresponda el derecho legal de revocación de una contratación a distancia, podrá ejercerse dentro del
          plazo y por los canales exigidos por la normativa aplicable. Una condición comercial específica prevalece
          sobre estos Términos sólo si fue aceptada por escrito y no reduce derechos imperativos.
        </p>
      </>
    ),
  },
  {
    id: "uso-aceptable",
    title: "Uso aceptable",
    content: (
      <>
        <p>Queda prohibido usar el servicio para:</p>
        <ul className="list-disc space-y-6 pl-20">
          <li>Violar leyes, derechos de terceros, medidas sanitarias o reglas de medios de pago.</li>
          <li>Publicar información engañosa, ilícita, discriminatoria, dañina o que infrinja propiedad intelectual.</li>
          <li>Introducir malware, automatizar accesos abusivos, extraer datos masivamente o afectar la disponibilidad.</li>
          <li>Suplantar identidades, alterar pedidos o pagos, probar credenciales ajenas o evadir límites del plan.</li>
          <li>Recolectar datos sensibles innecesarios o utilizar datos de comensales para fines incompatibles.</li>
        </ul>
        <p>
          Podemos limitar una función o suspender un acceso ante un riesgo concreto y documentado, procurando aviso y
          oportunidad de subsanar cuando ello no comprometa la seguridad, a terceros o una investigación.
        </p>
      </>
    ),
  },
  {
    id: "contenido",
    title: "Contenido del comercio y propiedad intelectual",
    content: (
      <>
        <p>
          El comercio conserva los derechos sobre sus marcas, imágenes, textos, cartas y datos. Declara contar con las
          autorizaciones necesarias y otorga a Zelko una licencia limitada, no exclusiva y revocable para alojar,
          reproducir y mostrar ese contenido únicamente con el fin de prestar el servicio.
        </p>
        <p>
          El software, diseño, marca y documentación de Zelko pertenecen a sus titulares y no se transfieren por
          el uso del servicio. No se permite copiar, revender, descompilar o crear un servicio derivado salvo autorización
          expresa o excepción legal.
        </p>
      </>
    ),
  },
  {
    id: "datos-personales",
    title: "Datos personales y confidencialidad",
    content: (
      <>
        <p>
          El tratamiento de datos se describe en la{" "}
          <a className={legalLinkClass} href="/privacidad">Política de Privacidad</a>. Cada comercio debe informar a sus
          comensales, tratar sólo los datos necesarios, respetar sus derechos y limitar el acceso de su equipo. Zelko
          aplicará controles razonables y usará los datos operativos del comercio para prestar, proteger y mantener el
          servicio, no para venderlos.
        </p>
        <p>
          Las partes deben mantener en reserva credenciales, información no pública, secretos comerciales y datos
          personales, excepto cuando su comunicación sea necesaria para el servicio, esté autorizada o sea exigida por ley.
        </p>
        <p>
          Cuando Zelko trate datos de comensales por cuenta del comercio, lo hará sólo para prestar y proteger el
          servicio conforme a la configuración e instrucciones documentadas del comercio; limitará el acceso a personal
          y proveedores sujetos a confidencialidad; aplicará medidas razonables de seguridad; colaborará con solicitudes
          de titulares e incidentes; e informará las categorías de subencargados en la Política de Privacidad. Al finalizar,
          eliminará, anonimizará o devolverá los datos según corresponda, salvo conservación legal. El comercio garantiza
          la licitud de sus instrucciones y sigue siendo responsable de la información que decide recopilar y utilizar.
        </p>
      </>
    ),
  },
  {
    id: "terceros",
    title: "Servicios y enlaces de terceros",
    content: (
      <p>
        Google, Mercado Pago, proveedores de correo, alojamiento y otros servicios integrados tienen disponibilidad,
        seguridad y condiciones propias. Zelko procura seleccionar y configurar proveedores adecuados, pero no
        controla sus sistemas. Un enlace externo no implica recomendación ni responsabilidad sobre contenidos ajenos.
      </p>
    ),
  },
  {
    id: "disponibilidad",
    title: "Disponibilidad, mantenimiento y cambios funcionales",
    content: (
      <>
        <p>
          Trabajamos para mantener el servicio disponible, pero pueden existir mantenimientos, fallos de red, incidentes
          de terceros o causas de fuerza mayor. Cuando sea razonable, se informarán interrupciones programadas y se
          intentará restaurar el servicio con prioridad acorde al impacto.
        </p>
        <p>
          Podemos modificar funciones para mejorar seguridad, cumplir la ley o evolucionar el producto. No eliminaremos
          arbitrariamente una prestación esencial ya pagada durante su período vigente; si un cambio material la afecta,
          se comunicará y se ofrecerá la solución que corresponda según el contrato y la ley.
        </p>
      </>
    ),
  },
  {
    id: "responsabilidad",
    title: "Garantías y responsabilidad",
    content: (
      <>
        <p>
          Zelko responde por sus obligaciones conforme a la ley. No garantiza ventas, demanda, resultados comerciales,
          ausencia absoluta de interrupciones ni la exactitud del contenido cargado por comercios o terceros. El usuario
          debe mantener procedimientos alternativos razonables para operaciones críticas.
        </p>
        <p>
          En relaciones exclusivamente empresariales y sólo cuando la ley lo permita, la responsabilidad por daños directos
          comprobados derivados de un mismo hecho se limita al importe efectivamente abonado por el servicio durante los
          doce meses anteriores. Esa limitación no se aplica a dolo o culpa grave, lesiones, deberes de confidencialidad o
          protección de datos, obligaciones de pago ni derechos irrenunciables de consumidores.
        </p>
        <p>
          No se excluye ni limita una responsabilidad que legalmente no pueda excluirse. Cada parte debe adoptar medidas
          razonables para evitar o reducir el daño y comunicar el incidente sin demora injustificada.
        </p>
      </>
    ),
  },
  {
    id: "suspension-y-baja",
    title: "Suspensión, baja y datos al finalizar",
    content: (
      <>
        <p>
          El comercio puede solicitar la baja por el canal contractual o a{" "}
          <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a>. Antes de finalizar,
          debe descargar la información que necesite y resolver pedidos, cobros y obligaciones pendientes. Podemos suspender
          por incumplimiento grave, fraude, riesgo de seguridad, falta de pago o mandato legal, aplicando proporcionalidad.
        </p>
        <p>
          Tras la baja, los accesos cesan y los datos se eliminan, anonimizan o bloquean de acuerdo con la Política de
          Privacidad, los plazos informados y las obligaciones legales. La baja de Zelko no cancela automáticamente
          contratos o credenciales mantenidos directamente con Google, Mercado Pago u otros proveedores.
        </p>
      </>
    ),
  },
  {
    id: "modificaciones",
    title: "Modificaciones y comunicaciones",
    content: (
      <>
        <p>
          La versión vigente muestra su fecha. Los cambios materiales se comunicarán con antelación razonable por correo,
          dentro del panel o en el sitio. Si una modificación requiere nueva aceptación, no se aplicará por silencio en los
          casos en que la ley exija consentimiento expreso.
        </p>
        <p>
          Las comunicaciones operativas y legales se enviarán al correo registrado. Es responsabilidad del comercio
          mantenerlo vigente. Las notificaciones a Zelko deben dirigirse a {" "}
          <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a>.
        </p>
      </>
    ),
  },
  {
    id: "ley-y-jurisdiccion",
    title: "Ley aplicable, jurisdicción y disposiciones finales",
    content: (
      <>
        <p>
          Estos Términos se rigen por las leyes de la República Argentina. Toda controversia procurará resolverse primero
          de buena fe. En relaciones empresariales, y salvo acuerdo específico válido, serán competentes los tribunales que
          correspondan al domicilio del proveedor. Para consumidores prevalecerán su domicilio, las opciones y las
          autoridades administrativas reconocidas por las normas imperativas.
        </p>
        <p>
          Si una cláusula es inválida, se interpretará o separará en la mínima medida necesaria sin afectar las restantes.
          La falta de ejercicio de un derecho no implica renuncia. Estos Términos, la Política de Privacidad y la condición
          comercial aceptada integran el acuerdo aplicable, sin desplazar obligaciones legales.
        </p>
        <p>
          Referencias oficiales: <a className={legalLinkClass} href="https://www.argentina.gob.ar/normativa/nacional/ley-24240-638/actualizacion" target="_blank" rel="noreferrer">Ley 24.240 de Defensa del Consumidor</a> y{" "}
          <a className={legalLinkClass} href="https://www.argentina.gob.ar/normativa/nacional/ley-26994-235975/actualizacion" target="_blank" rel="noreferrer">Código Civil y Comercial de la Nación</a>.
        </p>
      </>
    ),
  },
];

export default function TerminosPage() {
  return (
    <LegalDocument
      eyebrow="Condiciones de uso"
      title="Términos del servicio"
      description="Reglas claras para comercios, equipos y comensales que utilizan Zelko."
      sections={sections}
      alternateHref="/privacidad"
      alternateLabel="Ver privacidad"
    />
  );
}
