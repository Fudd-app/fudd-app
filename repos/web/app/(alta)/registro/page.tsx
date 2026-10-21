"use client";
/* eslint-disable @next/next/no-img-element -- imagen de referencia de la maqueta (Unsplash) */

import React, { useCallback, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, getErrorMessage } from "@/lib/api";
import GoogleSignInButton from "@/components/auth/GoogleSignInButton";
import LandingIcon from "@/components/landing/LandingIcon";
import RegistroBrand from "@/components/registro/RegistroBrand";
import { useRegistro } from "@/components/registro/RegistroContext";
import { emailValido, errorPassword, esErrorDeConexion, leerIdentidadGoogle } from "@/components/registro/validaciones";
import { PLAN_PRUEBA_MESES } from "@/lib/planes";

type Errores = Partial<Record<"nombre" | "email" | "password", string>>;

export default function RegistroPage() {
  const router = useRouter();
  const { cuenta, setCuenta } = useRegistro();
  const googleClientID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? "";

  // Si vuelve desde /onboarding con correo y contraseña o magic link, se precompleta lo cargado.
  const [metodo, setMetodo] = useState<"password" | "magic">(cuenta?.usarMagicLink ? "magic" : "password");
  const [nombre, setNombre] = useState(cuenta?.nombre ?? "");
  const [email, setEmail] = useState(cuenta && !cuenta.googleCredential ? cuenta.email : "");
  const [password, setPassword] = useState(cuenta?.password ?? "");
  const [mostrarPassword, setMostrarPassword] = useState(false);
  const [errores, setErrores] = useState<Errores>({});
  const [error, setError] = useState("");
  const [emailEnUso, setEmailEnUso] = useState(false);
  const [validando, setValidando] = useState(false);

  // Devuelve si el correo puede usarse; si el servidor no responde, deja seguir.
  const emailDisponible = async (correo: string): Promise<boolean> => {
    try {
      const res = await api.verificarEmailAdminDisponible(correo);
      if (!res.disponible) {
        setEmailEnUso(true);
        setError("Ese correo ya tiene un negocio registrado en Zelko.");
        return false;
      }
      return true;
    } catch (err: unknown) {
      if (esErrorDeConexion(err)) return true;
      setError(getErrorMessage(err, "No pudimos validar el correo. Intentá nuevamente."));
      return false;
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    setEmailEnUso(false);

    const nuevosErrores: Errores = {};
    if (nombre.trim().length < 3) nuevosErrores.nombre = "Ingresá tu nombre completo.";
    if (!emailValido(email)) nuevosErrores.email = "Ingresá un correo válido.";
    if (metodo === "password") {
      const errPassword = errorPassword(password);
      if (errPassword) nuevosErrores.password = errPassword;
    }
    setErrores(nuevosErrores);
    if (Object.keys(nuevosErrores).length > 0) return;

    setValidando(true);
    const disponible = await emailDisponible(email);
    setValidando(false);
    if (!disponible) return;

    setCuenta({
      nombre: nombre.trim(),
      email: email.trim().toLowerCase(),
      password: metodo === "password" ? password : "",
      googleCredential: "",
      usarMagicLink: metodo === "magic",
    });
    router.push("/onboarding");
  };

  const handleGoogleCredential = async (credential: string) => {
    setError("");
    setEmailEnUso(false);
    const identidad = leerIdentidadGoogle(credential);
    if (!identidad.email) {
      setError("No pudimos leer tu cuenta de Google. Intentá nuevamente.");
      return;
    }

    setValidando(true);
    const disponible = await emailDisponible(identidad.email);
    setValidando(false);
    if (!disponible) return;

    setCuenta({
      nombre: identidad.nombre || nombre.trim(),
      email: identidad.email.toLowerCase(),
      password: "",
      googleCredential: credential,
    });
    router.push("/onboarding");
  };

  const handleGoogleError = useCallback((message: string) => setError(message), []);

  return (
    <div className="registro">
      <header className="auth-header">
        <RegistroBrand />
        <span className="auth-header-link">¿Ya tenés una cuenta? <Link href="/login">Ingresar</Link></span>
      </header>
      <main className="auth-layout">
        <section className="auth-form-panel">
          <div className="auth-form">
            <span className="auth-eyebrow">{PLAN_PRUEBA_MESES} meses de Pro gratis</span>
            <h1>Creá tu cuenta</h1>
            <p>Primero, tus datos de acceso. Al crear tu negocio activamos Pro durante {PLAN_PRUEBA_MESES} meses, sin cargo ni tarjeta.</p>

            <div className="google-slot">
              <GoogleSignInButton
                clientID={googleClientID}
                disabled={validando}
                onCredential={handleGoogleCredential}
                onError={handleGoogleError}
                text="signup_with"
              />
            </div>
            <div className="auth-divider"><span>o registrate con tu correo</span></div>

            <div className="auth-method-tabs" role="tablist" aria-label="Modalidad de registro">
              <button
                type="button"
                role="tab"
                aria-selected={metodo === "password"}
                className={`auth-tab-btn ${metodo === "password" ? "active" : ""}`}
                onClick={() => {
                  setMetodo("password");
                  setErrores({});
                }}
              >
                Con contraseña
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={metodo === "magic"}
                className={`auth-tab-btn ${metodo === "magic" ? "active" : ""}`}
                onClick={() => {
                  setMetodo("magic");
                  setErrores({});
                }}
              >
                Enlace mágico (sin clave)
              </button>
            </div>

            <form onSubmit={handleSubmit} noValidate>
              <label className="auth-field">
                <span>Nombre y apellido</span>
                <div className={errores.nombre ? "invalid" : ""}>
                  <LandingIcon name="users" size={18} />
                  <input value={nombre} onChange={(event) => setNombre(event.target.value)} placeholder="Ej: Martina López" autoComplete="name" />
                </div>
                {errores.nombre && <small>{errores.nombre}</small>}
              </label>

              <label className="auth-field">
                <span>Correo electrónico</span>
                <div className={errores.email || emailEnUso ? "invalid" : ""}>
                  <span className="field-at">@</span>
                  <input
                    value={email}
                    onChange={(event) => { setEmail(event.target.value); setEmailEnUso(false); }}
                    placeholder="vos@turestaurante.com"
                    type="email"
                    autoComplete="email"
                  />
                </div>
                {errores.email && <small>{errores.email}</small>}
              </label>

              {metodo === "password" ? (
                <label className="auth-field">
                  <span>Contraseña</span>
                  <div className={errores.password ? "invalid" : ""}>
                    <LandingIcon name="lock" size={18} />
                    <input
                      value={password}
                      onChange={(event) => setPassword(event.target.value)}
                      placeholder="Mínimo 10 caracteres"
                      type={mostrarPassword ? "text" : "password"}
                      autoComplete="new-password"
                    />
                    <button type="button" onClick={() => setMostrarPassword(!mostrarPassword)}>
                      {mostrarPassword ? "Ocultar" : "Ver"}
                    </button>
                  </div>
                  {errores.password ? <small>{errores.password}</small> : <em>Usá 10 caracteres o más, con letras y números.</em>}
                </label>
              ) : (
                <div className="auth-magic-info">
                  <LandingIcon name="spark" size={16} />
                  <span>Sin contraseñas para memorizar. Te enviaremos un enlace de acceso seguro a tu correo para ingresar cada vez.</span>
                </div>
              )}

              {error && (
                <div role="alert" className="auth-alert">
                  {error}
                  {emailEnUso && <> <Link href="/login">Iniciá sesión</Link> o usá otro correo.</>}
                </div>
              )}

              <button type="submit" className="auth-submit" disabled={validando}>
                {validando ? "Validando correo…" : "Crear cuenta"}
              </button>
            </form>

            <small className="auth-terms">
              Al continuar, aceptás nuestros <Link href="/terminos">Términos de uso</Link> y la <Link href="/privacidad">Política de privacidad</Link>.
            </small>
            <p className="auth-mobile-alternate">
              ¿Ya tenés una cuenta? <Link href="/login">Ingresar</Link>
            </p>
          </div>
        </section>

        <aside className="auth-story" aria-hidden="true">
          <img src="https://images.unsplash.com/photo-1517248135467-4c7edcad34c4?auto=format&fit=crop&w=1300&q=86" alt="" />
          <div className="auth-story-overlay">
            <span className="story-label"><i /> Configuración simple</span>
            <blockquote>“En pocos minutos tenés tu salón listo para recibir el primer pedido.”</blockquote>
            <div><span className="story-avatar">ZK</span><p><strong>Tu operación, conectada</strong><small>Salón · Cocina · Comensales</small></p></div>
          </div>
          <div className="story-card">
            <span><LandingIcon name="check" /></span>
            <div><small>Siguiente paso</small><strong>Contanos sobre tu negocio</strong></div>
          </div>
        </aside>
      </main>
    </div>
  );
}
