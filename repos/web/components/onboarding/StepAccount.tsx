import React, { useState } from "react";
import GoogleSignInButton from "@/components/auth/GoogleSignInButton";
import LandingIcon from "@/components/landing/LandingIcon";

interface StepAccountProps {
  data: {
    nombreAdmin: string;
    emailAdmin: string;
    password: string;
    confirmarPassword: string;
  };
  errors?: Partial<Record<"nombreAdmin" | "emailAdmin" | "password" | "confirmarPassword", string>>;
  loading?: boolean;
  onChange: (fields: Partial<{
    nombreAdmin: string;
    emailAdmin: string;
    password: string;
    confirmarPassword: string;
  }>) => void;
  onNext: () => void | Promise<void>;
  google: {
    clientID: string;
    // Correo de la cuenta de Google conectada; si está, no se pide contraseña.
    email?: string;
    disabled?: boolean;
    onCredential: (credential: string) => void;
    onUsarCorreo: () => void;
    onError: (message: string) => void;
  };
}

export default function StepAccount({
  data,
  errors = {},
  loading = false,
  onChange,
  onNext,
  google,
}: StepAccountProps) {
  const [mostrarPassword, setMostrarPassword] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (google.email) {
      if (!data.nombreAdmin.trim()) return;
    } else if (!data.emailAdmin.trim() || !data.nombreAdmin.trim() || !data.password || !data.confirmarPassword) {
      return;
    }
    void onNext();
  };

  return (
    <div className="space-y-24 font-inter">
      {google.email ? (
        <div className="auth-note">
          <span><LandingIcon name="check" size={14} /></span>
          <p>
            <strong>Cuenta de Google conectada.</strong> Vas a ingresar con <strong className="font-mono">{google.email}</strong>, sin contraseña.{" "}
            <button type="button" onClick={google.onUsarCorreo} className="font-bold text-ash-graphite underline underline-offset-4">
              Usar correo y contraseña
            </button>
          </p>
        </div>
      ) : (
        <div>
          <GoogleSignInButton
            clientID={google.clientID}
            disabled={google.disabled || loading}
            onCredential={google.onCredential}
            onError={google.onError}
            text="signup_with"
          />
          <div className="mt-20 flex items-center gap-12" aria-hidden="true">
            <span className="h-px flex-1 bg-concrete" />
            <span className="text-10 font-bold uppercase tracking-wider text-stone">o con tu correo</span>
            <span className="h-px flex-1 bg-concrete" />
          </div>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-20">
        <div className="space-y-8">
          <label
            htmlFor="nombreAdmin"
            className="block text-11 font-bold"
          >
            Nombre completo del responsable
          </label>
          <input
            id="nombreAdmin"
            type="text"
            required
            value={data.nombreAdmin}
            onChange={(e) => onChange({ nombreAdmin: e.target.value })}
            placeholder="Ej: Pablo Aguirre"
            aria-invalid={Boolean(errors.nombreAdmin)}
            className={`w-full h-52 px-16 bg-canvas-white border rounded-lg focus:border-plain-green outline-none transition-all text-15 ${
              errors.nombreAdmin ? "border-alert-red" : "border-ash-graphite"
            }`}
          />
          {errors.nombreAdmin && (
            <p className="text-11 text-alert-red px-1">{errors.nombreAdmin}</p>
          )}
        </div>

        {!google.email && (
          <>
        <div className="space-y-8">
          <label
            htmlFor="emailAdmin"
            className="block text-11 font-bold"
          >
            Email de acceso
          </label>
          <input
            id="emailAdmin"
            type="email"
            required
            value={data.emailAdmin}
            onChange={(e) => onChange({ emailAdmin: e.target.value })}
            placeholder="admin@minegocio.com"
            aria-invalid={Boolean(errors.emailAdmin)}
            className={`w-full h-52 px-16 bg-canvas-white border rounded-lg focus:border-plain-green outline-none transition-all text-15 ${
              errors.emailAdmin ? "border-alert-red" : "border-ash-graphite"
            }`}
          />
          {errors.emailAdmin && (
            <p className="text-11 text-alert-red px-1">{errors.emailAdmin}</p>
          )}
        </div>

        <div className="grid gap-16 sm:grid-cols-2">
          <div className="space-y-8">
            <label
              htmlFor="password"
              className="block text-11 font-bold"
            >
              Contraseña
            </label>
            <div className="relative">
              <input
                id="password"
                type={mostrarPassword ? "text" : "password"}
                autoComplete="new-password"
                required
                value={data.password}
                onChange={(e) => onChange({ password: e.target.value })}
                placeholder="Mínimo 10 caracteres"
                aria-invalid={Boolean(errors.password)}
                className={`w-full h-52 px-16 pr-48 bg-canvas-white border rounded-lg focus:border-plain-green outline-none transition-all text-15 ${
                  errors.password ? "border-alert-red" : "border-ash-graphite"
                }`}
              />
              <button
                type="button"
                onClick={() => setMostrarPassword((actual) => !actual)}
                aria-label={mostrarPassword ? "Ocultar contraseñas" : "Mostrar contraseñas"}
                className="absolute inset-y-0 right-0 flex w-44 items-center justify-center text-stone hover:text-ash-graphite"
              >
                <span className="material-symbols-outlined text-20">
                  {mostrarPassword ? "visibility_off" : "visibility"}
                </span>
              </button>
            </div>
            {errors.password && <p className="text-11 text-alert-red px-1">{errors.password}</p>}
          </div>

          <div className="space-y-8">
            <label
              htmlFor="confirmarPassword"
              className="block text-11 font-bold"
            >
              Confirmar contraseña
            </label>
            <input
              id="confirmarPassword"
              type={mostrarPassword ? "text" : "password"}
              autoComplete="new-password"
              required
              value={data.confirmarPassword}
              onChange={(e) => onChange({ confirmarPassword: e.target.value })}
              placeholder="Repetí la contraseña"
              aria-invalid={Boolean(errors.confirmarPassword)}
              className={`w-full h-52 px-16 bg-canvas-white border rounded-lg focus:border-plain-green outline-none transition-all text-15 ${
                errors.confirmarPassword ? "border-alert-red" : "border-ash-graphite"
              }`}
            />
            {errors.confirmarPassword && (
              <p className="text-11 text-alert-red px-1">{errors.confirmarPassword}</p>
            )}
          </div>
        </div>

        <div className="p-16 bg-vanilla-cream rounded-lg text-13 text-sage-green space-y-6">
          <div className="flex items-center gap-6 font-medium text-ash-graphite">
            <span className="material-symbols-outlined text-16 text-plain-green">verified_user</span>
            <span>Acceso seguro</span>
          </div>
          <p className="text-13 leading-normal">
            Creá una contraseña exclusiva para Zelko. También vas a poder ingresar con Google usando este mismo correo.
          </p>
        </div>
          </>
        )}

        <button
          type="submit"
          disabled={loading}
          className="landing-cta mt-24 w-full disabled:cursor-not-allowed disabled:opacity-40"
        >
          {loading ? (
            <>
              Validando correo
              <span className="animate-spin material-symbols-outlined text-16">
                progress_activity
              </span>
            </>
          ) : (
            "Continuar"
          )}
        </button>
      </form>
    </div>
  );
}
