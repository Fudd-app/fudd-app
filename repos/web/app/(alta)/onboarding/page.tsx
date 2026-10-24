"use client";
/* eslint-disable @next/next/no-img-element -- imagen de referencia de la maqueta (Unsplash) */

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, getErrorMessage } from "@/lib/api";
import LandingIcon, { type LandingIconName } from "@/components/landing/LandingIcon";
import RegistroBrand from "@/components/registro/RegistroBrand";
import { useRegistro } from "@/components/registro/RegistroContext";
import { emailValido, esErrorDeConexion, esSlugReservado, generarSlug, limpiarSlugEscrito, SLUG_RE } from "@/components/registro/validaciones";
import { formatearPrecioPlan, PLAN_MONEDA, PLAN_PRUEBA_MESES } from "@/lib/planes";

const TOTAL_PASOS = 3;
// El alta inicial crea hasta 10 mesas en lote; durante la prueba Pro se pueden sumar más desde el panel.
const MAX_MESAS_INICIALES = 10;
const CAPACIDAD_MESA = 4;

const RUBROS: { label: string; valor: string; icon: LandingIconName }[] = [
  { label: "Restaurante", valor: "restaurante", icon: "brand" },
  { label: "Cafetería", valor: "cafeteria", icon: "clock" },
  { label: "Bar o cervecería", valor: "bar", icon: "card" },
  { label: "Otro", valor: "otro", icon: "more" },
];

const PASOS = [
  ["Negocio", "Nombre y tipo de local"],
  ["Primera sucursal", "Datos básicos del local"],
  ["Todo listo", "Revisá y empezá"],
];

const DIAS = ["lunes", "martes", "miercoles", "jueves", "viernes", "sabado", "domingo"];
const HORARIO_ESTANDAR = Object.fromEntries(
  DIAS.map((dia) => [dia, { abierto: true, tramos: [{ apertura: "12:00", cierre: "00:00" }] }]),
);

type EstadoSlug = "vacio" | "invalido" | "verificando" | "disponible" | "en-uso" | "sin-verificar";

function normalizarWhatsapp(value: string) {
  const sinInvalidos = value.replace(/[^\d+ ]/g, "");
  const soloUnPrefijo = sinInvalidos.replace(/\+/g, (signo, offset) => (offset === 0 ? signo : ""));
  return soloUnPrefijo.replace(/ {2,}/g, " ").slice(0, 25);
}

