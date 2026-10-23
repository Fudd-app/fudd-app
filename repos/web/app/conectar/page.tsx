"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { api, ApiError, codigoError, getErrorMessage, SesionUsuario, SolicitudMcp } from "@/lib/api";
import Logo from "@/components/brand/Logo";

type Estado =
  | { tipo: "cargando" }
  | { tipo: "incompleta" }
  | { tipo: "invalida" }
  | { tipo: "error"; mensaje: string }
  | { tipo: "ok"; solicitud: SolicitudMcp; sesion: SesionUsuario };

const MSG_SOLO_DUENIO = "Solo el dueño del negocio puede conectar asistentes de IA.";
const MSG_OTRO_NAVEGADOR =
  "Esta solicitud se inició en otro navegador. Volvé a conectar desde tu asistente de IA.";

// Defensa en profundidad: https, o http solo en loopback (clientes de escritorio).
function esRedirectPermitido(valor: string): boolean {
  try {
    const url = new URL(valor);
    if (url.protocol === "https:") return true;
    return (
      url.protocol === "http:" &&
      ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname)
    );
  } catch {
    return false;
  }
}

export default function ConectarPage() {
  // useSearchParams requiere un límite de Suspense para que la página siga prerrenderizable.
  return (
    <Suspense fallback={<Marco><p className="text-13 text-sage-green">Cargando…</p></Marco>}>
      <Conectar />
    </Suspense>
  );
}

function Marco({ children }: { children: React.ReactNode }) {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-24 bg-ghost-fog p-16">
      <Link href="/" aria-label="fudd">
        <Logo mark="full" className="h-28 w-auto" />
      </Link>
      <div className="w-full max-w-md rounded-xl border border-concrete bg-canvas-white p-20 sm:p-28">
        {children}
      </div>
    </main>
  );
}

function LinkComoConectar() {
  return (
    <Link href="/ia" className="mt-12 inline-block text-13 font-semibold text-ash-graphite underline">
      Cómo conectar Fudd con tu asistente
    </Link>
  );
}

