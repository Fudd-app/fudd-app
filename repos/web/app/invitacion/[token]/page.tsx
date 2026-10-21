"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import GoogleSignInButton from "@/components/auth/GoogleSignInButton";
import Logo from "@/components/brand/Logo";
import { api, getErrorMessage, InvitacionEquipoAPI } from "@/lib/api";

function nombreGoogle(credential: string): string {
  try {
    const payload = credential.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
    const parsed = JSON.parse(decodeURIComponent(Array.from(atob(payload), (c) => `%${c.charCodeAt(0).toString(16).padStart(2, '0')}`).join(''))) as { name?: string };
    return parsed.name?.trim() ?? '';
  } catch { return ''; }
}

export default function InvitacionPage() {
  const { token } = useParams<{ token: string }>();
  const router = useRouter();
  const [invitacion, setInvitacion] = useState<InvitacionEquipoAPI | null>(null);
  const [credential, setCredential] = useState('');
  const [nombre, setNombre] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [aceptando, setAceptando] = useState(false);
  const clientID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? '';

  useEffect(() => {
    void api.obtenerInvitacion(token).then(setInvitacion).catch((err: unknown) => setError(getErrorMessage(err, 'La invitación no es válida o ya venció.'))).finally(() => setLoading(false));
  }, [token]);

  const seleccionarGoogle = useCallback((valor: string) => {
    setCredential(valor);
    setNombre(nombreGoogle(valor));
    setError('');
  }, []);

  const aceptar = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!credential) { setError('Primero elegí la cuenta de Google invitada.'); return; }
    if (!nombre.trim()) { setError('Ingresá tu nombre.'); return; }
    setAceptando(true); setError('');
    try {
      const respuesta = await api.aceptarInvitacion(token, credential, nombre.trim());
      router.replace(respuesta.usuario.rol === 'cocina' ? '/kds' : '/dashboard');
    } catch (err: unknown) { setError(getErrorMessage(err, 'No pudimos aceptar la invitación.')); }
    finally { setAceptando(false); }
  };

  return <main className="min-h-screen bg-ghost-fog/50 p-16 font-inter sm:p-32">
    <section className="mx-auto max-w-[560px] overflow-hidden rounded-2xl border border-concrete bg-canvas-white shadow-xl">
      <header className="border-b border-concrete bg-ash-graphite p-24 text-canvas-white sm:p-32">
        <div className="flex items-center gap-10"><Logo className="h-32 w-32"/><span className="text-14 font-bold">fudd</span></div>
        <p className="mt-24 text-11 font-mono uppercase tracking-[0.16em] text-concrete">Invitación al equipo</p>
        <h1 className="mt-6 text-28 font-semibold">Sumate a {invitacion?.negocio ?? 'este negocio'}</h1>
      </header>
      <div className="space-y-20 p-20 sm:p-32">
        {loading && <p className="text-13 text-sage-green">Validando invitación…</p>}
        {invitacion && <div className="grid gap-10 rounded-xl border border-concrete bg-ghost-fog/60 p-16 text-13"><p><span className="text-sage-green">Sucursal</span><br/><strong>{invitacion.sucursal}</strong></p><p><span className="text-sage-green">Rol</span><br/><strong>{invitacion.rol}</strong></p><p><span className="text-sage-green">Correo invitado</span><br/><strong>{invitacion.email}</strong></p></div>}
        {invitacion && !credential && <><p className="text-13 leading-relaxed text-sage-green">Elegí exactamente la cuenta de Google asociada a <strong>{invitacion.email}</strong>. La vista previa no consume la invitación.</p><GoogleSignInButton clientID={clientID} disabled={aceptando} onCredential={seleccionarGoogle} onError={setError}/></>}
        {invitacion && credential && <form onSubmit={aceptar} className="space-y-12"><label className="block text-12 font-semibold text-ash-graphite">Tu nombre<input value={nombre} onChange={(e) => setNombre(e.target.value)} className="mt-6 h-48 w-full rounded-lg border border-concrete px-12 text-14 outline-none focus:border-plain-green" /></label><p className="text-11 text-sage-green">Google sugirió este nombre. Podés corregirlo antes de crear tu acceso.</p><button disabled={aceptando} className="h-48 w-full rounded-lg bg-plain-green text-14 font-semibold text-white disabled:opacity-50">{aceptando ? 'Creando acceso…' : 'Aceptar invitación e ingresar'}</button><button type="button" onClick={() => setCredential('')} className="w-full text-12 text-sage-green">Elegir otra cuenta</button></form>}
        {error && <div role="alert" className="rounded-lg border border-alert-red/30 bg-warm-pink/20 p-12 text-12 text-alert-red">{error}</div>}
        <p className="text-center text-11 text-sage-green">¿Administrás tu propio negocio? <Link href="/registro" className="font-semibold text-plain-green">Registralo acá</Link></p>
      </div>
    </section>
  </main>;
}
