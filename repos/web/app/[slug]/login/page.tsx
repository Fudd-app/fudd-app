"use client";

import { useCallback, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import GoogleSignInButton from "@/components/auth/GoogleSignInButton";
import Logo from "@/components/brand/Logo";
import { api, getErrorMessage } from "@/lib/api";

export default function LoginEquipoPage() {
  const { slug } = useParams<{ slug: string }>();
  const router = useRouter();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const clientID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? '';
  const ingresar = useCallback(async (credential: string) => {
    setLoading(true); setError('');
    try {
      const respuesta = await api.autenticarEquipoConGoogle(slug, credential);
      router.replace(respuesta.usuario.rol === 'cocina' ? '/kds' : '/dashboard');
    } catch (err: unknown) { setError(getErrorMessage(err, 'No pudimos iniciar sesión en este negocio.')); }
    finally { setLoading(false); }
  }, [router, slug]);

  return <main className="flex min-h-screen items-center justify-center bg-ghost-fog/50 p-16 font-inter">
    <section className="w-full max-w-[460px] rounded-2xl border border-concrete bg-canvas-white p-24 shadow-xl sm:p-32">
      <div className="flex items-center gap-10"><Logo className="h-36 w-36"/><div><p className="text-16 font-bold uppercase">Mesa CLICK</p><p className="text-11 font-mono text-sage-green">/{slug}</p></div></div>
      <p className="mt-28 text-11 font-mono uppercase tracking-[0.14em] text-sage-green">Acceso del equipo</p>
      <h1 className="mt-6 text-28 font-semibold text-ash-graphite">Entrá con Google</h1>
      <p className="mb-20 mt-8 text-13 leading-relaxed text-sage-green">Usá la misma cuenta con la que aceptaste tu invitación. El personal no necesita contraseña ni magic link para volver a entrar.</p>
      <GoogleSignInButton clientID={clientID} disabled={loading} onCredential={ingresar} onError={setError}/>
      {error && <div role="alert" className="mt-14 rounded-lg border border-alert-red/30 bg-warm-pink/20 p-12 text-12 text-alert-red">{error}</div>}
      <div className="mt-24 space-y-8 border-t border-ghost-fog pt-16 text-center text-12 text-sage-green"><p>¿Sos dueño? <Link href="/login" className="font-semibold text-plain-green">Acceso administración</Link></p><p>¿No tenés negocio? <Link href="/registro" className="font-semibold text-plain-green">Crear una cuenta</Link></p></div>
    </section>
  </main>;
}
