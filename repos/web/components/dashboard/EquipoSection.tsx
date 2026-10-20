"use client";

import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  api,
  EquipoAPI,
  EstadoUsuario,
  getErrorMessage,
  InvitacionEquipoAPI,
  SesionUsuario,
  Sucursal,
  Tenant,
  UsuarioAPI,
} from "@/lib/api";
import { EmptyState, Skeleton, useConfirm, useToast } from "@/components/ui";

type RolUsuario = "admin" | "encargado" | "mozo" | "cocina";
type RolInvitable = "encargado" | "mozo" | "cocina";
type ModoAlta = "credenciales" | "invitacion";
type EntregaInvitacion = {
  link: string;
  email: string;
  emailEnviado: boolean;
  advertencia?: string;
};

const DESCRIPCIONES: Record<RolInvitable, string> = {
  encargado: "Administra la operación y el equipo de una única sucursal.",
  mozo: "Gestiona carta y pedidos en vivo; puede ver mesas sin modificar mesas ni QR.",
  cocina: "Acceso directo a la pantalla Kitchen Display System (KDS) y carta en lectura.",
};

const ROL_LABELS: Record<RolUsuario, string> = {
  admin: "Admin",
  encargado: "Encargado",
  mozo: "Mozo / Salón",
  cocina: "Cocina (KDS)",
};

