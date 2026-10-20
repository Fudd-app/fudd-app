"use client";
/* eslint-disable @next/next/no-img-element -- logo dinámico del comercio */

import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import GoogleSignInButton from "@/components/auth/GoogleSignInButton";
import Logo from "@/components/brand/Logo";
import { api, getErrorMessage, NegocioPublicoAuth } from "@/lib/api";

export default function LoginEquipoPage() {
  const params = useParams<{ slug: string }>();
  const router = useRouter();
  const slug = (params?.slug || "").toLowerCase();
  const clientID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? "";

  const [negocio, setNegocio] = useState<NegocioPublicoAuth | null>(null);
  const [cargandoNegocio, setCargandoNegocio] = useState(true);
  const [noEncontrado, setNoEncontrado] = useState(false);

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [mostrarPassword, setMostrarPassword] = useState(false);
  const [enviando, setEnviando] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!slug) return;
    let cancelado = false;

    async function cargarNegocio() {
      setCargandoNegocio(true);
      setNoEncontrado(false);
      try {
        const data = await api.obtenerNegocioPublico(slug);
        if (!cancelado) {
          setNegocio(data);
        }
      } catch {
        if (!cancelado) {
          setNoEncontrado(true);
        }
      } finally {
        if (!cancelado) {
          setCargandoNegocio(false);
        }
      }
    }

    void cargarNegocio();
    return () => {
      cancelado = true;
    };
  }, [slug]);

  const redirigirPorRol = useCallback(
    (rol: string, redirectTo?: string) => {
      if (typeof window !== "undefined") {
        localStorage.setItem("mesaclick_staff_rol", rol);
        localStorage.setItem("mesaclick_staff_slug", slug);
        if (rol === "mozo") {
          localStorage.setItem("mesaclick_initial_section", "recepcionista");
        }
      }
      router.replace(redirectTo || (rol === "cocina" ? "/kds" : "/dashboard"));
    },
    [router, slug],
  );

  const handleSubmitCredenciales = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const cleanUser = username.trim().replace(/^@/, "");
    if (!cleanUser || !password) {
      setError("Ingresá tu usuario y contraseña.");
      return;
    }

    setEnviando(true);
    setError("");
    try {
      const res = await api.autenticarStaffNegocio(slug, cleanUser, password);
      redirigirPorRol(res.usuario.rol, res.redirect_to);
    } catch (err: unknown) {
      setError(getErrorMessage(err, "Usuario o contraseña incorrectos para este local."));
    } finally {
      setEnviando(false);
    }
  };

  const ingresarConGoogle = useCallback(
    async (credential: string) => {
      setEnviando(true);
      setError("");
      try {
        const respuesta = await api.autenticarEquipoConGoogle(slug, credential);
        redirigirPorRol(respuesta.usuario.rol);
      } catch (err: unknown) {
        setError(getErrorMessage(err, "No pudimos iniciar sesión en este negocio."));
      } finally {
        setEnviando(false);
      }
    },
    [redirigirPorRol, slug],
  );

  const nombreComercial = negocio?.nombre_fantasia || negocio?.nombre || slug;
  const colorAcento = negocio?.color_primario || "#1F4D3A";
  const iniciales =
    nombreComercial
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((p) => p[0]?.toUpperCase())
      .join("") || "MC";

  if (cargandoNegocio) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center bg-ghost-fog/60 p-16 font-inter">
        <div className="w-full max-w-[420px] rounded-2xl border border-concrete bg-canvas-white p-24 text-center shadow-lg">
          <div className="mx-auto mb-12 h-56 w-56 animate-pulse rounded-2xl bg-ghost-fog" />
          <div className="mx-auto mb-8 h-16 w-48 animate-pulse rounded bg-ghost-fog" />
          <div className="mx-auto h-12 w-32 animate-pulse rounded bg-ghost-fog" />
        </div>
      </div>
    );
  }

  if (noEncontrado || !negocio) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center bg-ghost-fog/60 p-16 font-inter">
        <div className="w-full max-w-[440px] rounded-2xl border border-concrete bg-canvas-white p-24 text-center shadow-lg sm:p-32">
          <div className="mx-auto mb-16 flex h-56 w-56 items-center justify-center rounded-2xl border border-concrete bg-ghost-fog text-ash-graphite">
            <span className="material-symbols-outlined text-28">storefront</span>
          </div>
          <h1 className="text-20 font-bold tracking-tight text-ash-graphite">
            Comercio no encontrado
          </h1>
          <p className="mt-8 text-13 leading-relaxed text-sage-green">
            No encontramos un local activo con el identificador{" "}
            <code className="rounded bg-ghost-fog px-6 py-2 font-mono text-ash-graphite">
              /{slug}
            </code>
            . Verificá el enlace con el encargado o administrador de tu local.
          </p>
          <div className="mt-24 flex flex-col gap-10">
            <Link
              href="/"
              className="inline-flex h-44 items-center justify-center rounded-xl bg-plain-green px-16 text-13 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted"
            >
              Ir al inicio de Mesa CLICK
            </Link>
            <Link
              href="/login"
              className="inline-flex h-40 items-center justify-center rounded-xl border border-concrete bg-canvas-white px-16 text-12 font-semibold text-ash-graphite transition-colors hover:bg-ghost-fog"
            >
              Ingresar al panel de propietarios
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen flex-col justify-between bg-ghost-fog/60 font-inter">
      <header className="flex items-center justify-between border-b border-concrete bg-canvas-white px-16 py-12 sm:px-24">
        <Link
          href="/"
          className="flex items-center gap-8 text-ash-graphite transition-opacity hover:opacity-80"
        >
          <Logo className="h-22 w-22" />
          <span className="text-13 font-bold uppercase tracking-tight">Mesa CLICK</span>
        </Link>
        <span className="rounded-full border border-concrete bg-ghost-fog px-10 py-4 font-mono text-11 text-sage-green">
          /{negocio.slug}/login
        </span>
      </header>

      <main className="flex flex-1 items-center justify-center p-16 sm:p-24">
        <div className="w-full max-w-[460px] overflow-hidden rounded-2xl border border-concrete bg-canvas-white shadow-xl">
          <div className="h-6 w-full" style={{ backgroundColor: colorAcento }} />

          <div className="p-24 sm:p-32">
            <div className="mb-24 flex flex-col items-center text-center">
              {negocio.logo_url ? (
                <img
                  src={negocio.logo_url}
                  alt={nombreComercial}
                  className="mb-12 h-64 w-64 rounded-2xl border border-concrete object-cover shadow-xs"
                />
              ) : (
                <div
                  className="mb-12 flex h-64 w-64 items-center justify-center rounded-2xl text-20 font-bold text-canvas-white shadow-xs"
                  style={{ backgroundColor: colorAcento }}
                >
                  {iniciales}
                </div>
              )}
              <span className="inline-flex items-center gap-4 rounded-full border border-concrete bg-ghost-fog px-10 py-2 text-11 font-semibold text-sage-green">
                Acceso de Equipo · {nombreComercial}
              </span>
              <h1 className="mt-8 text-22 font-bold tracking-tight text-ash-graphite">
                Iniciar turno en {nombreComercial}
              </h1>
              <p className="mt-4 text-12 text-sage-green">
                Ingresá con tu usuario y clave o con la cuenta de Google vinculada a tu invitación.
              </p>
            </div>

            <form onSubmit={handleSubmitCredenciales} className="space-y-14" noValidate>
              <label className="block space-y-6">
                <span className="text-12 font-semibold text-ash-graphite">
                  Usuario o correo electrónico
                </span>
                <div className="flex h-46 items-center rounded-xl border border-concrete bg-canvas-white px-12 focus-within:border-plain-green">
                  <span className="material-symbols-outlined mr-8 text-18 text-sage-green">
                    badge
                  </span>
                  <input
                    type="text"
                    autoComplete="username"
                    autoCapitalize="none"
                    autoCorrect="off"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    placeholder="Ej: mozo_juan o cocina_1"
                    className="h-full flex-1 bg-transparent text-14 text-ash-graphite outline-none"
                  />
                </div>
              </label>

              <label className="block space-y-6">
                <span className="text-12 font-semibold text-ash-graphite">Contraseña</span>
                <div className="flex h-46 items-center rounded-xl border border-concrete bg-canvas-white px-12 focus-within:border-plain-green">
                  <span className="material-symbols-outlined mr-8 text-18 text-sage-green">
                    lock
                  </span>
                  <input
                    type={mostrarPassword ? "text" : "password"}
                    autoComplete="current-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Tu clave de acceso"
                    className="h-full flex-1 bg-transparent text-14 text-ash-graphite outline-none"
                  />
                  <button
                    type="button"
                    onClick={() => setMostrarPassword((prev) => !prev)}
                    className="ml-8 text-12 font-semibold text-sage-green hover:text-ash-graphite"
                  >
                    {mostrarPassword ? "Ocultar" : "Ver"}
                  </button>
                </div>
              </label>

              <button
                type="submit"
                disabled={enviando}
                className="flex h-46 w-full items-center justify-center gap-8 rounded-xl bg-plain-green px-16 text-14 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted disabled:opacity-50"
              >
                {enviando ? "Ingresando al local…" : "Ingresar con usuario y clave"}
                {!enviando && <span className="material-symbols-outlined text-18">login</span>}
              </button>
            </form>

            <div className="my-20 flex items-center gap-12">
              <div className="h-1 flex-1 bg-concrete" />
              <span className="text-11 font-mono uppercase tracking-wider text-sage-green">
                o entrá con Google
              </span>
              <div className="h-1 flex-1 bg-concrete" />
            </div>

            <GoogleSignInButton
              clientID={clientID}
              disabled={enviando}
              onCredential={ingresarConGoogle}
              onError={setError}
            />

            {error && (
              <div
                role="alert"
                className="mt-14 flex items-start gap-8 rounded-xl border border-alert-red/30 bg-warm-pink/20 p-12 text-12 font-medium text-alert-red"
              >
                <span className="material-symbols-outlined shrink-0 text-16">error</span>
                <span>{error}</span>
              </div>
            )}

            <div className="mt-24 space-y-8 border-t border-ghost-fog pt-16 text-center text-12 text-sage-green">
              <p>
                ¿Sos el dueño o administrador del negocio?{" "}
                <Link href="/login" className="font-semibold text-plain-green hover:underline">
                  Acceso administración
                </Link>
              </p>
              <p>
                ¿No tenés negocio?{" "}
                <Link href="/registro" className="font-semibold text-plain-green hover:underline">
                  Crear una cuenta
                </Link>
              </p>
            </div>
          </div>
        </div>
      </main>

      <footer className="py-12 text-center text-11 text-sage-green">
        Operación gastronómica en tiempo real · Mesa CLICK
      </footer>
    </div>
  );
}