function Conectar() {
  const router = useRouter();
  const solicitudId = useSearchParams().get("solicitud");
  const [estado, setEstado] = useState<Estado>(
    solicitudId ? { tipo: "cargando" } : { tipo: "incompleta" },
  );
  const [enviando, setEnviando] = useState<"aprobar" | "rechazar" | null>(null);
  const [redirigiendo, setRedirigiendo] = useState(false);
  const [errorAccion, setErrorAccion] = useState("");

  useEffect(() => {
    if (!solicitudId) return;
    let cancelado = false;
    const irALogin = () =>
      router.replace(
        "/login?returnTo=" +
          encodeURIComponent("/conectar?solicitud=" + encodeURIComponent(solicitudId)),
      );

    const cargar = async () => {
      let sesion: SesionUsuario;
      try {
        sesion = await api.obtenerSesion();
      } catch (err: unknown) {
        if (cancelado) return;
        if (err instanceof ApiError && err.status === 401) {
          irALogin();
        } else {
          setEstado({
            tipo: "error",
            mensaje: getErrorMessage(err, "No pudimos cargar la solicitud. Probá de nuevo."),
          });
        }
        return;
      }
      try {
        const solicitud = await api.obtenerSolicitudMcp(solicitudId);
        if (!cancelado) setEstado({ tipo: "ok", solicitud, sesion });
      } catch (err: unknown) {
        if (cancelado) return;
        const status = (err as { status?: number }).status;
        if (status === 404 || codigoError(err) === "SOLICITUD_INVALIDA") {
          setEstado({ tipo: "invalida" });
        } else if (status === 401) {
          irALogin();
        } else {
          setEstado({
            tipo: "error",
            mensaje: getErrorMessage(err, "No pudimos cargar la solicitud. Probá de nuevo."),
          });
        }
      }
    };

    void cargar();
    return () => {
      cancelado = true;
    };
  }, [solicitudId, router]);

  const responder = async (accion: "aprobar" | "rechazar") => {
    if (!solicitudId || enviando) return;
    setEnviando(accion);
    setErrorAccion("");
    try {
      const { redirect_url } =
        accion === "aprobar"
          ? await api.aprobarSolicitudMcp(solicitudId)
          : await api.rechazarSolicitudMcp(solicitudId);
      if (!esRedirectPermitido(redirect_url)) {
        setErrorAccion("No pudimos completar la conexión. Probá de nuevo.");
        setEnviando(null);
        return;
      }
      setRedirigiendo(true);
      // Navegación completa: el redirect apunta al sitio del asistente.
      window.location.assign(redirect_url);
    } catch (err: unknown) {
      const codigo = codigoError(err);
      if (codigo === "SOLICITUD_INVALIDA" || (err as { status?: number }).status === 404) {
        setEstado({ tipo: "invalida" });
      } else if (codigo === "MCP_SOLO_DUENIO") {
        setErrorAccion(MSG_SOLO_DUENIO);
      } else if (codigo === "SOLICITUD_OTRO_NAVEGADOR") {
        setErrorAccion(MSG_OTRO_NAVEGADOR);
      } else {
        setErrorAccion(getErrorMessage(err, "No pudimos completar la conexión. Probá de nuevo."));
      }
      setEnviando(null);
    }
  };

  if (estado.tipo === "cargando") {
    return (
      <Marco>
        <p className="text-13 text-sage-green">Cargando…</p>
      </Marco>
    );
  }

  if (estado.tipo === "incompleta") {
    return (
      <Marco>
        <p role="alert" className="text-15 font-semibold text-alert-red">
          El enlace de conexión está incompleto.
        </p>
        <LinkComoConectar />
      </Marco>
    );
  }

  if (estado.tipo === "invalida") {
    return (
      <Marco>
        <p role="alert" className="text-15 font-semibold text-alert-red">
          Esta solicitud de conexión no existe o venció. Volvé a conectar desde tu asistente de IA.
        </p>
        <LinkComoConectar />
      </Marco>
    );
  }

  if (estado.tipo === "error") {
    return (
      <Marco>
        <p role="alert" className="text-15 font-semibold text-alert-red">{estado.mensaje}</p>
        <LinkComoConectar />
      </Marco>
    );
  }

  const { solicitud, sesion } = estado;
  const esDuenio = sesion.rol === "admin";
  const puedePermitir = esDuenio && solicitud.mismo_navegador;
  const ocupado = enviando !== null || redirigiendo;

  return (
    <Marco>
      <p className="text-11 font-mono uppercase tracking-[0.14em] text-sage-green">
        Conectar asistente de IA
      </p>
      <h1 className="mt-4 text-24 font-semibold text-ash-graphite">
        {solicitud.client_name} quiere acceder a {solicitud.negocio}
      </h1>
      <p className="mt-8 text-13 text-sage-green">
        Al permitir, vas a volver a{" "}
        <strong className="text-ash-graphite">{solicitud.dominio_retorno}</strong>.
      </p>

      <div className="mt-20">
        <p className="text-12 font-semibold text-ash-graphite">Podrá ver:</p>
        <ul className="mt-8 space-y-6">
          {solicitud.permisos.map((permiso) => (
            <li key={permiso} className="flex items-start gap-8 text-13 text-ash-graphite">
              <span
                aria-hidden="true"
                className="material-symbols-outlined text-18 text-success-muted select-none"
              >
                check
              </span>
              {permiso}
            </li>
          ))}
        </ul>
        {solicitud.solo_lectura && (
          <span className="mt-12 inline-block rounded-full bg-vanilla-cream px-12 py-4 text-12 font-semibold text-ash-graphite">
            Solo lectura: no puede modificar nada
          </span>
        )}
      </div>

      {!solicitud.mismo_navegador && (
        <div
          role="alert"
          className="mt-20 rounded-lg border border-alert-red bg-warm-pink p-12 text-13 font-semibold text-alert-red"
        >
          Esta conexión se inició en otro navegador o dispositivo. Si no la iniciaste vos, cancelá.
        </div>
      )}
      {!esDuenio && (
        <div
          role="alert"
          className="mt-12 rounded-lg border border-alert-red bg-warm-pink p-12 text-13 font-semibold text-alert-red"
        >
          {MSG_SOLO_DUENIO}
        </div>
      )}
      {errorAccion && (
        <div
          role="alert"
          className="mt-12 rounded-lg border border-alert-red bg-warm-pink p-12 text-13 text-alert-red"
        >
          {errorAccion}
        </div>
      )}

      <p className="mt-20 text-12 text-stone">
        Permití solo si iniciaste vos esta conexión desde tu asistente de IA.
      </p>

      {redirigiendo ? (
        <p role="status" className="mt-16 text-13 font-semibold text-ash-graphite">
          Listo, volviendo a {solicitud.client_name}…
        </p>
      ) : (
        <div className="mt-16 flex flex-col gap-8 sm:flex-row-reverse">
          <button
            type="button"
            onClick={() => void responder("aprobar")}
            disabled={!puedePermitir || ocupado}
            className="h-44 w-full rounded-lg bg-plain-green px-16 text-12 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted disabled:cursor-not-allowed disabled:opacity-40 sm:flex-1"
          >
            {enviando === "aprobar" ? "Conectando…" : "Permitir"}
          </button>
          <button
            type="button"
            onClick={() => void responder("rechazar")}
            disabled={ocupado}
            className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-16 text-12 font-semibold text-ash-graphite transition-colors hover:bg-ghost-fog disabled:cursor-not-allowed disabled:opacity-40 sm:flex-1"
          >
            {enviando === "rechazar" ? "Cancelando…" : "Cancelar"}
          </button>
        </div>
      )}

      <p className="mt-20 text-11 text-stone">
        Conectado como {sesion.nombre || sesion.email}
      </p>
    </Marco>
  );
}
