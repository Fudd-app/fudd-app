import type { Metadata } from "next";
import LegalDocument, { legalLinkClass, type LegalSection } from "@/components/legal/LegalDocument";
import { LEGAL } from "@/lib/legal";

export const metadata: Metadata = {
  title: "Política de Privacidad | Zelko",
  description: "Cómo Zelko recopila, usa, protege y comparte datos personales.",
  alternates: { canonical: "/privacidad" },
};

const sections: LegalSection[] = [
  {
    id: "responsable-y-alcance",
    title: "Responsable y alcance",
    content: (
      <>
        <p>
          Esta Política describe el tratamiento de datos personales realizado a través de {LEGAL.brand}, disponible en <a className={legalLinkClass} href={LEGAL.website}>{LEGAL.website}</a>, incluyendo el sitio institucional, el panel de administración, la carta digital, la gestión de pedidos y las integraciones de autenticación y pago.
        </p>
        <p>
          El servicio es operado bajo la marca <strong>{LEGAL.operator}</strong>, con actividad en {LEGAL.country}. El canal para consultas y ejercicio de derechos es <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a>. La identificación societaria, CUIT y domicilio legal del proveedor deberán constar además en la propuesta, orden de contratación o documentación fiscal aplicable antes de una contratación comercial.
        </p>
      </>
    ),
  },
  {
    id: "roles",
    title: "Quién decide sobre cada dato",
    content: (
      <>
        <p>
          Zelko actúa como responsable respecto de los datos necesarios para crear cuentas, autenticar usuarios, administrar la relación contractual, proteger la plataforma y brindar soporte.
        </p>
        <p>
          Cada comercio adherido decide qué información solicita a sus comensales, cómo gestiona pedidos, facturación, alergias, preferencias y atención al cliente. Para esos tratamientos, el comercio es el responsable y Zelko opera como proveedor tecnológico siguiendo sus instrucciones, salvo cuando la ley determine otra función.
        </p>
        <p>
          Mercado Pago, Google y otros terceros procesan datos bajo sus propias políticas para prestar sus servicios. Zelko no controla sus decisiones independientes de tratamiento.
        </p>
      </>
    ),
  },
  {
    id: "datos-recopilados",
    title: "Datos que podemos tratar",
    content: (
      <>
        <ul className="list-disc space-y-6 pl-20">
          <li><strong>Cuenta y equipo:</strong> nombre, correo, rol, negocio asociado, identificador interno, estado de acceso y, cuando corresponda, contraseña protegida mediante hash irreversible. Nunca almacenamos la contraseña en texto plano.</li>
          <li><strong>Identidad de Google:</strong> identificador estable de cuenta (<code>sub</code>), correo y confirmación de que Google verificó ese correo.</li>
          <li><strong>Negocio:</strong> nombre comercial, descripción, rubro, contactos, sucursales, horarios, logo, configuración visual y, cuando se carguen, datos fiscales.</li>
          <li><strong>Operación gastronómica:</strong> mesas, categorías, artículos, precios, variantes, disponibilidad, pedidos, cantidades, notas, estados y marcas de tiempo.</li>
          <li><strong>Comensales:</strong> alias opcional, identificador temporal de sesión y asociación a una mesa y cuenta. No se exige crear una cuenta personal para consultar una carta o pedir desde un QR.</li>
          <li><strong>Pagos:</strong> identificadores de preferencia u operación, monto y estado informado por Mercado Pago. Las credenciales de integración del comercio se tratan como información confidencial de configuración.</li>
          <li><strong>Datos técnicos:</strong> dirección IP, fecha y hora, navegador, dispositivo, registros de seguridad, errores y actividad imprescindible para operar y proteger el servicio.</li>
          <li><strong>Soporte:</strong> mensajes, solicitudes y documentación que la persona decida enviar.</li>
        </ul>
        <p>
          Solicitamos no incluir datos sensibles —por ejemplo, información de salud detallada, documentos, datos biométricos o financieros— en notas de pedidos o campos libres. Una indicación alimentaria debe limitarse a lo estrictamente necesario para que el comercio la atienda.
        </p>
      </>
    ),
  },
  {
    id: "fuentes",
    title: "De dónde provienen los datos",
    content: (
      <p>
        Los datos pueden ser proporcionados por la persona usuaria, por el administrador del comercio, generados durante el uso del servicio o recibidos de proveedores habilitados, como Google al iniciar sesión y Mercado Pago al informar el resultado de un pago. No adquirimos bases de datos personales para publicidad.
      </p>
    ),
  },
  {
    id: "finalidades",
    title: "Para qué usamos los datos",
    content: (
      <>
        <ul className="list-disc space-y-6 pl-20">
          <li>Crear y administrar cuentas, negocios, sucursales, cartas y permisos de equipo.</li>
          <li>Autenticar administradores mediante correo y contraseña de Zelko o Google, gestionar enlaces de invitación del equipo y prevenir accesos no autorizados.</li>
          <li>Recibir, preparar, entregar y cerrar pedidos; mostrar su estado en tiempo real.</li>
          <li>Facilitar pagos solicitados por el comercio a través de Mercado Pago.</li>
          <li>Producir métricas operativas del propio comercio, sin vender perfiles a terceros.</li>
          <li>Brindar soporte, resolver incidentes, mantener continuidad y mejorar funciones visibles del producto.</li>
          <li>Cumplir obligaciones legales, requerimientos de autoridad y defender derechos.</li>
        </ul>
        <p>
          El tratamiento se sustenta, según el caso, en el consentimiento informado, la ejecución de una relación contractual, el cumplimiento de obligaciones legales y la necesidad de operar y proteger el servicio respetando los derechos de las personas.
        </p>
      </>
    ),
  },
  {
    id: "google",
    title: "Datos obtenidos mediante Google",
    content: (
      <>
        <p>
          El acceso con Google se utiliza exclusivamente para confirmar identidad y permitir el ingreso de administradores o integrantes de equipo que ya estén registrados en Zelko. Se reciben únicamente el identificador estable de Google, el correo y su estado de verificación. No solicitamos acceso a Gmail, Drive, contactos, calendario ni otros contenidos de la cuenta.
        </p>
        <p>
          Esa información no se vende, no se utiliza para publicidad, evaluación crediticia, seguimiento ajeno al servicio ni entrenamiento de modelos de inteligencia artificial. Sólo se comparte con proveedores indispensables para operar o cuando exista obligación legal. El uso de información recibida de las API de Google cumple la <a className={legalLinkClass} href="https://developers.google.com/terms/api-services-user-data-policy" target="_blank" rel="noreferrer">Google API Services User Data Policy</a>, incluidos sus requisitos de uso limitado.
        </p>
        <p>
          Para solicitar la desvinculación de Google y supresión del identificador asociado, escribí a <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a>. La desvinculación puede impedir ese método de acceso, pero no elimina datos que deban conservarse por una obligación legal o para resolver controversias.
        </p>
      </>
    ),
  },
  {
    id: "cookies",
    title: "Cookies y almacenamiento local",
    content: (
      <>
        <p>
          Utilizamos tecnologías estrictamente necesarias para mantener sesiones, recordar el contexto de una mesa, conservar temporalmente un alias y proteger el acceso. Pueden incluir una cookie de sesión segura y almacenamiento local del navegador. No utilizamos estas tecnologías para publicidad comportamental ni vendemos identificadores de navegación.
        </p>
        <p>
          Google puede emplear sus propias tecnologías cuando se carga el botón de acceso; su tratamiento se rige por la política de Google. Bloquear almacenamiento esencial puede impedir el inicio de sesión, el seguimiento de pedidos o la continuidad de la cuenta de una mesa.
        </p>
      </>
    ),
  },
  {
    id: "destinatarios",
    title: "Con quién podemos compartir datos",
    content: (
      <>
        <p>No vendemos datos personales. Podemos comunicar únicamente lo necesario a:</p>
        <ul className="list-disc space-y-6 pl-20">
          <li>El comercio y su personal autorizado, para administrar su operación y atender pedidos.</li>
          <li>Proveedores de infraestructura y alojamiento, actualmente servicios desplegados en Render.</li>
          <li>Proveedores de correo transaccional configurados por Zelko, como Brevo, SMTP o Resend.</li>
          <li>Google, para la autenticación solicitada por la persona.</li>
          <li>Mercado Pago, cuando el comercio habilita y el comensal elige ese medio de pago.</li>
          <li>Asesores profesionales, auditores o proveedores de seguridad sujetos a confidencialidad.</li>
          <li>Autoridades competentes cuando una norma u orden válida lo exija.</li>
        </ul>
        <p>
          Ante una reorganización empresarial, sólo se transferirán los datos bajo deberes equivalentes de confidencialidad, información previa cuando corresponda y respeto por los derechos aplicables.
        </p>
      </>
    ),
  },
  {
    id: "transferencias",
    title: "Alojamiento y transferencias internacionales",
    content: (
      <>
        <p>
          La infraestructura puede procesar o almacenar información fuera de Argentina, incluida la región de Ohio, Estados Unidos. Cuando el destino no sea reconocido como adecuado por la autoridad argentina, deberán utilizarse las excepciones legales aplicables o salvaguardas como cláusulas contractuales modelo y controles técnicos y organizativos.
        </p>
        <p>
          Podés consultar los mecanismos reconocidos por la Agencia de Acceso a la Información Pública en su guía de <a className={legalLinkClass} href="https://www.argentina.gob.ar/transferencias-internacionales" target="_blank" rel="noreferrer">transferencias internacionales</a>.
        </p>
      </>
    ),
  },
  {
    id: "conservacion",
    title: "Plazos de conservación",
    content: (
      <>
        <p>
          Conservamos los datos mientras la cuenta o relación con el comercio permanezca activa y luego sólo durante el tiempo necesario para atender solicitudes, cumplir obligaciones legales, fiscales o contractuales, prevenir fraude y ejercer o defender derechos. Los enlaces mágicos expiran a los 15 minutos y la sesión de administración tiene una vigencia técnica máxima de 30 días, sin perjuicio de cierres anticipados por seguridad.
        </p>
        <p>
          Los pedidos y comprobantes pueden requerir conservación por parte del comercio conforme a sus obligaciones. Cuando ya no exista una finalidad legítima, los datos se eliminan, anonimizan o bloquean de forma segura. Las copias de respaldo pueden persistir por ciclos limitados antes de su sobrescritura.
        </p>
      </>
    ),
  },
  {
    id: "seguridad",
    title: "Seguridad e incidentes",
    content: (
      <>
        <p>
          Aplicamos medidas administrativas, técnicas y organizativas razonables según el riesgo, incluyendo control de acceso por rol y negocio, sesiones protegidas, enlaces de un solo uso, validación de identidad, comunicaciones cifradas en producción, registro de eventos y separación lógica entre comercios.
        </p>
        <p>
          Ningún sistema es infalible. Si se detecta un incidente que pueda generar un riesgo significativo, se documentará, contendrá y comunicará a las personas o autoridades cuando corresponda legalmente. Las vulnerabilidades pueden reportarse de forma responsable a <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a> sin divulgar datos de terceros.
        </p>
      </>
    ),
  },
  {
    id: "derechos",
    title: "Tus derechos y cómo ejercerlos",
    content: (
      <>
        <p>
          Podés solicitar información, acceso, rectificación, actualización, confidencialidad o supresión de tus datos y, cuando corresponda, retirar el consentimiento. El acceso es gratuito en los intervalos previstos por la Ley 25.326. Las solicitudes de acceso deben responderse dentro de 10 días corridos; las de rectificación, actualización o supresión, dentro de 5 días hábiles, salvo excepción legal.
        </p>
        <p>
          Enviá la solicitud desde el correo asociado —indicando nombre, relación con el negocio y derecho que querés ejercer— a <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a>. Podemos pedir información razonable para verificar identidad y evitar entregar datos a terceros. Si el dato está bajo control de un comercio, coordinaremos la solicitud con ese responsable.
        </p>
        <p>
          La Agencia de Acceso a la Información Pública, órgano de control de la Ley 25.326, recibe denuncias y reclamos. Consultá sus canales y modelos en <a className={legalLinkClass} href="https://www.argentina.gob.ar/aaip/datospersonales/derechos" target="_blank" rel="noreferrer">argentina.gob.ar/aaip</a>.
        </p>
      </>
    ),
  },
  {
    id: "menores",
    title: "Personas menores de edad",
    content: (
      <p>
        Las cuentas administrativas están destinadas a personas con capacidad legal y autoridad para representar al comercio. La carta pública puede ser consultada por menores bajo supervisión de una persona adulta o conforme a las reglas del establecimiento. No buscamos recopilar deliberadamente datos de menores ni datos sensibles. Si advertís un tratamiento indebido, contactanos para revisarlo y adoptar las medidas correspondientes.
      </p>
    ),
  },
  {
    id: "cambios-y-contacto",
    title: "Cambios, normativa y contacto",
    content: (
      <>
        <p>
          Podemos actualizar esta Política por cambios legales, técnicos o funcionales. La versión vigente indicará su fecha y, si el cambio es material, se informará por un medio razonable antes de aplicarlo. Cuando una nueva finalidad requiera consentimiento, se solicitará antes de iniciar ese tratamiento.
        </p>
        <p>
          Esta Política se interpreta conforme a la <a className={legalLinkClass} href="https://www.argentina.gob.ar/normativa/nacional/64790/actualizacion" target="_blank" rel="noreferrer">Ley 25.326</a>, su reglamentación y normas complementarias. Consultas: <a className={legalLinkClass} href={`mailto:${LEGAL.supportEmail}`}>{LEGAL.supportEmail}</a>.
        </p>
      </>
    ),
  },
];

export default function PrivacidadPage() {
  return (
    <LegalDocument
      eyebrow="Privacidad y protección de datos"
      title="Política de privacidad"
      description="Explicamos qué datos trata Zelko, para qué se utilizan, con quién pueden compartirse y cómo ejercer tus derechos."
      sections={sections}
      alternateHref="/terminos"
      alternateLabel="Ver términos"
    />
  );
}