export default function OnboardingPage() {
  const router = useRouter();
  const { cuenta, setCuenta } = useRegistro();

  const [paso, setPaso] = useState(1);
  const [enviado, setEnviado] = useState(false);
  const [nombreNegocio, setNombreNegocio] = useState("");
  const [rubro, setRubro] = useState(RUBROS[0].valor);
  const [rubroDetalle, setRubroDetalle] = useState("");
  const [slugManual, setSlugManual] = useState<string | null>(null);
  const [editandoSlug, setEditandoSlug] = useState(false);
  const [resultadoSlug, setResultadoSlug] = useState<{ slug: string; estado: EstadoSlug } | null>(null);
  const [nombreSucursal, setNombreSucursal] = useState("Casa central");
  const [whatsapp, setWhatsapp] = useState("");
  const [emailSucursal, setEmailSucursal] = useState("");
  const [mesas, setMesas] = useState(MAX_MESAS_INICIALES);
  const [creando, setCreando] = useState("");
  const [magicLinkEnviado, setMagicLinkEnviado] = useState<{ email: string; devLink?: string } | null>(null);
  const [error, setError] = useState("");
  const [negocioCreado, setNegocioCreado] = useState(false);

  // Sin datos de cuenta (recarga o acceso directo) no se puede crear el negocio.
  useEffect(() => {
    if (!cuenta) router.replace("/registro");
  }, [cuenta, router]);

  const slug = slugManual ?? generarSlug(nombreNegocio);
  const slugValido = SLUG_RE.test(slug);

  // Verifica el nombre en URL con una pequeña espera mientras se escribe.
  useEffect(() => {
    // Los slugs reservados se resuelven en estadoSlug sin consultar la API.
    if (!slugValido || esSlugReservado(slug)) return;
    let cancelado = false;
    const timer = setTimeout(async () => {
      try {
        const res = await api.verificarSlugDisponible(slug);
        if (!cancelado) setResultadoSlug({ slug, estado: res.disponible ? "disponible" : "en-uso" });
      } catch (err: unknown) {
        if (!cancelado) setResultadoSlug({ slug, estado: esErrorDeConexion(err) ? "sin-verificar" : "invalido" });
      }
    }, 400);
    return () => {
      cancelado = true;
      clearTimeout(timer);
    };
  }, [slug, slugValido]);

  const estadoSlug: EstadoSlug = !slug
    ? "vacio"
    : !slugValido
      ? "invalido"
      : esSlugReservado(slug)
        ? "en-uso"
        : resultadoSlug?.slug === slug
        ? resultadoSlug.estado
        : "verificando";

  if (!cuenta) return null;

  const nombrePila = cuenta.nombre.split(" ")[0] || "";
  const negocioValido = nombreNegocio.trim().length >= 2;
  const emailSucursalValido = !emailSucursal.trim() || emailValido(emailSucursal);

  const irAPaso = (destino: number) => {
    setEnviado(false);
    setError("");
    setPaso(destino);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const continuar = () => {
    setEnviado(true);
    if (paso === 1) {
      if (!negocioValido || estadoSlug === "vacio" || estadoSlug === "invalido" || estadoSlug === "en-uso") return;
      if (estadoSlug === "verificando") return;
    }
    if (paso === 2 && (!nombreSucursal.trim() || !emailSucursalValido)) return;
    irAPaso(Math.min(TOTAL_PASOS, paso + 1));
  };

  const crearNegocio = async () => {
    setError("");
    if (!negocioCreado) {
      setCreando("Creando tu negocio…");
      try {
        await api.crearTenant({
          nombre: nombreNegocio.trim(),
          nombre_fantasia: nombreNegocio.trim(),
          slug,
          rubro,
          descripcion: rubro === "otro" ? rubroDetalle.trim() : undefined,
          email_admin: cuenta.email,
          nombre_admin: cuenta.nombre,
          password: cuenta.usarMagicLink || cuenta.googleCredential ? undefined : cuenta.password,
          google_credential: cuenta.googleCredential || undefined,
          sucursal_nombre: nombreSucursal.trim(),
          whatsapp: whatsapp.trim(),
          email_sucursal: emailSucursal.trim(),
          horarios: HORARIO_ESTANDAR,
        });
        setNegocioCreado(true);
      } catch (err: unknown) {
        setCreando("");
        const mensaje = getErrorMessage(err, "No pudimos crear el negocio. Revisá los datos e intentá nuevamente.");
        const normalizado = mensaje.toLowerCase();
        if (normalizado.includes("nombre en url")) {
          setResultadoSlug({ slug, estado: "en-uso" });
          setEditandoSlug(true);
          irAPaso(1);
          setError("Ese nombre en URL ya está en uso. Elegí otro para continuar.");
          return;
        }
        if (normalizado.includes("correo de acceso") || normalizado.includes("cuenta de google")) {
          setError(`${mensaje} Volvé a crear tu cuenta para continuar.`);
          return;
        }
        setError(mensaje);
        return;
      }
    }

    // Si el usuario eligió enlace mágico, se solicita el link y se muestra pantalla de confirmación.
    if (cuenta.usarMagicLink) {
      setCreando("Enviando enlace de acceso…");
      try {
        const res = await api.solicitarMagicLink(cuenta.email);
        setCreando("");
        setMagicLinkEnviado({ email: cuenta.email, devLink: res.magic_link_dev });
        return;
      } catch (err: unknown) {
        setCreando("");
        setError(getErrorMessage(err, "El negocio fue creado pero ocurrió un error al enviar el enlace mágico. Podés solicitarlo desde el inicio de sesión."));
        return;
      }
    }

    // El negocio ya existe con contraseña o Google: se inicia la sesión y se crean las mesas.
    setCreando("Preparando tus mesas…");
    try {
      if (cuenta.googleCredential) {
        await api.autenticarConGoogle(cuenta.googleCredential);
      } else {
        await api.autenticarConPassword(cuenta.email, cuenta.password || "");
      }
    } catch {
      setCreando("");
      setError("Tu negocio ya fue creado, pero no pudimos iniciar sesión automáticamente. Podés reintentar o ingresar con tu correo y contraseña.");
      return;
    }

    try {
      const sucursales = await api.listarSucursales();
      const sucursalID = sucursales[0]?.id;
      if (sucursalID) {
        for (let numero = 1; numero <= mesas; numero++) {
          await api.crearMesa({ sucursal_id: sucursalID, numero, capacidad: CAPACIDAD_MESA });
        }
      }
    } catch (err: unknown) {
      // Las mesas que falten se pueden crear desde el panel; no bloquea el ingreso.
      console.warn("No se pudieron crear todas las mesas del registro:", err);
    }

    router.replace("/dashboard");
  };

  const rubroActual = RUBROS.find((r) => r.valor === rubro) ?? RUBROS[0];
  const iniciales = nombreNegocio.trim() ? nombreNegocio.trim().slice(0, 2).toUpperCase() : "FD";

  return (
    <div className="registro">
      <header className="auth-header">
        <RegistroBrand />
        <Link href="/" className="onboarding-exit">Salir</Link>
      </header>
      <div className="onboarding-progress-mobile"><span style={{ width: `${(paso / TOTAL_PASOS) * 100}%` }} /></div>

      <main className="onboarding-layout">
        <aside className="onboarding-aside">
          <div>
            <span className="onboarding-welcome">Hola{nombrePila ? `, ${nombrePila}` : ""}</span>
            <h2>Preparemos tu espacio de trabajo</h2>
            <p>Solo necesitamos lo esencial. Después vas a poder personalizar todo desde Configuración.</p>
          </div>
          <nav className="onboarding-steps">
            {PASOS.map(([titulo, detalle], index) => {
              const numero = index + 1;
              const clase = paso === numero ? "active" : paso > numero ? "done" : "";
              return (
                <button key={titulo} type="button" className={clase} onClick={() => numero < paso && !creando && irAPaso(numero)}>
                  <span>{paso > numero ? <LandingIcon name="check" size={16} /> : numero}</span>
                  <div><strong>{titulo}</strong><small>{detalle}</small></div>
                </button>
              );
            })}
          </nav>
          <div className="onboarding-help">
            <LandingIcon name="help" />
            <span><strong>¿Te trabaste en algo?</strong><a href="mailto:supportmesaclick@gmail.com">Escribir a soporte</a></span>
          </div>
        </aside>

        <section className="onboarding-main">
          <div className="onboarding-step-label">Paso {paso} de {TOTAL_PASOS}</div>

          {paso === 1 && (
            <div className="onboarding-form">
              <div className="onboarding-title">
                <span className="onboarding-icon"><LandingIcon name="brand" /></span>
                <div><h1>Contanos sobre tu negocio</h1><p>Esto es lo que verán tus clientes cuando ingresen a la carta.</p></div>
              </div>

              <label className="onboarding-field">
                <span>Nombre del restaurante</span>
                <input
                  className={enviado && !negocioValido ? "invalid" : ""}
                  value={nombreNegocio}
                  onChange={(event) => setNombreNegocio(event.target.value)}
                  placeholder="Ej: Bajo Limonero"
                  autoFocus
                />
                {enviado && !negocioValido && <small>Ingresá el nombre de tu negocio.</small>}
              </label>

              <fieldset className="business-types">
                <legend>¿Qué tipo de negocio tenés?</legend>
                <div>
                  {RUBROS.map((opcion) => (
                    <button type="button" key={opcion.valor} className={rubro === opcion.valor ? "active" : ""} onClick={() => setRubro(opcion.valor)} aria-pressed={rubro === opcion.valor}>
                      <span><LandingIcon name={opcion.icon} /></span>
                      <strong>{opcion.label}</strong>
                      {rubro === opcion.valor && <i><LandingIcon name="check" size={14} /></i>}
                    </button>
                  ))}
                </div>
              </fieldset>

              {rubro === "otro" && (
                <label className="onboarding-field">
                  <span>¿Qué tipo de negocio es? <em>Opcional</em></span>
                  <input
                    value={rubroDetalle}
                    onChange={(event) => setRubroDetalle(event.target.value.slice(0, 120))}
                    placeholder="Ej: panadería artesanal, food truck…"
                    autoFocus
                  />
                  <small>Nos ayuda a adaptar mejor la experiencia; podés completarlo después.</small>
                </label>
              )}

              <div className="slug-preview">
                <span className="onboarding-icon small"><LandingIcon name="book" size={18} /></span>
                <div>
                  <small>Tu nombre en URL de fudd</small>
                  {editandoSlug ? (
                    <input
                      value={slug}
                      onChange={(event) => setSlugManual(limpiarSlugEscrito(event.target.value))}
                      aria-label="Nombre en URL"
                      placeholder="tu-negocio"
                    />
                  ) : (
                    <strong>/<b>{slug || "tu-negocio"}</b></strong>
                  )}
                </div>
                <div className="slug-actions">
                  <EstadoSlugLabel estado={estadoSlug} />
                  <button type="button" onClick={() => { setSlugManual(slug); setEditandoSlug(!editandoSlug); }}>
                    {editandoSlug ? "Listo" : "Editar"}
                  </button>
                </div>
              </div>
              {enviado && (estadoSlug === "en-uso" || estadoSlug === "invalido") && (
                <div role="alert" className="auth-alert">
                  {estadoSlug === "en-uso"
                    ? "Ese nombre en URL ya lo usa otro negocio. Editalo para continuar."
                    : "Usá solo letras, números y guiones en el nombre en URL."}
                </div>
              )}
              {error && <div role="alert" className="auth-alert">{error}</div>}
            </div>
          )}

          {paso === 2 && (
            <div className="onboarding-form">
              <div className="onboarding-title">
                <span className="onboarding-icon"><LandingIcon name="table" /></span>
                <div><h1>Tu primera sucursal</h1><p>Creá el local desde el que vas a empezar a operar.</p></div>
              </div>

              <label className="onboarding-field">
                <span>Nombre de la sucursal</span>
                <input
                  className={enviado && !nombreSucursal.trim() ? "invalid" : ""}
                  value={nombreSucursal}
                  onChange={(event) => setNombreSucursal(event.target.value)}
                  placeholder="Ej: Casa central"
                  autoFocus
                />
                {enviado && !nombreSucursal.trim() && <small>Ingresá el nombre de la sucursal.</small>}
              </label>
              <div className="onboarding-field-row">
                <label className="onboarding-field">
                  <span>WhatsApp <em>Opcional</em></span>
                  <input
                    value={whatsapp}
                    onChange={(event) => setWhatsapp(normalizarWhatsapp(event.target.value))}
                    placeholder="+54 9 11 1234 5678"
                    inputMode="tel"
                    autoComplete="tel"
                    aria-label="WhatsApp del local"
                  />
                </label>
                <label className="onboarding-field">
                  <span>Correo del local <em>Opcional</em></span>
                  <input
                    className={enviado && !emailSucursalValido ? "invalid" : ""}
                    value={emailSucursal}
                    onChange={(event) => setEmailSucursal(event.target.value)}
                    placeholder="local@turestaurante.com"
                    type="email"
                  />
                  {enviado && !emailSucursalValido && <small>Ingresá un correo válido.</small>}
                </label>
              </div>

              <div className="tables-question">
                <div>
                  <span className="onboarding-icon small"><LandingIcon name="table" size={18} /></span>
                  <div><strong>¿Cuántas mesas tenés?</strong><p>Vamos a crearlas automáticamente con su QR. Podés cambiarlas después.</p></div>
                </div>
                <div className="table-counter">
                  <button type="button" onClick={() => setMesas(Math.max(1, mesas - 1))} disabled={mesas <= 1} aria-label="Quitar una mesa">−</button>
                  <strong>{mesas}</strong>
                  <button type="button" onClick={() => setMesas(Math.min(MAX_MESAS_INICIALES, mesas + 1))} disabled={mesas >= MAX_MESAS_INICIALES} aria-label="Agregar una mesa">+</button>
                </div>
                <span className="table-limit">Creamos hasta {MAX_MESAS_INICIALES} ahora. Podés sumar más durante tus {PLAN_PRUEBA_MESES} meses de prueba Pro.</span>
              </div>

              <div className="schedule-choice">
                <div>
                  <span className="onboarding-icon small"><LandingIcon name="clock" size={18} /></span>
                  <div><strong>Horarios de atención</strong><p>Empezamos con el horario estándar. Podés configurar turnos y días específicos más adelante.</p></div>
                </div>
                <div><LandingIcon name="check" size={16} /> Horario estándar, todos los días <span>12:00 — 00:00</span></div>
              </div>
            </div>
          )}

          {paso === 3 && (
            <>
              {magicLinkEnviado ? (
                <div className="onboarding-complete">
                  <span className="complete-mark">
                    <LandingIcon name="check" size={32} />
                  </span>
                  <span className="auth-eyebrow">¡Negocio creado con éxito!</span>
                  <h1>Revisá tu correo</h1>
                  <p>
                    Te enviamos un enlace de acceso seguro a <strong>{magicLinkEnviado.email}</strong> para entrar al panel de administración de <strong>{nombreNegocio.trim()}</strong>.
                    Tu prueba Pro de {PLAN_PRUEBA_MESES} meses ya está activa.
                  </p>

                  {magicLinkEnviado.devLink && (
                    <div style={{ marginTop: "20px" }}>
                      <a
                        href={magicLinkEnviado.devLink}
                        className="onboarding-primary"
                        style={{ textDecoration: "none", display: "inline-flex", justifyContent: "center", width: "100%" }}
                      >
                        Ingresar al panel (modo desarrollo)
                      </a>
                    </div>
                  )}

                  <div style={{ marginTop: "16px" }}>
                    <Link
                      href="/login"
                      className="onboarding-back"
                      style={{ textDecoration: "none", display: "inline-flex", alignItems: "center", justifyContent: "center", width: "100%" }}
                    >
                      Ir al inicio de sesión
                    </Link>
                  </div>
                </div>
              ) : (
                <div className="onboarding-complete">
                  <span className="complete-mark"><LandingIcon name="check" size={32} /></span>
                  <span className="auth-eyebrow">Último paso</span>
                  <h1>Todo listo para empezar</h1>
                  <p>Revisá los datos de <strong>{nombreNegocio.trim()}</strong>. Al confirmar creamos tu negocio, tu sucursal y tus mesas con su QR.</p>
                  <div className="setup-summary">
                    <div>
                      <span><LandingIcon name="brand" /></span>
                      <p><small>Negocio</small><strong>{nombreNegocio.trim()}</strong><em>{rubroActual.label} · /{slug}</em></p>
                      <button type="button" onClick={() => irAPaso(1)} disabled={Boolean(creando)}>Editar</button>
                    </div>
                    <div>
                      <span><LandingIcon name="table" /></span>
                      <p><small>Sucursal</small><strong>{nombreSucursal.trim()}</strong><em>{mesas} {mesas === 1 ? "mesa" : "mesas"} · 12:00 — 00:00</em></p>
                      <button type="button" onClick={() => irAPaso(2)} disabled={Boolean(creando)}>Editar</button>
                    </div>
                    <div>
                      <span><LandingIcon name="qr" /></span>
                      <p>
                        <small>Acceso</small>
                        <strong>{cuenta.email}</strong>
                        <em>
                          {cuenta.googleCredential
                            ? "Con tu cuenta de Google"
                            : cuenta.usarMagicLink
                              ? "Con enlace mágico (sin contraseña)"
                              : "Con correo y contraseña"}
                        </em>
                      </p>
                      <i><LandingIcon name="check" size={14} /></i>
                    </div>
                    <div>
                      <span><LandingIcon name="spark" /></span>
                      <p><small>Plan inicial</small><strong>{PLAN_PRUEBA_MESES} meses de Pro gratis</strong><em>Después: Free o Pro por {formatearPrecioPlan()} {PLAN_MONEDA}/mes</em></p>
                      <i><LandingIcon name="check" size={14} /></i>
                    </div>
                  </div>
                  {error && (
                    <div role="alert" className="auth-alert">
                      {error}
                      {(error.includes("correo") || error.includes("Google")) && <> <Link href="/registro">Volver a crear la cuenta</Link></>}
                      {negocioCreado && <> <Link href={`/login?email=${encodeURIComponent(cuenta.email)}`}>Ir a iniciar sesión</Link></>}
                    </div>
                  )}
                  <button type="button" className="onboarding-primary finish" onClick={crearNegocio} disabled={Boolean(creando)}>
                    {creando || (negocioCreado ? "Reintentar inicio de sesión" : "Crear mi negocio")}
                  </button>
                  <small>Tu prueba Pro de {PLAN_PRUEBA_MESES} meses comienza al crear el negocio. Después te guiamos para cargar tu primer plato.</small>
                </div>
              )}
            </>
          )}

          {paso < TOTAL_PASOS && (
            <footer className="onboarding-actions">
              <button type="button" className="onboarding-back" disabled={paso === 1} onClick={() => irAPaso(paso - 1)}>Atrás</button>
              <span>Podés modificar estos datos cuando quieras.</span>
              <button type="button" className="onboarding-primary" onClick={continuar} disabled={paso === 1 && estadoSlug === "verificando" && enviado}>
                Continuar
              </button>
            </footer>
          )}
        </section>

        <aside className="onboarding-preview" aria-hidden="true">
          <div className="preview-orbit one" /><div className="preview-orbit two" />
          <span className="preview-label">Vista previa</span>
          <div className="onboarding-phone">
            <div className="phone-top">
              <span className="phone-logo">{iniciales}</span>
              <span><strong>{nombreNegocio.trim() || "Tu restaurante"}</strong><small>{paso >= 2 ? nombreSucursal.trim() || "Tu primera sucursal" : "Tu primera sucursal"}</small></span>
              <LandingIcon name="menu" size={18} />
            </div>
            <div className="phone-hero">
              <img src="https://images.unsplash.com/photo-1643757343278-5d50309dfa44?auto=format&fit=crop&w=700&q=82" alt="" />
              <span><small>Bienvenidos</small><strong>{nombreNegocio.trim() || "Tu restaurante"}</strong></span>
            </div>
            <div className="phone-categories"><span className="active">Recomendados</span><span>Entradas</span><span>Principales</span></div>
            <div className="phone-dish"><span><small>Muy pronto</small><strong>Tu carta empieza acá</strong><em>Cargá tus primeros platos</em></span><b>+</b></div>
          </div>
          <div className="preview-note">
            <span><LandingIcon name="spark" /></span>
            <p><strong>Así te verán tus clientes</strong><small>La vista se actualiza mientras completás los datos.</small></p>
          </div>
        </aside>
      </main>
    </div>
  );
}

function EstadoSlugLabel({ estado }: { estado: EstadoSlug }) {
  if (estado === "vacio") return null;
  const textos: Record<Exclude<EstadoSlug, "vacio">, [string, string]> = {
    disponible: ["", "Disponible"],
    "en-uso": ["taken", "En uso"],
    invalido: ["taken", "No válido"],
    verificando: ["checking", "Verificando…"],
    "sin-verificar": ["checking", "Se valida al crear"],
  };
  const [clase, texto] = textos[estado];
  return <span className={`available ${clase}`}><i /> {texto}</span>;
}
