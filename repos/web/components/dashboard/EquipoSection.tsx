"use client";

import { useCallback, useEffect, useState } from "react";
import { api, EquipoAPI, getErrorMessage, InvitacionEquipoAPI, SesionUsuario, Sucursal, UsuarioAPI } from "@/lib/api";
import { EmptyState, Skeleton, useConfirm, useToast } from "@/components/ui";

type RolInvitable = 'encargado' | 'mozo' | 'cocina';
type EntregaInvitacion = { link: string; email: string; emailEnviado: boolean; advertencia?: string };

const DESCRIPCIONES: Record<RolInvitable, string> = {
  encargado: 'Administra la operación y el equipo de una única sucursal.',
  mozo: 'Gestiona carta y pedidos; puede ver mesas sin modificar mesas ni QR.',
  cocina: 'Ve la carta y opera únicamente los estados de ítems del KDS.',
};

export default function EquipoSection({ embedded = false, sesion = null }: { embedded?: boolean; sesion?: SesionUsuario | null }) {
  const toast = useToast();
  const confirmar = useConfirm();
  const [equipo, setEquipo] = useState<EquipoAPI>({ usuarios: [], invitaciones: [] });
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [form, setForm] = useState<{ email: string; rol: RolInvitable; sucursal_id: string }>({ email: '', rol: 'mozo', sucursal_id: '' });
  const [loading, setLoading] = useState(true);
  const [guardando, setGuardando] = useState(false);
  const [error, setError] = useState('');
  const [entrega, setEntrega] = useState<EntregaInvitacion | null>(null);

	const esDuenio = !sesion || sesion.rol === 'admin';
  const cargar = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [respuesta, sedes] = await Promise.all([api.listarUsuarios(), api.listarSucursales()]);
      setEquipo(respuesta);
      setSucursales(sedes);
      setForm((actual) => ({ ...actual, sucursal_id: actual.sucursal_id || sesion?.sucursal_id || sedes[0]?.id || '' }));
    } catch (err: unknown) {
      setError(getErrorMessage(err, 'No pudimos cargar el equipo.'));
    } finally { setLoading(false); }
  }, [sesion?.sucursal_id]);

  useEffect(() => {
    void Promise.resolve().then(cargar);
  }, [cargar]);

  const invitar = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!form.email.includes('@')) { setError('Ingresá un email válido.'); return; }
    if (esDuenio && !form.sucursal_id) { setError('Elegí una sucursal.'); return; }
    setGuardando(true); setError(''); setEntrega(null);
    try {
      const respuesta = await api.invitarUsuario({ email: form.email, rol: form.rol, sucursal_id: esDuenio ? form.sucursal_id : undefined });
      setEquipo((actual) => ({ ...actual, invitaciones: [respuesta.invitacion, ...actual.invitaciones] }));
      setForm((actual) => ({ ...actual, email: '' }));
      setEntrega({ link: respuesta.invitacion_link, email: respuesta.invitacion.email, emailEnviado: respuesta.email_enviado, advertencia: respuesta.advertencia_email });
      if (respuesta.email_enviado) toast.success(`Invitación enviada a ${respuesta.invitacion.email}.`);
      else toast.info(respuesta.advertencia_email ?? 'La invitación se creó. Compartí el enlace manualmente.');
    } catch (err: unknown) { const mensaje = getErrorMessage(err, 'No pudimos enviar la invitación.'); setError(mensaje); toast.error(mensaje); }
    finally { setGuardando(false); }
  };

  const cambiarEstado = async (miembro: UsuarioAPI) => {
    const estado = miembro.estado === 'activo' ? 'bloqueado' : 'activo';
    try {
      const actualizado = await api.actualizarUsuario(miembro.id, { estado });
      setEquipo((actual) => ({ ...actual, usuarios: actual.usuarios.map((u) => u.id === miembro.id ? actualizado : u) }));
      toast.success(estado === 'bloqueado' ? 'Acceso bloqueado y sesiones revocadas.' : 'Acceso reactivado.');
    } catch (err: unknown) { toast.error(getErrorMessage(err, 'No pudimos cambiar el acceso.')); }
  };

  const eliminar = async (miembro: UsuarioAPI) => {
    const ok = await confirmar({ titulo: 'Eliminar miembro', mensaje: `Se eliminará el acceso de ${miembro.nombre}. Los pedidos históricos conservarán su nombre.`, labelAceptar: 'Eliminar', variante: 'danger' });
    if (!ok) return;
    try { await api.eliminarUsuario(miembro.id); setEquipo((actual) => ({ ...actual, usuarios: actual.usuarios.filter((u) => u.id !== miembro.id) })); toast.success('Miembro eliminado.'); }
    catch (err: unknown) { toast.error(getErrorMessage(err, 'No pudimos eliminar el miembro.')); }
  };

  const cancelar = async (invitacion: InvitacionEquipoAPI) => {
    try { await api.cancelarInvitacion(invitacion.id); setEquipo((actual) => ({ ...actual, invitaciones: actual.invitaciones.filter((i) => i.id !== invitacion.id) })); toast.success('Invitación cancelada.'); }
    catch (err: unknown) { toast.error(getErrorMessage(err, 'No pudimos cancelar la invitación.')); }
  };

  const reenviar = async (invitacion: InvitacionEquipoAPI) => {
    try {
      const respuesta = await api.reenviarInvitacion(invitacion.id);
      setEquipo((actual) => ({ ...actual, invitaciones: actual.invitaciones.map((i) => i.id === invitacion.id ? respuesta.invitacion : i) }));
      setEntrega({ link: respuesta.invitacion_link, email: respuesta.invitacion.email, emailEnviado: respuesta.email_enviado, advertencia: respuesta.advertencia_email });
      if (respuesta.email_enviado) toast.success(`Enviamos un enlace nuevo a ${respuesta.invitacion.email}; el anterior dejó de ser válido.`);
      else toast.info(respuesta.advertencia_email ?? 'Generamos un enlace nuevo. Compartilo manualmente.');
    } catch (err: unknown) { toast.error(getErrorMessage(err, 'No pudimos reenviar la invitación.')); }
  };

  return (
    <div className={`${embedded ? 'space-y-20' : 'h-full space-y-24 overflow-y-auto bg-ghost-fog/45 p-16 sm:p-24 md:p-32'} font-inter`}>
      <header>
        <p className="text-11 font-mono uppercase tracking-[0.14em] text-sage-green">Acceso del equipo</p>
        <h2 className="mt-4 text-24 font-semibold text-ash-graphite sm:text-32">Personas e invitaciones</h2>
        <p className="mt-4 text-13 text-sage-green">Cada miembro queda asociado a una sucursal y entra siempre con Google.</p>
      </header>

      <form onSubmit={invitar} className="space-y-12 rounded-xl border border-concrete bg-canvas-white p-16 sm:p-20">
        <div className="grid gap-10 md:grid-cols-[1fr_180px_1fr_auto]">
          <input type="email" required value={form.email} onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))} placeholder="persona@negocio.com" className="h-44 rounded-lg border border-concrete px-12 text-13 outline-none focus:border-plain-green" />
          <select value={form.rol} onChange={(e) => setForm((f) => ({ ...f, rol: e.target.value as RolInvitable }))} className="h-44 rounded-lg border border-concrete bg-white px-10 text-13">
            {esDuenio && <option value="encargado">Encargado</option>}
            <option value="mozo">Mozo</option><option value="cocina">Cocinero</option>
          </select>
          {esDuenio ? <select value={form.sucursal_id} onChange={(e) => setForm((f) => ({ ...f, sucursal_id: e.target.value }))} className="h-44 rounded-lg border border-concrete bg-white px-10 text-13"><option value="">Sucursal…</option>{sucursales.map((s) => <option key={s.id} value={s.id}>{s.nombre}</option>)}</select> : <div className="flex h-44 items-center rounded-lg border border-concrete bg-ghost-fog px-12 text-12 text-sage-green">Sucursal asignada automáticamente</div>}
          <button disabled={guardando} className="h-44 rounded-lg bg-plain-green px-16 text-13 font-semibold text-white disabled:opacity-50">{guardando ? 'Enviando…' : 'Invitar'}</button>
        </div>
        <p className="text-12 text-sage-green">{DESCRIPCIONES[form.rol]} La invitación vence en 48 horas y se usa una sola vez.</p>
        {error && <p role="alert" className="text-12 text-alert-red">{error}</p>}
        {entrega && <div className={`rounded-lg border p-12 ${entrega.emailEnviado ? 'border-success/40 bg-success/10' : 'border-[#d99b36]/40 bg-[#fff8ea]'}`}>
          <p className={`text-12 font-semibold ${entrega.emailEnviado ? 'text-[#087645]' : 'text-[#8a5a10]'}`}>
            {entrega.emailEnviado ? `Invitación enviada por email a ${entrega.email}.` : entrega.advertencia}
          </p>
          <p className="mt-4 text-11 text-sage-green">También podés compartir este enlace manualmente:</p>
          <div className="mt-8 flex min-w-0 flex-col gap-8 sm:flex-row">
            <code className="min-w-0 flex-1 truncate rounded-md border border-concrete bg-white px-10 py-9 text-11 text-ash-graphite">{entrega.link}</code>
            <button type="button" onClick={() => void navigator.clipboard.writeText(entrega.link).then(() => toast.success('Enlace copiado.')).catch(() => toast.error('No pudimos copiar el enlace.'))} className="h-36 shrink-0 rounded-md bg-ash-graphite px-14 text-11 font-semibold text-white">Copiar enlace</button>
          </div>
        </div>}
      </form>

      <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
        <div className="border-b border-concrete px-16 py-12"><h3 className="text-15 font-semibold">Miembros activos y bloqueados</h3></div>
        {loading ? <div className="space-y-12 p-16"><Skeleton className="h-44 w-full"/><Skeleton className="h-44 w-full"/></div> : equipo.usuarios.filter((u) => u.rol !== 'admin').length === 0 ? <EmptyState icon="group" title="Todavía no hay miembros" description="Las personas aparecerán aquí cuando acepten la invitación." /> : <div className="divide-y divide-ghost-fog">{equipo.usuarios.filter((u) => u.rol !== 'admin').map((u) => <article key={u.id} className="flex flex-col gap-10 px-16 py-14 sm:flex-row sm:items-center sm:justify-between"><div><p className="text-14 font-semibold text-ash-graphite">{u.nombre}</p><p className="text-12 text-sage-green">{u.email} · {u.rol} · {u.sucursal_nombre}</p></div><div className="flex gap-8"><button onClick={() => void cambiarEstado(u)} className="h-40 rounded-lg border border-concrete px-12 text-11 font-semibold">{u.estado === 'activo' ? 'Bloquear' : 'Reactivar'}</button><button onClick={() => void eliminar(u)} className="h-40 rounded-lg px-10 text-alert-red"><span className="material-symbols-outlined text-18">delete</span></button></div></article>)}</div>}
      </section>

      <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
        <div className="border-b border-concrete px-16 py-12"><h3 className="text-15 font-semibold">Invitaciones pendientes</h3></div>
        {equipo.invitaciones.length === 0 ? <EmptyState compact icon="mark_email_unread" title="No hay invitaciones pendientes" /> : <div className="divide-y divide-ghost-fog">{equipo.invitaciones.map((i) => <article key={i.id} className="flex flex-col gap-10 px-16 py-14 sm:flex-row sm:items-center sm:justify-between"><div><p className="text-14 font-semibold">{i.email}</p><p className="text-12 text-sage-green">{i.rol} · {i.sucursal} · vence {new Date(i.expires_at).toLocaleString('es-AR')}</p></div><div className="flex gap-8"><button onClick={() => void reenviar(i)} className="h-40 rounded-lg border border-concrete px-12 text-11 font-semibold">Reenviar</button><button onClick={() => void cancelar(i)} className="h-40 rounded-lg px-12 text-11 text-alert-red">Cancelar</button></div></article>)}</div>}
      </section>
    </div>
  );
}
