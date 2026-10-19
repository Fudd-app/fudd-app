"use client";
import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { api, UsuarioAPI, Sucursal, Tenant, EstadoUsuario, getErrorMessage } from "@/lib/api";
import { EmptyState, Skeleton, useToast, useConfirm } from "@/components/ui";

type RolUsuario = 'admin' | 'encargado' | 'mozo' | 'cocina';
type RolInvitable = 'encargado' | 'mozo' | 'cocina';
type ModoAlta = 'credenciales' | 'magic';

const ROL_DESCRIPTIONS: Record<RolInvitable, string> = {
  encargado: 'Puede gestionar la operación de una sucursal: carta, mesas y pedidos activos.',
  mozo: 'Puede ver pedidos en vivo, avanzar estados y atender solicitudes de cuenta.',
  cocina: 'Acceso directo a la pantalla Kitchen Display System (KDS) para despachar comandas.',
};

const ROL_LABELS: Record<RolUsuario, string> = {
  admin: 'Admin',
  encargado: 'Encargado',
  mozo: 'Mozo / Salón',
  cocina: 'Cocina (KDS)',
};

function generarClaveSegura(): string {
  const prefijos = ['Mesa', 'Salon', 'Cocina', 'Turno', 'Clave', 'Staff'];
  const prefijo = prefijos[Math.floor(Math.random() * prefijos.length)];
  const numeros = Math.floor(1000 + Math.random() * 9000);
  return `${prefijo}${numeros}`;
}

function sugerirUsername(nombre: string, rol: RolInvitable): string {
  const limpio = nombre
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");
  if (!limpio) return "";
  const partes = limpio.split("_").filter(Boolean);
  const primero = partes[0] || limpio;
  return `${rol}_${primero}`.slice(0, 28);
}