function generarClaveSegura(): string {
  const prefijos = ["Mesa", "Salon", "Cocina", "Turno", "Clave", "Staff"];
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

export default function EquipoSection({
  embedded = false,
  sesion = null,
}: {
  embedded?: boolean;
  sesion?: SesionUsuario | null;
}) {
  const toast = useToast();
  const confirmar = useConfirm();

  const [equipo, setEquipo] = useState<EquipoAPI>({ usuarios: [], invitaciones: [] });
  const [sucursales, setSucursales] = useState<Sucursal[]>([]);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loading, setLoading] = useState(true);
  const [actualizandoId, setActualizandoId] = useState<string | null>(null);
  const [mostrarArchivados, setMostrarArchivados] = useState(false);

  // Modo de alta: credenciales (usuario y clave) o invitación por Google/email
  const [modoAlta, setModoAlta] = useState<ModoAlta>("credenciales");
  const [nombre, setNombre] = useState("");
  const [username, setUsername] = useState("");
  const [usernameEditado, setUsernameEditado] = useState(false);
  const [email, setEmail] = useState("");
  const [rol, setRol] = useState<RolInvitable>("mozo");
  const [sucursalId, setSucursalId] = useState<string>("");
  const [password, setPassword] = useState(() => generarClaveSegura());
  const [mostrarPassword, setMostrarPassword] = useState(true);
  const [guardando, setGuardando] = useState(false);
  const [error, setError] = useState("");

  // Resultados de alta
  const [credencialesCreadas, setCredencialesCreadas] = useState<{
    nombre: string;
    username: string;
    password: string;
    rol: RolInvitable;
  } | null>(null);
  const [entrega, setEntrega] = useState<EntregaInvitacion | null>(null);
  const [copiado, setCopiado] = useState(false);

  // Modal de blanqueo de contraseña
  const [usuarioReset, setUsuarioReset] = useState<UsuarioAPI | null>(null);
  const [nuevaPasswordReset, setNuevaPasswordReset] = useState("");
  const [mostrarPasswordReset, setMostrarPasswordReset] = useState(true);
  const [guardandoReset, setGuardandoReset] = useState(false);

  const esDuenio = !sesion || sesion.rol === "admin";

  const cargar = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [equipoRes, sedesRes, tenantRes] = await Promise.allSettled([
        api.listarUsuarios(),
        api.listarSucursales(),
        api.obtenerMiTenant(),
      ]);

      if (equipoRes.status === "fulfilled" && equipoRes.value) {
        if (Array.isArray(equipoRes.value)) {
          setEquipo({ usuarios: equipoRes.value, invitaciones: [] });
        } else {
          setEquipo({
            usuarios: equipoRes.value.usuarios ?? [],
            invitaciones: equipoRes.value.invitaciones ?? [],
          });
        }
      }

      if (sedesRes.status === "fulfilled" && Array.isArray(sedesRes.value)) {
        setSucursales(sedesRes.value);
        const defaultSuc = sesion?.sucursal_id || sedesRes.value[0]?.id || "";
        setSucursalId((actual) => actual || defaultSuc);
      }

      if (tenantRes.status === "fulfilled" && tenantRes.value) {
        setTenant(tenantRes.value);
      }
    } catch (err: unknown) {
      setError(getErrorMessage(err, "No pudimos cargar el equipo."));
    } finally {
      setLoading(false);
    }
  }, [sesion?.sucursal_id]);

  useEffect(() => {
    void Promise.resolve().then(cargar);
  }, [cargar]);

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

  const portalUrl =
    typeof window !== "undefined" && tenant?.slug
      ? `${window.location.origin}/${tenant.slug}/login`
      : tenant?.slug
        ? `/${tenant.slug}/login`
        : "";

  const copiarTexto = (texto: string, mensajeToast: string) => {
    void navigator.clipboard
      .writeText(texto)
      .then(() => {
        setCopiado(true);
        toast.success(mensajeToast);
        setTimeout(() => setCopiado(false), 2500);
      })
      .catch(() => toast.error("No pudimos copiar al portapapeles."));
  };

  const armarTextoCredenciales = (user: string, pass: string, nombreStaff?: string) => {
    const urlLinea = portalUrl ? `\nPortal de ingreso: ${portalUrl}` : "";
    return `Hola${nombreStaff ? ` ${nombreStaff}` : ""}! Tus datos de acceso a Mesa CLICK (${tenant?.nombre || "Local"}):${urlLinea}\nUsuario: ${user}\nClave: ${pass}`;
  };

  const handleSubmitAlta = async (event: React.FormEvent) => {
    event.preventDefault();
    setError("");
    setCredencialesCreadas(null);
    setEntrega(null);

    const sucursalObjetivo = esDuenio ? sucursalId : sesion?.sucursal_id || sucursalId;
    if (esDuenio && !sucursalObjetivo) {
      setError("Elegí una sucursal.");
      return;
    }

    if (modoAlta === "credenciales") {
      if (!nombre.trim()) {
        setError("El nombre completo es obligatorio.");
        return;
      }
      const cleanUser = username.trim().toLowerCase().replace(/^@/, "");
      if (cleanUser.length < 3) {
        setError("El nombre de usuario debe tener al menos 3 caracteres.");
        return;
      }
      if (password.length < 8 || !/[a-zA-Z]/.test(password) || !/[0-9]/.test(password)) {
        setError("La contraseña debe tener al menos 8 caracteres e incluir letras y números.");
        return;
      }

      setGuardando(true);
      try {
        const creado = await api.crearUsuarioStaff({
          nombre: nombre.trim(),
          username: cleanUser,
          email: email.trim() || undefined,
          password,
          rol,
          sucursal_id: sucursalObjetivo || undefined,
        });
        setEquipo((actual) => ({
          ...actual,
          usuarios: [...actual.usuarios, creado],
        }));
        setCredencialesCreadas({
          nombre: creado.nombre,
          username: creado.username || cleanUser,
          password,
          rol,
        });
        setNombre("");
        setUsername("");
        setUsernameEditado(false);
        setEmail("");
        setPassword(generarClaveSegura());
        toast.success(`Usuario @${creado.username || cleanUser} creado correctamente.`);
      } catch (err: unknown) {
        const msg = getErrorMessage(err, "No se pudo crear el usuario de equipo.");
        setError(msg);
        toast.error(msg);
      } finally {
        setGuardando(false);
      }
    } else {
      if (!email.includes("@")) {
        setError("Ingresá un email válido.");
        return;
      }
      setGuardando(true);
      try {
        const respuesta = await api.invitarUsuario({
          email: email.trim(),
          rol,
          sucursal_id: esDuenio ? sucursalObjetivo : undefined,
        });
        setEquipo((actual) => ({
          ...actual,
          invitaciones: [respuesta.invitacion, ...actual.invitaciones],
        }));
        setEmail("");
        setEntrega({
          link: respuesta.invitacion_link,
          email: respuesta.invitacion.email,
          emailEnviado: respuesta.email_enviado,
          advertencia: respuesta.advertencia_email,
        });
        if (respuesta.email_enviado) {
          toast.success(`Invitación enviada a ${respuesta.invitacion.email}.`);
        } else {
          toast.info(
            respuesta.advertencia_email ?? "La invitación se creó. Compartí el enlace manualmente.",
          );
        }
      } catch (err: unknown) {
        const mensaje = getErrorMessage(err, "No pudimos enviar la invitación.");
        setError(mensaje);
        toast.error(mensaje);
      } finally {
        setGuardando(false);
      }
    }
  };

  const handleCambiarRol = async (id: string, nuevoRol: RolUsuario) => {
    try {
      setActualizandoId(id);
      const actualizado = await api.actualizarUsuario(id, { rol: nuevoRol });
      setEquipo((actual) => ({
        ...actual,
        usuarios: actual.usuarios.map((u) =>
          u.id === id ? { ...u, ...actualizado, rol: nuevoRol } : u,
        ),
      }));
      toast.success("Rol actualizado.");
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "No se pudo actualizar el rol."));
    } finally {
      setActualizandoId(null);
    }
  };

  const handleCambiarSucursal = async (id: string, nuevaSucursalId: string) => {
    try {
      setActualizandoId(id);
      const actualizado = await api.actualizarUsuario(id, { sucursal_id: nuevaSucursalId });
      setEquipo((actual) => ({
        ...actual,
        usuarios: actual.usuarios.map((u) => (u.id === id ? { ...u, ...actualizado } : u)),
      }));
      toast.success("Sucursal asignada actualizada.");
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "No se pudo actualizar la sucursal."));
    } finally {
      setActualizandoId(null);
    }
  };

  const handleCambiarEstado = async (miembro: UsuarioAPI, nuevoEstado: EstadoUsuario) => {
    if (nuevoEstado === "bloqueado") {
      const ok = await confirmar({
        titulo: `Bloquear acceso de ${miembro.nombre}`,
        mensaje: `¿Querés bloquear temporalmente a ${miembro.nombre}? Sus sesiones activas se cerrarán y no podrá ingresar hasta que lo reactives.`,
        labelAceptar: "Sí, bloquear acceso",
        variante: "danger",
      });
      if (!ok) return;
    }

    try {
      setActualizandoId(miembro.id);
      const actualizado = await api.cambiarEstadoUsuario(miembro.id, nuevoEstado);
      setEquipo((actual) => ({
        ...actual,
        usuarios: actual.usuarios.map((u) =>
          u.id === miembro.id ? { ...u, ...actualizado, estado: nuevoEstado } : u,
        ),
      }));
      const etiquetas: Record<EstadoUsuario, string> = {
        activo: "Acceso reactivado.",
        bloqueado: "Acceso bloqueado y sesiones revocadas.",
        archivado: "Miembro archivado.",
      };
      toast.success(etiquetas[nuevoEstado]);
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "No pudimos cambiar el estado del miembro."));
    } finally {
      setActualizandoId(null);
    }
  };

  const eliminar = async (miembro: UsuarioAPI) => {
    const ok = await confirmar({
      titulo: "Eliminar miembro",
      mensaje: `Se eliminará el acceso de ${miembro.nombre}. Los pedidos históricos conservarán su nombre.`,
      labelAceptar: "Eliminar",
      variante: "danger",
    });
    if (!ok) return;
    try {
      await api.eliminarUsuario(miembro.id);
      setEquipo((actual) => ({
        ...actual,
        usuarios: actual.usuarios.filter((u) => u.id !== miembro.id),
      }));
      toast.success("Miembro eliminado.");
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "No pudimos eliminar el miembro."));
    }
  };

  const cancelar = async (invitacion: InvitacionEquipoAPI) => {
    try {
      await api.cancelarInvitacion(invitacion.id);
      setEquipo((actual) => ({
        ...actual,
        invitaciones: actual.invitaciones.filter((i) => i.id !== invitacion.id),
      }));
      toast.success("Invitación cancelada.");
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "No pudimos cancelar la invitación."));
    }
  };

  const reenviar = async (invitacion: InvitacionEquipoAPI) => {
    try {
      const respuesta = await api.reenviarInvitacion(invitacion.id);
      setEquipo((actual) => ({
        ...actual,
        invitaciones: actual.invitaciones.map((i) =>
          i.id === invitacion.id ? respuesta.invitacion : i,
        ),
      }));
      setEntrega({
        link: respuesta.invitacion_link,
        email: respuesta.invitacion.email,
        emailEnviado: respuesta.email_enviado,
        advertencia: respuesta.advertencia_email,
      });
      if (respuesta.email_enviado) {
        toast.success(
          `Enviamos un enlace nuevo a ${respuesta.invitacion.email}; el anterior dejó de ser válido.`,
        );
      } else {
        toast.info(
          respuesta.advertencia_email ?? "Generamos un enlace nuevo. Compartilo manualmente.",
        );
      }
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "No pudimos reenviar la invitación."));
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
    if (
      nuevaPasswordReset.length < 8 ||
      !/[a-zA-Z]/.test(nuevaPasswordReset) ||
      !/[0-9]/.test(nuevaPasswordReset)
    ) {
      toast.error("La clave debe tener al menos 8 caracteres con letras y números.");
      return;
    }
    setGuardandoReset(true);
    try {
      await api.resetearPasswordUsuario(usuarioReset.id, nuevaPasswordReset);
      const userIdentifier = usuarioReset.username || usuarioReset.email || usuarioReset.nombre;
      copiarTexto(
        armarTextoCredenciales(userIdentifier, nuevaPasswordReset, usuarioReset.nombre),
        "Clave actualizada y copiada al portapapeles.",
      );
      setUsuarioReset(null);
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "No se pudo blanquear la contraseña."));
    } finally {
      setGuardandoReset(false);
    }
  };

  const miembrosStaff = equipo.usuarios.filter((u) => u.rol !== "admin");
  const archivadosCount = miembrosStaff.filter((u) => u.estado === "archivado").length;
  const miembrosVisibles = miembrosStaff.filter(
    (u) => mostrarArchivados || u.estado !== "archivado",
  );

  const renderEstadoBadge = (estado?: EstadoUsuario) => {
    const est = estado || "activo";
    if (est === "bloqueado") {
      return (
        <span className="inline-flex items-center gap-4 rounded-full border border-amber-300 bg-amber-50 px-8 py-2 text-10 font-semibold text-amber-800">
          <span className="h-6 w-6 rounded-full bg-amber-500" />
          Bloqueado
        </span>
      );
    }
    if (est === "archivado") {
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
    <div
      className={`${
        embedded
          ? "space-y-20"
          : "h-full space-y-24 overflow-y-auto bg-ghost-fog/45 p-16 sm:p-24 md:p-32"
      } font-inter`}
    >
      <header>
        <p className="text-11 font-mono uppercase tracking-[0.14em] text-sage-green">
          Acceso del equipo
        </p>
        <h2 className="mt-4 text-24 font-semibold text-ash-graphite sm:text-32">
          Personas, credenciales e invitaciones
        </h2>
        <p className="mt-4 text-13 text-sage-green">
          Creá usuarios con clave operativa o enviá invitaciones por correo para que entren con
          Google.
        </p>
      </header>

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
                Compartí este link en las tablets de cocina o celulares de tus mozos para que
                ingresen con su usuario y clave o con su cuenta de Google:
              </p>
              <p className="break-all font-mono text-12 font-semibold text-plain-green">
                {portalUrl}
              </p>
            </div>
            <div className="flex shrink-0 flex-wrap items-center gap-8">
              <button
                type="button"
                onClick={() => copiarTexto(portalUrl, "Link del portal copiado al portapapeles.")}
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

      {/* Formulario Unificado de Alta de Equipo (US-83) */}
      <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
        <div className="flex flex-col gap-12 border-b border-concrete/70 px-16 py-14 sm:flex-row sm:items-center sm:justify-between sm:px-20">
          <div>
            <p className="text-15 font-semibold tracking-[-0.01em] text-ash-graphite">
              Agregar miembro al equipo
            </p>
            <p className="text-12 text-sage-green">
              Elegí entre crear usuario y clave listos para usar o enviar una invitación para
              acceso con Google.
            </p>
          </div>
          <div className="inline-flex rounded-lg border border-concrete bg-ghost-fog p-3">
            <button
              type="button"
              onClick={() => {
                setModoAlta("credenciales");
                setError("");
              }}
              className={`rounded-md px-12 py-6 text-12 font-semibold transition-colors ${
                modoAlta === "credenciales"
                  ? "bg-canvas-white text-ash-graphite shadow-xs"
                  : "text-sage-green hover:text-ash-graphite"
              }`}
            >
              Usuario y clave
            </button>
            <button
              type="button"
              onClick={() => {
                setModoAlta("invitacion");
                setError("");
              }}
              className={`rounded-md px-12 py-6 text-12 font-semibold transition-colors ${
                modoAlta === "invitacion"
                  ? "bg-canvas-white text-ash-graphite shadow-xs"
                  : "text-sage-green hover:text-ash-graphite"
              }`}
            >
              Invitación Google (Email)
            </button>
          </div>
        </div>

        <form onSubmit={handleSubmitAlta} className="space-y-16 p-16 sm:p-20">
          {modoAlta === "credenciales" ? (
            <>
              <div className="grid grid-cols-1 gap-12 sm:grid-cols-2">
                <label className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">Nombre completo *</span>
                  <input
                    className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                    placeholder="Ej: Juan Pérez"
                    value={nombre}
                    onChange={(e) => handleNombreChange(e.target.value)}
                  />
                </label>

                <label className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">
                    Nombre de usuario (Alias) *
                  </span>
                  <div className="flex h-44 items-center rounded-lg border border-concrete bg-canvas-white px-12 focus-within:border-plain-green">
                    <span className="mr-4 font-mono text-13 text-sage-green">@</span>
                    <input
                      className="h-full flex-1 bg-transparent font-mono text-13 outline-none"
                      placeholder="mozo_juan"
                      value={username}
                      onChange={(e) => {
                        setUsernameEditado(true);
                        setUsername(e.target.value.toLowerCase().replace(/[^a-z0-9._-]/g, ""));
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
                    onChange={(e) => handleRolFormChange(e.target.value as RolInvitable)}
                  >
                    {esDuenio && <option value="encargado">Encargado</option>}
                    <option value="mozo">Mozo</option>
                    <option value="cocina">Cocinero</option>
                  </select>
                </label>

                <div className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">
                    Sucursal asignada *
                  </span>
                  {esDuenio ? (
                    <select
                      className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                      value={sucursalId}
                      onChange={(e) => setSucursalId(e.target.value)}
                    >
                      <option value="">Sucursal…</option>
                      {sucursales.map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.nombre}
                        </option>
                      ))}
                    </select>
                  ) : (
                    <div className="flex h-44 items-center rounded-lg border border-concrete bg-ghost-fog px-12 text-12 text-sage-green">
                      Sucursal asignada automáticamente
                    </div>
                  )}
                </div>

                <label className="space-y-4">
                  <span className="text-12 font-semibold text-ash-graphite">Email (opcional)</span>
                  <input
                    type="email"
                    className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
                    placeholder="Opcional"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                  />
                </label>
              </div>

              <div className="space-y-6">
                <span className="text-12 font-semibold text-ash-graphite">
                  Contraseña inicial *
                </span>
                <div className="flex flex-col gap-8 sm:flex-row sm:items-center">
                  <div className="flex h-44 flex-1 items-center rounded-lg border border-concrete bg-canvas-white px-12 focus-within:border-plain-green">
                    <input
                      type={mostrarPassword ? "text" : "password"}
                      className="h-full flex-1 bg-transparent font-mono text-13 outline-none"
                      placeholder="Mínimo 8 caracteres (letras y números)"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                    />
                    <button
                      type="button"
                      onClick={() => setMostrarPassword((p) => !p)}
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
            <div className="grid gap-10 md:grid-cols-[1fr_180px_1fr]">
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="persona@negocio.com"
                className="h-44 rounded-lg border border-concrete px-12 text-13 outline-none focus:border-plain-green"
              />
              <select
                value={rol}
                onChange={(e) => setRol(e.target.value as RolInvitable)}
                className="h-44 rounded-lg border border-concrete bg-white px-10 text-13"
              >
                {esDuenio && <option value="encargado">Encargado</option>}
                <option value="mozo">Mozo</option>
                <option value="cocina">Cocinero</option>
              </select>
              {esDuenio ? (
                <select
                  value={sucursalId}
                  onChange={(e) => setSucursalId(e.target.value)}
                  className="h-44 rounded-lg border border-concrete bg-white px-10 text-13"
                >
                  <option value="">Sucursal…</option>
                  {sucursales.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.nombre}
                    </option>
                  ))}
                </select>
              ) : (
                <div className="flex h-44 items-center rounded-lg border border-concrete bg-ghost-fog px-12 text-12 text-sage-green">
                  Sucursal asignada automáticamente
                </div>
              )}
            </div>
          )}

          <div className="flex flex-col items-stretch justify-between gap-12 pt-4 sm:flex-row sm:items-center">
            <p className="text-12 text-sage-green">
              {DESCRIPCIONES[rol]}{" "}
              {modoAlta === "invitacion" && "La invitación vence en 48 horas y se usa una sola vez."}
            </p>
            <button
              type="submit"
              disabled={guardando}
              className="h-44 shrink-0 rounded-lg bg-plain-green px-20 text-13 font-semibold text-white transition-colors hover:bg-plain-green-muted disabled:opacity-50"
            >
              {guardando
                ? "Guardando…"
                : modoAlta === "credenciales"
                  ? "Crear usuario de equipo"
                  : "Invitar"}
            </button>
          </div>

          {error && (
            <p role="alert" className="text-12 font-medium text-alert-red">
              {error}
            </p>
          )}

          {/* Tarjeta de credenciales recién creadas */}
          {credencialesCreadas && (
            <div className="space-y-10 rounded-xl border border-emerald-300 bg-emerald-50/70 p-16">
              <div className="flex items-center justify-between gap-8">
                <p className="text-13 font-semibold text-emerald-900">
                  ✓ Credenciales listas para {credencialesCreadas.nombre} (
                  {ROL_LABELS[credencialesCreadas.rol]})
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
                  <span className="break-all font-semibold text-ash-graphite">
                    {portalUrl || "/login"}
                  </span>
                </div>
                <div>
                  <span className="block text-10 uppercase text-sage-green">Usuario</span>
                  <span className="font-semibold text-ash-graphite">
                    @{credencialesCreadas.username}
                  </span>
                </div>
                <div>
                  <span className="block text-10 uppercase text-sage-green">Contraseña</span>
                  <span className="font-semibold text-ash-graphite">
                    {credencialesCreadas.password}
                  </span>
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
                        credencialesCreadas.nombre,
                      ),
                      "Usuario, clave y link copiados para enviar por WhatsApp.",
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

          {/* Tarjeta de invitación por email */}
          {entrega && (
            <div
              className={`rounded-lg border p-12 ${
                entrega.emailEnviado
                  ? "border-success/40 bg-success/10"
                  : "border-[#d99b36]/40 bg-[#fff8ea]"
              }`}
            >
              <p
                className={`text-12 font-semibold ${
                  entrega.emailEnviado ? "text-[#087645]" : "text-[#8a5a10]"
                }`}
              >
                {entrega.emailEnviado
                  ? `Invitación enviada por email a ${entrega.email}.`
                  : entrega.advertencia}
              </p>
              <p className="mt-4 text-11 text-sage-green">
                También podés compartir este enlace manualmente:
              </p>
              <div className="mt-8 flex min-w-0 flex-col gap-8 sm:flex-row">
                <code className="min-w-0 flex-1 truncate rounded-md border border-concrete bg-white px-10 py-9 text-11 text-ash-graphite">
                  {entrega.link}
                </code>
                <button
                  type="button"
                  onClick={() => copiarTexto(entrega.link, "Enlace copiado.")}
                  className="h-36 shrink-0 rounded-md bg-ash-graphite px-14 text-11 font-semibold text-white"
                >
                  Copiar enlace
                </button>
              </div>
            </div>
          )}
        </form>
      </section>

      {/* Miembros del equipo */}
      <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
        <div className="flex flex-wrap items-center justify-between gap-12 border-b border-concrete px-16 py-12">
          <div>
            <h3 className="text-15 font-semibold text-ash-graphite">
              Miembros activos y bloqueados
            </h3>
            <p className="text-11 text-sage-green">
              {miembrosVisibles.length}{" "}
              {miembrosVisibles.length === 1 ? "miembro visible" : "miembros visibles"}
            </p>
          </div>
          {archivadosCount > 0 && (
            <button
              type="button"
              onClick={() => setMostrarArchivados((prev) => !prev)}
              className="inline-flex h-36 items-center gap-6 rounded-lg border border-concrete bg-ghost-fog px-10 text-11 font-medium text-ash-graphite hover:border-stone"
            >
              <span className="material-symbols-outlined text-15">
                {mostrarArchivados ? "visibility_off" : "inventory_2"}
              </span>
              {mostrarArchivados ? "Ocultar archivados" : `Ver archivados (${archivadosCount})`}
            </button>
          )}
        </div>

        {loading ? (
          <div className="space-y-12 p-16">
            <Skeleton className="h-44 w-full" />
            <Skeleton className="h-44 w-full" />
          </div>
        ) : miembrosVisibles.length === 0 ? (
          <EmptyState
            icon="group"
            title="Todavía no hay miembros"
            description="Creá credenciales de acceso o enviá una invitación para sumar personas al equipo."
          />
        ) : (
          <div className="divide-y divide-ghost-fog">
            {miembrosVisibles.map((u) => {
              const est = u.estado || "activo";
              return (
                <article
                  key={u.id}
                  className={`flex flex-col gap-12 px-16 py-14 sm:flex-row sm:items-center sm:justify-between ${
                    est !== "activo" ? "bg-ghost-fog/40 opacity-85" : ""
                  }`}
                >
                  <div className="min-w-0 space-y-4">
                    <div className="flex flex-wrap items-center gap-8">
                      <p className="truncate text-14 font-semibold text-ash-graphite">{u.nombre}</p>
                      {renderEstadoBadge(u.estado)}
                    </div>
                    <div className="flex flex-wrap items-center gap-10 text-12 text-sage-green">
                      {u.username && (
                        <button
                          type="button"
                          onClick={() =>
                            copiarTexto(u.username!, `Usuario @${u.username} copiado.`)
                          }
                          className="inline-flex items-center gap-4 rounded bg-ghost-fog px-6 py-2 font-mono text-11 font-semibold text-ash-graphite hover:bg-concrete/60"
                          title="Clic para copiar nombre de usuario"
                        >
                          @{u.username}
                          <span className="material-symbols-outlined text-13 text-sage-green">
                            content_copy
                          </span>
                        </button>
                      )}
                      {u.email && <span className="truncate">{u.email}</span>}
                      <span>· {ROL_LABELS[u.rol] || u.rol}</span>
                      {u.sucursal_nombre && <span>· {u.sucursal_nombre}</span>}
                    </div>
                  </div>

                  <div className="flex flex-wrap items-center justify-between gap-8 sm:justify-end">
                    <select
                      value={u.rol}
                      disabled={actualizandoId === u.id}
                      onChange={(e) => void handleCambiarRol(u.id, e.target.value as RolUsuario)}
                      aria-label={`Rol de ${u.nombre}`}
                      className="h-38 cursor-pointer rounded-lg border border-concrete bg-canvas-white px-10 text-11 font-medium text-ash-graphite outline-none focus:border-plain-green disabled:opacity-50"
                    >
                      {esDuenio && <option value="encargado">Encargado</option>}
                      <option value="mozo">Mozo / Salón</option>
                      <option value="cocina">Cocina (KDS)</option>
                    </select>

                    {esDuenio && sucursales.length > 0 && (
                      <select
                        value={u.sucursal_id || ""}
                        disabled={actualizandoId === u.id}
                        onChange={(e) => void handleCambiarSucursal(u.id, e.target.value)}
                        aria-label={`Sucursal de ${u.nombre}`}
                        className="h-38 cursor-pointer rounded-lg border border-concrete bg-canvas-white px-10 text-11 text-sage-green outline-none focus:border-plain-green disabled:opacity-50"
                      >
                        {sucursales.map((s) => (
                          <option key={s.id} value={s.id}>
                            {s.nombre}
                          </option>
                        ))}
                      </select>
                    )}

                    <div className="flex items-center gap-6">
                      <button
                        type="button"
                        onClick={() => abrirModalResetPassword(u)}
                        className="flex h-38 w-38 items-center justify-center rounded-lg border border-concrete bg-canvas-white text-ash-graphite transition-colors hover:bg-ghost-fog"
                        title="Blanquear / Asignar contraseña"
                        aria-label={`Cambiar contraseña de ${u.nombre}`}
                      >
                        <span className="material-symbols-outlined text-18">key</span>
                      </button>

                      <button
                        type="button"
                        disabled={actualizandoId === u.id}
                        onClick={() =>
                          void handleCambiarEstado(u, est === "activo" ? "bloqueado" : "activo")
                        }
                        className="h-38 rounded-lg border border-concrete px-12 text-11 font-semibold text-ash-graphite hover:bg-ghost-fog disabled:opacity-50"
                      >
                        {est === "activo" ? "Bloquear" : "Reactivar"}
                      </button>

                      {est === "archivado" ? (
                        <button
                          type="button"
                          disabled={actualizandoId === u.id}
                          onClick={() => void handleCambiarEstado(u, "activo")}
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
                          onClick={() => void handleCambiarEstado(u, "archivado")}
                          className="flex h-38 w-38 items-center justify-center rounded-lg border border-concrete bg-canvas-white text-sage-green transition-colors hover:bg-ghost-fog"
                          title="Archivar miembro"
                          aria-label={`Archivar a ${u.nombre}`}
                        >
                          <span className="material-symbols-outlined text-18">inventory_2</span>
                        </button>
                      )}

                      <button
                        type="button"
                        onClick={() => void eliminar(u)}
                        className="flex h-38 w-38 items-center justify-center rounded-lg text-alert-red transition-colors hover:bg-warm-pink/20"
                        title="Eliminar miembro"
                        aria-label={`Eliminar a ${u.nombre}`}
                      >
                        <span className="material-symbols-outlined text-18">delete</span>
                      </button>
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>

      {/* Invitaciones pendientes */}
      <section className="overflow-hidden rounded-xl border border-concrete bg-canvas-white">
        <div className="border-b border-concrete px-16 py-12">
          <h3 className="text-15 font-semibold text-ash-graphite">Invitaciones pendientes</h3>
        </div>
        {equipo.invitaciones.length === 0 ? (
          <EmptyState compact icon="mark_email_unread" title="No hay invitaciones pendientes" />
        ) : (
          <div className="divide-y divide-ghost-fog">
            {equipo.invitaciones.map((i) => (
              <article
                key={i.id}
                className="flex flex-col gap-10 px-16 py-14 sm:flex-row sm:items-center sm:justify-between"
              >
                <div>
                  <p className="text-14 font-semibold text-ash-graphite">{i.email}</p>
                  <p className="text-12 text-sage-green">
                    {i.rol} · {i.sucursal} · vence {new Date(i.expires_at).toLocaleString("es-AR")}
                  </p>
                </div>
                <div className="flex gap-8">
                  <button
                    type="button"
                    onClick={() => void reenviar(i)}
                    className="h-40 rounded-lg border border-concrete px-12 text-11 font-semibold"
                  >
                    Reenviar
                  </button>
                  <button
                    type="button"
                    onClick={() => void cancelar(i)}
                    className="h-40 rounded-lg px-12 text-11 text-alert-red"
                  >
                    Cancelar
                  </button>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>

      {/* Modal para Blanquear / Cambiar Contraseña (US-83 CA-3) */}
      {usuarioReset && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-system-black/50 p-16 backdrop-blur-xs">
          <div className="w-full max-w-[440px] overflow-hidden rounded-2xl border border-concrete bg-canvas-white shadow-2xl">
            <div className="flex items-center justify-between border-b border-concrete px-20 py-16">
              <div>
                <h3 className="text-15 font-semibold text-ash-graphite">Blanquear contraseña</h3>
                <p className="text-12 text-sage-green">
                  {usuarioReset.nombre} {usuarioReset.username ? `(@${usuarioReset.username})` : ""}
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
                Asigná una nueva contraseña temporal o definitiva. Al guardar, se revocarán las
                sesiones previas y se copiará automáticamente al portapapeles.
              </p>
              <div className="space-y-6">
                <label className="text-12 font-semibold text-ash-graphite">Nueva contraseña</label>
                <div className="flex items-center gap-8">
                  <div className="flex h-44 flex-1 items-center rounded-lg border border-concrete bg-canvas-white px-12 focus-within:border-plain-green">
                    <input
                      type={mostrarPasswordReset ? "text" : "password"}
                      value={nuevaPasswordReset}
                      onChange={(e) => setNuevaPasswordReset(e.target.value)}
                      className="h-full flex-1 bg-transparent font-mono text-13 outline-none"
                    />
                    <button
                      type="button"
                      onClick={() => setMostrarPasswordReset((p) => !p)}
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