export default function EquipoSection({ embedded = false }: { embedded?: boolean }) {
  const toast = useToast();
  const confirmar = useConfirm();
  const [equipo, setEquipo] = useState<UsuarioAPI[]>([]);
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loading, setLoading] = useState(true);
  const [actualizandoId, setActualizandoId] = useState<string | null>(null);
  const [mostrarArchivados, setMostrarArchivados] = useState(false);

  // Modo de alta: credenciales (US-83) o magic link
  const [modoAlta, setModoAlta] = useState<ModoAlta>('credenciales');
  const [nombre, setNombre] = useState('');
  const [username, setUsername] = useState('');
  const [usernameEditado, setUsernameEditado] = useState(false);
  const [email, setEmail] = useState('');
  const [rol, setRol] = useState<RolInvitable>('mozo');
  const [sucursalId, setSucursalId] = useState<string>('');
  const [password, setPassword] = useState(() => generarClaveSegura());
  const [mostrarPassword, setMostrarPassword] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  // Resultado de creación de credenciales o magic link
  const [credencialesCreadas, setCredencialesCreadas] = useState<{
    nombre: string;
    username: string;
    password: string;
    rol: RolInvitable;
  } | null>(null);
  const [invitacionLink, setInvitacionLink] = useState('');
  const [copiado, setCopiado] = useState(false);

  // Modal de blanqueo de clave
  const [usuarioReset, setUsuarioReset] = useState<UsuarioAPI | null>(null);
  const [nuevaPasswordReset, setNuevaPasswordReset] = useState('');
  const [mostrarPasswordReset, setMostrarPasswordReset] = useState(true);
  const [guardandoReset, setGuardandoReset] = useState(false);

  const cargarDatos = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const [usuariosRes, sucursalesRes, tenantRes] = await Promise.allSettled([
        api.listarUsuarios(),
        api.listarSucursales(),
        api.obtenerMiTenant(),
      ]);

      if (usuariosRes.status === 'fulfilled' && Array.isArray(usuariosRes.value)) {
        setEquipo(usuariosRes.value);
      } else {
        setEquipo([]);
      }

      if (sucursalesRes.status === 'fulfilled' && Array.isArray(sucursalesRes.value)) {
        setSucursales(sucursalesRes.value);
        if (sucursalesRes.value.length > 0) {
          setSucursalId(prev => prev || sucursalesRes.value[0].id);
        }
      }

      if (tenantRes.status === 'fulfilled' && tenantRes.value) {
        setTenant(tenantRes.value);
      }
    } catch (err: unknown) {
      setError(getErrorMessage(err, 'Error al cargar datos del equipo.'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      void cargarDatos();
    }, 0);
    return () => window.clearTimeout(timeoutId);
  }, [cargarDatos]);

  const handleNombreChange = (val: string) => {
    setNombre(val);
    if (!usernameEditado) {
      setUsername(sugerirUsername(val, rol));
    }
  };

  const handleRolFormChange = (nuevoRol: RolInvitable) => {
    setRol(nuevoRol);
    if (!usernameEditado && nombre.trim()) {
      setUsername(sugerirUsername(nombre, nuevoRol));
    }
  };

  const portalUrl = typeof window !== 'undefined' && tenant?.slug
    ? `${window.location.origin}/${tenant.slug}/login`
    : tenant?.slug
      ? `/${tenant.slug}/login`
      : '';

  const copiarTexto = (texto: string, mensajeToast: string) => {
    navigator.clipboard.writeText(texto);
    setCopiado(true);
    toast.success(mensajeToast);
    setTimeout(() => setCopiado(false), 2500);
  };

  const armarTextoCredenciales = (user: string, pass: string, nombreStaff?: string) => {
    const urlLinea = portalUrl ? `\nPortal de ingreso: ${portalUrl}` : '';
    return `Hola${nombreStaff ? ` ${nombreStaff}` : ''}! Tus datos de acceso a Mesa CLICK (${tenant?.nombre || 'Local'}):${urlLinea}\nUsuario: ${user}\nClave: ${pass}`;
  };

  const handleCrearStaff = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setCredencialesCreadas(null);
    setInvitacionLink('');

    if (!nombre.trim()) {
      setError('El nombre completo es obligatorio.');
      return;
    }

    if (modoAlta === 'credenciales') {
      const cleanUser = username.trim().toLowerCase().replace(/^@/, '');
      if (cleanUser.length < 3) {
        setError('El nombre de usuario debe tener al menos 3 caracteres.');
        return;
      }
      if (password.length < 8 || !/[a-zA-Z]/.test(password) || !/[0-9]/.test(password)) {
        setError('La contraseña debe tener al menos 8 caracteres e incluir letras y números.');
        return;
      }

      setSubmitting(true);
      try {
        const creado = await api.crearUsuarioStaff({
          nombre: nombre.trim(),
          username: cleanUser,
          email: email.trim() || undefined,
          password,
          rol,
          sucursal_id: sucursalId || undefined,
        });
        setEquipo(prev => [...prev, creado]);
        setCredencialesCreadas({
          nombre: creado.nombre,
          username: creado.username || cleanUser,
          password,
          rol,
        });
        setNombre('');
        setUsername('');
        setUsernameEditado(false);
        setEmail('');
        setPassword(generarClaveSegura());
        toast.success(`Usuario @${creado.username || cleanUser} creado correctamente.`);
      } catch (err: unknown) {
        const msg = getErrorMessage(err, 'No se pudo crear el usuario de equipo.');
        setError(msg);
        toast.error(msg);
      } finally {
        setSubmitting(false);
      }
    } else {
      if (!email.trim() || !email.includes('@')) {
        setError('Ingresá un correo electrónico válido para enviar el Magic Link.');
        return;
      }
      setSubmitting(true);
      try {
        const resp = await api.invitarUsuario({
          nombre: nombre.trim(),
          email: email.trim(),
          rol,
        });
        if (resp.usuario) {
          setEquipo(prev => [...prev, resp.usuario]);
        }
        const linkInvitacion = resp.magic_link || resp.url_invitacion;
        if (linkInvitacion) {
          setInvitacionLink(linkInvitacion);
        }
        setNombre('');
        setEmail('');
        toast.success('Invitación generada. Compartí el magic link con el nuevo miembro.');
      } catch (err: unknown) {
        const msg = getErrorMessage(err, 'Error al invitar al usuario.');
        setError(msg);
        toast.error(msg);
      } finally {
        setSubmitting(false);
      }
    }
  };

  const handleCambiarRol = async (id: string, nuevoRol: RolUsuario) => {
    try {
      setActualizandoId(id);
      const actualizado = await api.actualizarUsuario(id, { rol: nuevoRol });
      setEquipo(prev => prev.map(u => (u.id === id ? { ...u, ...actualizado, rol: nuevoRol } : u)));
      toast.success('Rol actualizado.');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, 'No se pudo actualizar el rol.'));
    } finally {
      setActualizandoId(null);
    }
  };

  const handleCambiarSucursal = async (id: string, nuevaSucursalId: string) => {
    try {
      setActualizandoId(id);
      const actualizado = await api.actualizarUsuario(id, {
        sucursal_id: nuevaSucursalId,
      });
      setEquipo(prev => prev.map(u => (u.id === id ? { ...u, ...actualizado } : u)));
      toast.success('Sucursal asignada actualizada.');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, 'No se pudo actualizar la sucursal.'));
    } finally {
      setActualizandoId(null);
    }
  };

  const handleCambiarEstado = async (u: UsuarioAPI, nuevoEstado: EstadoUsuario) => {
    if (nuevoEstado === 'bloqueado') {
      const ok = await confirmar({
        titulo: `Bloquear acceso de ${u.nombre}`,
        mensaje: `¿Querés bloquear temporalmente a ${u.nombre}? No podrá iniciar sesión en el local hasta que lo reactives.`,
        labelAceptar: 'Sí, bloquear acceso',
        variante: 'danger',
      });
      if (!ok) return;
    }

    try {
      setActualizandoId(u.id);
      const actualizado = await api.cambiarEstadoUsuario(u.id, nuevoEstado);
      setEquipo(prev => prev.map(item => (item.id === u.id ? { ...item, ...actualizado, estado: nuevoEstado } : item)));
      const etiquetas: Record<EstadoUsuario, string> = {
        activo: 'Acceso reactivado.',
        bloqueado: 'Usuario bloqueado.',
        archivado: 'Usuario archivado.',
      };
      toast.success(etiquetas[nuevoEstado]);
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, 'No se pudo cambiar el estado del usuario.'));
    } finally {
      setActualizandoId(null);
    }
  };

  const handleEliminarUsuario = async (u: UsuarioAPI) => {
    const ok = await confirmar({
      titulo: 'Dar de baja miembro',
      mensaje: `¿Eliminás a ${u.nombre} del equipo? Perderá el acceso inmediatamente, pero se conservará el historial de los pedidos que atendió.`,
      labelAceptar: 'Sí, dar de baja',
      variante: 'danger',
    });
    if (!ok) return;
    try {
      await api.eliminarUsuario(u.id);
      setEquipo(prev => prev.filter(item => item.id !== u.id));
      toast.success('Miembro dado de baja correctamente.');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, 'No se pudo eliminar el usuario.'));
    }
  };

  const abrirModalResetPassword = (u: UsuarioAPI) => {
    setUsuarioReset(u);
    setNuevaPasswordReset(generarClaveSegura());
    setMostrarPasswordReset(true);
  };

  const handleConfirmarResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!usuarioReset) return;
    if (nuevaPasswordReset.length < 8 || !/[a-zA-Z]/.test(nuevaPasswordReset) || !/[0-9]/.test(nuevaPasswordReset)) {
      toast.error('La clave debe tener al menos 8 caracteres con letras y números.');
      return;
    }
    setGuardandoReset(true);
    try {
      await api.resetearPasswordUsuario(usuarioReset.id, nuevaPasswordReset);
      const userIdentifier = usuarioReset.username || usuarioReset.email || usuarioReset.nombre;
      copiarTexto(
        armarTextoCredenciales(userIdentifier, nuevaPasswordReset, usuarioReset.nombre),
        'Clave actualizada y copiada al portapapeles.'
      );
      setUsuarioReset(null);
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, 'No se pudo blanquear la contraseña.'));
    } finally {
      setGuardandoReset(false);
    }
  };

  const archivadosCount = equipo.filter(u => u.estado === 'archivado').length;
  const equipoVisible = equipo.filter(u => mostrarArchivados || u.estado !== 'archivado');

  const renderEstadoBadge = (estado?: EstadoUsuario) => {
    const est = estado || 'activo';
    if (est === 'bloqueado') {
      return (
        <span className="inline-flex items-center gap-4 rounded-full border border-amber-300 bg-amber-50 px-8 py-2 text-10 font-semibold text-amber-800">
          <span className="h-6 w-6 rounded-full bg-amber-500" />
          Bloqueado
        </span>
      );
    }
    if (est === 'archivado') {
      return (
        <span className="inline-flex items-center gap-4 rounded-full border border-concrete bg-ghost-fog px-8 py-2 text-10 font-semibold text-sage-green">
          Archivado
        </span>
      );
    }
    return (
      <span className="inline-flex items-center gap-4 rounded-full border border-emerald-200 bg-emerald-50 px-8 py-2 text-10 font-semibold text-emerald-800">
        <span className="h-6 w-6 rounded-full bg-emerald-500" />
        Activo
      </span>
    );
  };

  return (
    <div className={`${embedded ? "space-y-20" : "h-full space-y-24 overflow-y-auto bg-ghost-fog/45 p-16 sm:p-24 md:p-32"} font-inter`}>
      {/* Banner del Portal Único de Acceso del Local (US-84) */}
      {tenant?.slug && (
        <section className="overflow-hidden rounded-xl border border-plain-green/25 bg-plain-green/5 p-16 sm:p-20">
          <div className="flex flex-col justify-between gap-12 sm:flex-row sm:items-center">
            <div className="space-y-4">
              <div className="flex items-center gap-8">
                <span className="material-symbols-outlined text-18 text-plain-green">badge</span>
                <p className="text-14 font-semibold text-ash-graphite">
                  Portal de ingreso exclusivo para tu equipo
                </p>
              </div>
              <p className="text-12 text-sage-green">
                Compartí este link en las tablets de cocina o celulares de tus mozos para que ingresen directo con su usuario y clave:
              </p>
              <p className="font-mono text-12 font-semibold text-plain-green break-all">
                {portalUrl}
              </p>
            </div>
            <div className="flex shrink-0 flex-wrap items-center gap-8">
              <button
                type="button"
                onClick={() => copiarTexto(portalUrl, 'Link del portal copiado al portapapeles.')}
                className="inline-flex h-40 items-center gap-6 rounded-lg border border-plain-green bg-canvas-white px-12 text-12 font-semibold text-plain-green transition-colors hover:bg-plain-green hover:text-canvas-white"
              >
                <span className="material-symbols-outlined text-16">content_copy</span>
                Copiar link
              </button>
              <Link
                href={`/${tenant.slug}/login`}
                target="_blank"
                className="inline-flex h-40 items-center gap-6 rounded-lg bg-plain-green px-12 text-12 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted"
              >
                Abrir portal
                <span className="material-symbols-outlined text-15">open_in_new</span>
              </Link>
            </div>
          </div>
        </section>
      )}

      {/* Listado de Miembros del Equipo */}
      <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
        <div className="flex flex-wrap items-center justify-between gap-12 border-b border-concrete/70 px-16 py-14 sm:px-20">
          <div>
            <p className="text-15 font-semibold tracking-[-0.01em] text-ash-graphite">Miembros del equipo</p>
            <p className="text-11 text-sage-green">
              {equipoVisible.length} {equipoVisible.length === 1 ? "miembro visible" : "miembros visibles"}
              {loading ? " (cargando...)" : ""}
            </p>
          </div>
          {archivadosCount > 0 && (
            <button
              type="button"
              onClick={() => setMostrarArchivados(prev => !prev)}
              className="inline-flex h-36 items-center gap-6 rounded-lg border border-concrete bg-ghost-fog px-10 text-11 font-medium text-ash-graphite hover:border-stone"
            >
              <span className="material-symbols-outlined text-15">
                {mostrarArchivados ? 'visibility_off' : 'inventory_2'}
              </span>
              {mostrarArchivados ? 'Ocultar archivados' : `Ver archivados (${archivadosCount})`}
            </button>
          )}
        </div>

        {loading && (
          <div className="divide-y divide-ghost-fog">
            {[1, 2, 3].map(i => (
              <div key={i} className="flex items-center justify-between px-16 py-14 sm:px-20">
                <div className="space-y-6">
                  <Skeleton className="h-12 w-48" />
                  <Skeleton className="h-10 w-64" />
                </div>
                <Skeleton className="h-20 w-24" />
              </div>
            ))}
          </div>
        )}

        {!loading && equipoVisible.length > 0 && (
          <div className="divide-y divide-ghost-fog">
            {equipoVisible.map(u => {
              const esAdmin = u.rol === 'admin';
              const est = u.estado || 'activo';
              return (
                <div
                  key={u.id}
                  className={`flex flex-col gap-12 px-16 py-14 sm:flex-row sm:items-center sm:justify-between sm:px-20 ${
                    est !== 'activo' ? 'bg-ghost-fog/40 opacity-80' : ''
                  }`}
                >
                  <div className="min-w-0 space-y-4">
                    <div className="flex flex-wrap items-center gap-8">
                      <p className="truncate text-14 font-semibold text-ash-graphite">{u.nombre}</p>
                      {renderEstadoBadge(u.estado)}
                      {esAdmin && (
                        <span className="inline-flex items-center rounded-full border border-concrete bg-ghost-fog px-8 py-2 text-10 font-semibold text-sage-green">
                          {ROL_LABELS.admin}
                        </span>
                      )}
                    </div>

                    <div className="flex flex-wrap items-center gap-10 text-12 text-sage-green">
                      {u.username && (
                        <button
                          type="button"
                          onClick={() => copiarTexto(u.username!, `Usuario @${u.username} copiado.`)}
                          className="inline-flex items-center gap-4 rounded bg-ghost-fog px-6 py-2 font-mono text-11 font-semibold text-ash-graphite hover:bg-concrete/60"
                          title="Clic para copiar nombre de usuario"
                        >
                          @{u.username}
                          <span className="material-symbols-outlined text-13 text-sage-green">content_copy</span>
                        </button>
                      )}
                      {u.email && (
                        <span className="truncate">{u.email}</span>
                      )}
                      {u.sucursal_nombre && (
                        <span className="inline-flex items-center gap-4 text-11 text-sage-green">
                          <span className="material-symbols-outlined text-14">storefront</span>
                          {u.sucursal_nombre}
                        </span>
                      )}
                    </div>
                  </div>

                  {!esAdmin && (
                    <div className="flex flex-wrap items-center justify-between gap-8 sm:justify-end">
                      {/* Selector de Rol */}
                      <select
                        value={u.rol}
                        disabled={actualizandoId === u.id}
                        onChange={(e) => void handleCambiarRol(u.id, e.target.value as RolUsuario)}
                        aria-label={`Rol de ${u.nombre}`}
                        className="h-38 cursor-pointer rounded-lg border border-concrete bg-canvas-white px-10 text-11 font-medium text-ash-graphite outline-none focus:border-plain-green disabled:opacity-50"
                      >
                        <option value="encargado">Encargado</option>
                        <option value="mozo">Mozo / Salón</option>
                        <option value="cocina">Cocina (KDS)</option>
                      </select>

                      {/* Selector de Sucursal */}
                      {sucursales.length > 0 && (
                        <select
                          value={u.sucursal_id || ''}
                          disabled={actualizandoId === u.id}
                          onChange={(e) => void handleCambiarSucursal(u.id, e.target.value)}
                          aria-label={`Sucursal de ${u.nombre}`}
                          className="h-38 cursor-pointer rounded-lg border border-concrete bg-canvas-white px-10 text-11 text-sage-green outline-none focus:border-plain-green disabled:opacity-50"
                        >
                          <option value="">Todas las sucursales</option>
                          {sucursales.map(s => (
                            <option key={s.id} value={s.id}>{s.nombre}</option>
                          ))}
                        </select>
                      )}

                      {/* Acciones de ciclo de vida y seguridad */}
                      <div className="flex items-center gap-4">
                        <button
                          type="button"
                          onClick={() => abrirModalResetPassword(u)}
                          className="flex h-38 w-38 items-center justify-center rounded-lg border border-concrete bg-canvas-white text-ash-graphite transition-colors hover:bg-ghost-fog"
                          title="Blanquear / Cambiar contraseña"
                          aria-label={`Cambiar contraseña de ${u.nombre}`}
                        >
                          <span className="material-symbols-outlined text-18">key</span>
                        </button>

                        {est === 'bloqueado' ? (
                          <button
                            type="button"
                            disabled={actualizandoId === u.id}
                            onClick={() => void handleCambiarEstado(u, 'activo')}
                            className="flex h-38 w-38 items-center justify-center rounded-lg border border-emerald-300 bg-emerald-50 text-emerald-700 transition-colors hover:bg-emerald-100"
                            title="Desbloquear acceso"
                            aria-label={`Desbloquear a ${u.nombre}`}
                          >
                            <span className="material-symbols-outlined text-18">lock_open</span>
                          </button>
                        ) : (
                          <button
                            type="button"
                            disabled={actualizandoId === u.id}
                            onClick={() => void handleCambiarEstado(u, 'bloqueado')}
                            className="flex h-38 w-38 items-center justify-center rounded-lg border border-concrete bg-canvas-white text-amber-700 transition-colors hover:bg-amber-50"
                            title="Bloquear acceso temporalmente"
                            aria-label={`Bloquear a ${u.nombre}`}
                          >
                            <span className="material-symbols-outlined text-18">block</span>
                          </button>
                        )}

                        {est === 'archivado' ? (
                          <button
                            type="button"
                            disabled={actualizandoId === u.id}
                            onClick={() => void handleCambiarEstado(u, 'activo')}
                            className="flex h-38 w-38 items-center justify-center rounded-lg border border-concrete bg-canvas-white text-ash-graphite transition-colors hover:bg-ghost-fog"
                            title="Restaurar miembro archivado"
                            aria-label={`Restaurar a ${u.nombre}`}
                          >
                            <span className="material-symbols-outlined text-18">unarchive</span>
                          </button>
                        ) : (
                          <button
                            type="button"
                            disabled={actualizandoId === u.id}
                            onClick={() => void handleCambiarEstado(u, 'archivado')}
                            className="flex h-38 w-38 items-center justify-center rounded-lg border border-concrete bg-canvas-white text-sage-green transition-colors hover:bg-ghost-fog"
                            title="Archivar miembro (fin de temporada / inactivo)"
                            aria-label={`Archivar a ${u.nombre}`}
                          >
                            <span className="material-symbols-outlined text-18">inventory_2</span>
                          </button>
                        )}

                        <button
                          type="button"
                          onClick={() => void handleEliminarUsuario(u)}
                          className="flex h-38 w-38 items-center justify-center rounded-lg text-alert-red transition-colors hover:bg-warm-pink/20"
                          title="Dar de baja"
                          aria-label={`Dar de baja a ${u.nombre}`}
                        >
                          <span className="material-symbols-outlined text-18">delete</span>
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {equipoVisible.length === 0 && !loading && (
          <EmptyState
            icon="group_add"
            title="No hay miembros activos en el equipo"
            description="Creá credenciales de acceso para tus mozos, encargados y personal de cocina."
            actionLabel="Agregar miembro"
            onAction={() => document.getElementById('form-invitar')?.scrollIntoView({ behavior: 'smooth' })}
          />
        )}
      </section>

      {/* Formulario de Alta de Equipo (US-83) */}
      <section id="form-invitar" className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
        <div className="flex flex-col gap-12 border-b border-concrete/70 px-16 py-14 sm:flex-row sm:items-center sm:justify-between sm:px-20">
          <div>
            <p className="text-15 font-semibold tracking-[-0.01em] text-ash-graphite">Agregar miembro al equipo</p>
            <p className="text-12 text-sage-green">
              Creá un usuario y clave listos para usar o enviá una invitación por correo electrónico.
            </p>
          </div>
          <div className="inline-flex rounded-lg border border-concrete bg-ghost-fog p-3">
            <button
              type="button"
              onClick={() => { setModoAlta('credenciales'); setError(''); }}
              className={`rounded-md px-12 py-6 text-12 font-semibold transition-colors ${
                modoAlta === 'credenciales'
                  ? 'bg-canvas-white text-ash-graphite shadow-xs'
                  : 'text-sage-green hover:text-ash-graphite'
              }`}
            >
              Usuario y clave
            </button>
            <button
              type="button"
              onClick={() => { setModoAlta('magic'); setError(''); }}
              className={`rounded-md px-12 py-6 text-12 font-semibold transition-colors ${
                modoAlta === 'magic'
                  ? 'bg-canvas-white text-ash-graphite shadow-xs'
                  : 'text-sage-green hover:text-ash-graphite'
              }`}
            >
              Magic Link (Email)
            </button>
          </div>
        </div>

        <form onSubmit={handleCrearStaff} className="space-y-16 p-16 sm:p-20">
          {modoAlta === 'credenciales' ? (
            <>
              <div className="rounded-lg border border-concrete bg-ghost-fog/70 p-12 text-12 leading-normal text-sage-green">
                Generá un <strong>nombre de usuario</strong> (ej: <code className="font-mono text-ash-graphite">mozo_juan</code>) y una <strong>contraseña</strong> para que tu personal ingrese desde el portal de tu negocio sin depender de su correo personal.
              </div>

              <div className="grid grid-cols-1 gap-12 sm:grid-cols-2">
                <label className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">Nombre completo *</span>
                  <input
                    className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                    placeholder="Ej: Juan Pérez"
                    value={nombre}
                    onChange={e => handleNombreChange(e.target.value)}
                  />
                </label>

                <label className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">Nombre de usuario (Alias) *</span>
                  <div className="flex h-44 items-center rounded-lg border border-concrete bg-canvas-white px-12 focus-within:border-plain-green">
                    <span className="mr-4 font-mono text-13 text-sage-green">@</span>
                    <input
                      className="h-full flex-1 bg-transparent font-mono text-13 outline-none"
                      placeholder="mozo_juan"
                      value={username}
                      onChange={e => {
                        setUsernameEditado(true);
                        setUsername(e.target.value.toLowerCase().replace(/[^a-z0-9._-]/g, ''));
                      }}
                    />
                  </div>
                </label>
              </div>

              <div className="grid grid-cols-1 gap-12 sm:grid-cols-3">
                <label className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">Rol operativo *</span>
                  <select
                    className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                    value={rol}
                    onChange={e => handleRolFormChange(e.target.value as RolInvitable)}
                  >
                    <option value="mozo">Mozo / Recepcionista</option>
                    <option value="cocina">Cocina (KDS)</option>
                    <option value="encargado">Encargado de Sucursal</option>
                  </select>
                </label>

                <label className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">Sucursal asignada</span>
                  <select
                    className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                    value={sucursalId}
                    onChange={e => setSucursalId(e.target.value)}
                  >
                    <option value="">Todas las sucursales</option>
                    {sucursales.map(s => (
                      <option key={s.id} value={s.id}>{s.nombre}</option>
                    ))}
                  </select>
                </label>

                <label className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">Email (opcional)</span>
                  <input
                    type="email"
                    className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                    placeholder="Opcional para notificaciones"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                  />
                </label>
              </div>

              <div className="space-y-6">
                <span className="text-12 font-semibold text-ash-graphite">Contraseña inicial *</span>
                <div className="flex flex-col gap-8 sm:flex-row sm:items-center">
                  <div className="flex h-44 flex-1 items-center rounded-lg border border-concrete bg-canvas-white px-12 focus-within:border-plain-green">
                    <input
                      type={mostrarPassword ? "text" : "password"}
                      className="h-full flex-1 bg-transparent font-mono text-13 outline-none"
                      placeholder="Mínimo 8 caracteres (letras y números)"
                      value={password}
                      onChange={e => setPassword(e.target.value)}
                    />
                    <button
                      type="button"
                      onClick={() => setMostrarPassword(p => !p)}
                      className="ml-8 text-11 font-semibold text-sage-green hover:text-ash-graphite"
                    >
                      {mostrarPassword ? "Ocultar" : "Ver"}
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setPassword(generarClaveSegura());
                      setMostrarPassword(true);
                    }}
                    className="inline-flex h-44 shrink-0 items-center justify-center gap-6 rounded-lg border border-concrete bg-ghost-fog px-14 text-12 font-semibold text-ash-graphite transition-colors hover:border-stone"
                  >
                    <span className="material-symbols-outlined text-16">autorenew</span>
                    Autogenerar clave segura
                  </button>
                </div>
              </div>
            </>
          ) : (
            <>
              <div className="rounded-lg border border-concrete bg-ghost-fog/70 p-12 text-12 leading-normal text-sage-green">
                Invitá a quienes atienden la sucursal para que entren con su propio enlace mágico por correo electrónico.
              </div>
              <div className="grid grid-cols-1 gap-12 sm:grid-cols-3">
                <input
                  className="h-44 rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                  placeholder="Nombre completo *"
                  value={nombre}
                  onChange={e => setNombre(e.target.value)}
                />
                <input
                  type="email"
                  className="h-44 rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                  placeholder="Email *"
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                />
                <select
                  className="h-44 rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                  value={rol}
                  onChange={e => setRol(e.target.value as RolInvitable)}
                >
                  <option value="encargado">Encargado de Sucursal</option>
                  <option value="mozo">Mozo / Recepcionista</option>
                  <option value="cocina">Cocina (KDS)</option>
                </select>
              </div>
            </>
          )}

          <div className="flex flex-col items-stretch justify-between gap-12 pt-4 sm:flex-row sm:items-center">
            <p className="text-12 text-sage-green leading-normal">
              {ROL_DESCRIPTIONS[rol]}
            </p>
            <button
              type="submit"
              disabled={submitting}
              className="h-44 shrink-0 rounded-lg bg-plain-green px-20 text-center text-13 font-semibold text-canvas-white transition-colors hover:bg-plain-green-muted disabled:opacity-50"
            >
              {submitting
                ? "Guardando..."
                : modoAlta === 'credenciales'
                  ? "Crear usuario de equipo"
                  : "Generar Magic Link"}
            </button>
          </div>

          {error && <p className="text-12 font-medium text-alert-red">{error}</p>}

          {/* Tarjeta de credenciales recién creadas (US-83 CA-2) */}
          {credencialesCreadas && (
            <div className="space-y-10 rounded-xl border border-emerald-300 bg-emerald-50/70 p-16">
              <div className="flex items-center justify-between gap-8">
                <p className="text-13 font-semibold text-emerald-900">
                  ✓ Credenciales listas para {credencialesCreadas.nombre} ({ROL_LABELS[credencialesCreadas.rol]})
                </p>
                <button
                  type="button"
                  onClick={() => setCredencialesCreadas(null)}
                  className="text-11 text-emerald-700 hover:underline"
                >
                  Cerrar
                </button>
              </div>
              <div className="grid grid-cols-1 gap-8 rounded-lg border border-emerald-200 bg-canvas-white p-12 font-mono text-12 sm:grid-cols-3">
                <div>
                  <span className="block text-10 uppercase text-sage-green">Portal</span>
                  <span className="font-semibold text-ash-graphite break-all">{portalUrl || '/login'}</span>
                </div>
                <div>
                  <span className="block text-10 uppercase text-sage-green">Usuario</span>
                  <span className="font-semibold text-ash-graphite">@{credencialesCreadas.username}</span>
                </div>
                <div>
                  <span className="block text-10 uppercase text-sage-green">Contraseña</span>
                  <span className="font-semibold text-ash-graphite">{credencialesCreadas.password}</span>
                </div>
              </div>
              <div className="flex justify-end">
                <button
                  type="button"
                  onClick={() =>
                    copiarTexto(
                      armarTextoCredenciales(
                        credencialesCreadas.username,
                        credencialesCreadas.password,
                        credencialesCreadas.nombre
                      ),
                      'Usuario, clave y link copiados para enviar por WhatsApp.'
                    )
                  }
                  className="inline-flex h-40 items-center gap-6 rounded-lg bg-plain-green px-16 text-12 font-semibold text-canvas-white hover:bg-plain-green-muted"
                >
                  <span className="material-symbols-outlined text-16">content_copy</span>
                  {copiado ? "¡Copiado!" : "Copiar usuario y clave"}
                </button>
              </div>
            </div>
          )}

          {invitacionLink && (
            <div className="space-y-8 rounded-lg border border-success/40 bg-success/10 p-12">
              <p className="text-12 font-semibold text-[#087645]">✓ Invitación generada correctamente:</p>
              <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-8">
                <input
                  type="text"
                  readOnly
                  value={invitacionLink}
                  className="h-44 flex-1 rounded-lg border border-concrete bg-canvas-white px-10 text-11 font-mono"
                />
                <button
                  type="button"
                  onClick={() => copiarTexto(invitacionLink, 'Magic link copiado.')}
                  className="h-44 rounded-lg bg-plain-green px-16 text-center text-12 font-semibold text-canvas-white hover:bg-plain-green-muted"
                >
                  {copiado ? "¡Copiado!" : "Copiar Link"}
                </button>
              </div>
            </div>
          )}
        </form>
      </section>

      {/* Modal para Blanquear / Cambiar Contraseña (US-83 CA-3) */}
      {usuarioReset && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-system-black/50 p-16 backdrop-blur-xs">
          <div className="w-full max-w-[440px] overflow-hidden rounded-2xl border border-concrete bg-canvas-white shadow-2xl">
            <div className="flex items-center justify-between border-b border-concrete px-20 py-16">
              <div>
                <h3 className="text-15 font-semibold text-ash-graphite">Blanquear contraseña</h3>
                <p className="text-12 text-sage-green">
                  {usuarioReset.nombre} {usuarioReset.username ? `(@${usuarioReset.username})` : ''}
                </p>
              </div>
              <button
                type="button"
                onClick={() => setUsuarioReset(null)}
                className="flex h-32 w-32 items-center justify-center rounded-lg text-sage-green hover:bg-ghost-fog"
              >
                <span className="material-symbols-outlined text-18">close</span>
              </button>
            </div>
            <form onSubmit={handleConfirmarResetPassword} className="space-y-16 p-20">
              <p className="text-12 text-sage-green">
                Asigná una nueva contraseña temporal o definitiva. Al guardar, se copiará automáticamente al portapapeles para que se la envíes.
              </p>
              <div className="space-y-6">
                <label className="text-12 font-semibold text-ash-graphite">Nueva contraseña</label>
                <div className="flex items-center gap-8">
                  <div className="flex h-44 flex-1 items-center rounded-lg border border-concrete bg-canvas-white px-12 focus-within:border-plain-green">
                    <input
                      type={mostrarPasswordReset ? "text" : "password"}
                      value={nuevaPasswordReset}
                      onChange={e => setNuevaPasswordReset(e.target.value)}
                      className="h-full flex-1 bg-transparent font-mono text-13 outline-none"
                    />
                    <button
                      type="button"
                      onClick={() => setMostrarPasswordReset(p => !p)}
                      className="text-11 font-semibold text-sage-green"
                    >
                      {mostrarPasswordReset ? "Ocultar" : "Ver"}
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => setNuevaPasswordReset(generarClaveSegura())}
                    className="flex h-44 w-44 shrink-0 items-center justify-center rounded-lg border border-concrete bg-ghost-fog text-ash-graphite hover:border-stone"
                    title="Autogenerar otra clave"
                  >
                    <span className="material-symbols-outlined text-18">autorenew</span>
                  </button>
                </div>
              </div>
              <div className="flex items-center justify-end gap-8 pt-8">
                <button
                  type="button"
                  onClick={() => setUsuarioReset(null)}
                  className="h-40 rounded-lg border border-concrete px-16 text-12 font-semibold text-ash-graphite hover:bg-ghost-fog"
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  disabled={guardandoReset}
                  className="inline-flex h-40 items-center gap-6 rounded-lg bg-plain-green px-16 text-12 font-semibold text-canvas-white hover:bg-plain-green-muted disabled:opacity-50"
                >
                  <span className="material-symbols-outlined text-16">content_copy</span>
                  {guardandoReset ? "Guardando..." : "Guardar y copiar clave"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
