"use client";
import { useState, useEffect, useCallback } from "react";
import { api, CategoriaAPI, ArticuloAPI, FranjaHorariaAPI, EstadoPlan, esPlanLimitReached, detallePlanLimit, getErrorMessage } from "@/lib/api";
import { EmptyState, Skeleton, useToast, useConfirm } from "@/components/ui";
import ImportarCartaModal from "@/components/dashboard/ImportarCartaModal";
import AjustePreciosModal from "./AjustePreciosModal";
import FranjasHorariasModal from "./FranjasHorariasModal";
import UpgradeModal from "./UpgradeModal";
import { prepararImagenProducto } from "@/lib/product-image";

export interface CategoriaConItems extends CategoriaAPI {
  items: ArticuloAPI[];
}

type NuevoItemForm = { nombre: string; descripcion: string; precio: string; fotoUrl: string };
type NuevoItemErrors = Partial<Record<"nombre" | "precio", string>>;

const ICONOS_CATEGORIA = [
  'restaurant', 'lunch_dining', 'ramen_dining', 'local_pizza', 'breakfast_dining',
  'bakery_dining', 'tapas', 'soup_kitchen', 'set_meal', 'cake', 'icecream',
  'local_cafe', 'local_bar', 'liquor', 'emoji_food_beverage',
] as const;

const MAX_DESCRIPCION_ARTICULO = 140;

type SelectorFranjaCategoriaProps = {
  categoria: CategoriaConItems;
  franjas: FranjaHorariaAPI[];
  abierto: boolean;
  guardando: boolean;
  onToggle: () => void;
  onSelect: (franjaId: string) => void;
};

function SelectorFranjaCategoria({
  categoria,
  franjas,
  abierto,
  guardando,
  onToggle,
  onSelect,
}: SelectorFranjaCategoriaProps) {
  const franjaActual = franjas.find(franja => franja.id === categoria.franja_horaria_id);

  return (
    <div className="relative min-w-0 flex-1 sm:w-[190px] sm:flex-none" onClick={event => event.stopPropagation()}>
      <button
        type="button"
        disabled={guardando}
        onClick={event => {
          event.stopPropagation();
          onToggle();
        }}
        className={`group flex h-44 w-full min-w-0 items-center gap-8 rounded-lg border px-10 text-left transition-all disabled:cursor-wait disabled:opacity-60 ${
          franjaActual
            ? "border-plain-green/30 bg-plain-green/[0.07] hover:border-plain-green/55"
            : "border-concrete bg-canvas-white hover:border-stone hover:bg-ghost-fog"
        }`}
        aria-label={`Cambiar horario de ${categoria.nombre}`}
        aria-haspopup="listbox"
        aria-expanded={abierto}
      >
        <span className={`material-symbols-outlined shrink-0 text-18 ${franjaActual ? 'text-plain-green' : 'text-sage-green'}`}>
          schedule
        </span>
        <span className="min-w-0 flex-1 truncate text-11 font-semibold text-ash-graphite">
          {franjaActual?.nombre || 'Todo el día'}
        </span>
        <span className={`material-symbols-outlined shrink-0 text-18 text-sage-green transition-transform ${abierto ? 'rotate-180' : ''}`}>
          expand_more
        </span>
      </button>

      {abierto && (
        <div
          role="listbox"
          aria-label={`Horario de ${categoria.nombre}`}
          className="absolute right-0 top-[50px] z-40 w-[260px] overflow-hidden rounded-xl border border-concrete bg-canvas-white p-6 shadow-xl"
        >
          <div className="px-12 pb-8 pt-6 text-center">
            <p className="text-10 font-semibold uppercase tracking-[0.08em] text-sage-green">Disponibilidad horaria</p>
            <p className="mt-2 text-10 text-sage-green">Opcional para toda la categoría</p>
          </div>

          <button
            type="button"
            role="option"
            aria-selected={!categoria.franja_horaria_id}
            onClick={() => onSelect('')}
            className={`flex min-h-48 w-full items-center gap-8 rounded-lg px-10 py-8 text-left transition-colors ${
              !categoria.franja_horaria_id ? 'bg-plain-green/[0.08]' : 'hover:bg-ghost-fog'
            }`}
          >
            <span className="flex h-30 w-30 shrink-0 items-center justify-center rounded-lg bg-ghost-fog text-sage-green">
              <span className="material-symbols-outlined text-18">all_inclusive</span>
            </span>
            <span className="min-w-0 flex-1">
              <span className="block truncate text-12 font-semibold text-ash-graphite">Siempre disponible</span>
              <span className="block truncate text-10 text-sage-green">Sin restricción horaria</span>
            </span>
            {!categoria.franja_horaria_id && (
              <span className="material-symbols-outlined shrink-0 text-18 text-plain-green">check</span>
            )}
          </button>

          {franjas.map(franja => {
            const seleccionada = categoria.franja_horaria_id === franja.id;
            return (
              <button
                key={franja.id}
                type="button"
                role="option"
                aria-selected={seleccionada}
                onClick={() => onSelect(franja.id)}
                className={`mt-2 flex min-h-48 w-full items-center gap-8 rounded-lg px-10 py-8 text-left transition-colors ${
                  seleccionada ? 'bg-plain-green/[0.08]' : 'hover:bg-ghost-fog'
                }`}
              >
                <span className={`flex h-30 w-30 shrink-0 items-center justify-center rounded-lg ${
                  seleccionada ? 'bg-plain-green/15 text-plain-green' : 'bg-ghost-fog text-sage-green'
                }`}>
                  <span className="material-symbols-outlined text-18">schedule</span>
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-12 font-semibold text-ash-graphite">{franja.nombre}</span>
                  <span className="block font-mono text-10 text-sage-green">{franja.hora_inicio} — {franja.hora_fin}</span>
                </span>
                {seleccionada && (
                  <span className="material-symbols-outlined shrink-0 text-18 text-plain-green">check</span>
                )}
              </button>
            );
          })}

          {franjas.length === 0 && (
            <p className="mx-8 mb-6 mt-8 rounded-lg bg-ghost-fog px-10 py-8 text-10 leading-relaxed text-sage-green">
              Creá una franja desde el botón Horarios para poder limitar esta categoría.
            </p>
          )}
        </div>
      )}
    </div>
  );
}

export default function CartaSection() {
  const toast = useToast();
  const confirmar = useConfirm();
  const [categorias, setCategorias] = useState<CategoriaConItems[]>([]);
  const [franjas, setFranjas] = useState<FranjaHorariaAPI[]>([]);
  const [loading, setLoading] = useState(true);
  const [nuevaCatNombre, setNuevaCatNombre] = useState('');
  const [mostrarFormCat, setMostrarFormCat] = useState(false);
  const [mostrarFormItem, setMostrarFormItem] = useState<string | null>(null);
  const [nuevoItem, setNuevoItem] = useState<NuevoItemForm>({ nombre: '', descripcion: '', precio: '', fotoUrl: '' });
  const [nuevoItemErrors, setNuevoItemErrors] = useState<NuevoItemErrors>({});
  const [errorMsg, setErrorMsg] = useState('');
  const [articuloVariantesAbierto, setArticuloVariantesAbierto] = useState<string | null>(null);
  const [nuevaVariante, setNuevaVariante] = useState<{
    nombre: string;
    grupo: string;
    precioAdicional: string;
    seleccionUnica: boolean;
  }>({
    nombre: '',
    grupo: '',
    precioAdicional: '0',
    seleccionUnica: false,
  });
  const [guardandoVariante, setGuardandoVariante] = useState(false);
  const [modalImportarAbierto, setModalImportarAbierto] = useState(false);
  const [mostrarAjustePrecios, setMostrarAjustePrecios] = useState(false);
  const [mostrarFranjasHorarias, setMostrarFranjasHorarias] = useState(false);
  const [guardandoHorarioId, setGuardandoHorarioId] = useState<string | null>(null);
  const [menuHorarioCategoriaAbierto, setMenuHorarioCategoriaAbierto] = useState<string | null>(null);
  const [menuItemAbierto, setMenuItemAbierto] = useState<string | null>(null);
  const [menuIconoCategoriaAbierto, setMenuIconoCategoriaAbierto] = useState<string | null>(null);
  const [procesandoImagen, setProcesandoImagen] = useState(false);
  const [actualizandoFotoId, setActualizandoFotoId] = useState<string | null>(null);

  const [estadoPlan, setEstadoPlan] = useState<EstadoPlan | null>(null);
  const [modalUpgradeOpen, setModalUpgradeOpen] = useState(false);
  const [modalUpgradeRecurso, setModalUpgradeRecurso] = useState<"productos" | "carga_masiva" | "personalizacion">("productos");
  const [modalLimiteInfo, setModalLimiteInfo] = useState<{ limite?: number; uso?: number }>({});

  const cargarPlan = useCallback(async () => {
    try {
      const plan = await api.obtenerMiPlan();
      setEstadoPlan(plan);
    } catch (err) {
      console.warn("No se pudo cargar plan en CartaSection:", err);
    }
  }, []);

  const cargarCarta = useCallback(async () => {
    try {
      setLoading(true);
      setErrorMsg('');
      const [cats, arts, franjasConfiguradas] = await Promise.all([
        api.listarCategorias(),
        api.listarArticulos(),
        api.listarFranjasHorarias(),
      ]);

      setFranjas(franjasConfiguradas || []);

      if (cats && cats.length > 0) {
        const combinadas: CategoriaConItems[] = cats.map(c => ({
          ...c,
          items: arts ? arts.filter(a => a.categoria_id === c.id) : [],
        }));
        setCategorias(combinadas);
      } else {
        setCategorias([]);
      }
    } catch (err: unknown) {
      console.error("Error al cargar la carta desde la API:", err);
      setCategorias([]);
      setErrorMsg(getErrorMessage(err, 'Error al conectar con la API de carta.'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      void cargarCarta();
      void cargarPlan();
    }, 0);

    return () => window.clearTimeout(timeoutId);
  }, [cargarCarta, cargarPlan]);

  useEffect(() => {
    const cerrarMenus = () => {
      setMenuHorarioCategoriaAbierto(null);
      setMenuItemAbierto(null);
      setMenuIconoCategoriaAbierto(null);
    };
    window.addEventListener("click", cerrarMenus);
    return () => window.removeEventListener("click", cerrarMenus);
  }, []);

  const agregarCategoria = async () => {
    if (!nuevaCatNombre.trim()) return;
    setErrorMsg('');

    try {
      const creada = await api.crearCategoria(nuevaCatNombre.trim(), categorias.length);
      setCategorias(prev => [
        ...prev,
        { ...creada, items: [] },
      ]);
      setNuevaCatNombre('');
      setMostrarFormCat(false);
      toast.success('Categoría creada correctamente.');
    } catch (err: unknown) {
      setErrorMsg(getErrorMessage(err, 'Error al crear la categoría.'));
    }
  };

  const eliminarCategoria = async (catId: string) => {
    const ok = await confirmar({
      titulo: 'Eliminar categoría',
      mensaje: '¿Eliminás esta categoría y todos sus ítems? Esta acción no se puede deshacer.',
      labelAceptar: 'Sí, eliminar',
      variante: 'danger',
    });
    if (!ok) return;
    try {
      await api.eliminarCategoria(catId);
      setCategorias(prev => prev.filter(c => c.id !== catId));
      toast.success('Categoría eliminada.');
    } catch (err: unknown) {
      console.error("No se pudo eliminar categoría:", err);
      setErrorMsg(getErrorMessage(err, 'Error al eliminar categoría.'));
    }
  };

  const agregarItem = async (catId: string) => {
    const errores: NuevoItemErrors = {};
    if (!nuevoItem.nombre.trim()) {
      errores.nombre = 'Falta completar el nombre del ítem.';
    }
    if (!nuevoItem.precio.trim()) {
      errores.precio = 'Falta completar el precio del ítem.';
    }

    if (Object.keys(errores).length > 0) {
      setNuevoItemErrors(errores);
      setErrorMsg('');
      return;
    }

    const precioNum = parseFloat(nuevoItem.precio);
    if (isNaN(precioNum) || precioNum <= 0) {
      setNuevoItemErrors({ precio: 'Ingresa un precio válido mayor a 0.' });
      setErrorMsg('');
      return;
    }
    setNuevoItemErrors({});
    setErrorMsg('');

    try {
      const creado = await api.crearArticulo({
        categoria_id: catId,
        nombre: nuevoItem.nombre.trim(),
        descripcion: nuevoItem.descripcion.trim(),
        precio: precioNum,
        foto_url: nuevoItem.fotoUrl || undefined,
        activo: true,
      });

      setCategorias(prev =>
        prev.map(c => c.id === catId ? { ...c, items: [...c.items, creado] } : c)
      );
      setNuevoItem({ nombre: '', descripcion: '', precio: '', fotoUrl: '' });
      setMostrarFormItem(null);
      toast.success('Ítem agregado a la carta.');
      void cargarPlan();
    } catch (err: unknown) {
      if (esPlanLimitReached(err)) {
        const detalle = detallePlanLimit(err);
        setModalUpgradeRecurso("productos");
        setModalLimiteInfo({
          limite: detalle?.limite ?? 30,
          uso: detalle?.uso ?? 30,
        });
        setModalUpgradeOpen(true);
        return;
      }
      setErrorMsg(getErrorMessage(err, 'Error al crear el artículo.'));
    }
  };

  const eliminarItem = async (catId: string, itemId: string) => {
    const ok = await confirmar({
      titulo: 'Eliminar ítem',
      mensaje: '¿Eliminás este ítem del menú? Esta acción no se puede deshacer.',
      labelAceptar: 'Sí, eliminar',
      variante: 'danger',
    });
    if (!ok) return;
    try {
      await api.eliminarArticulo(itemId);
      setCategorias(prev =>
        prev.map(c =>
          c.id === catId ? { ...c, items: c.items.filter(i => i.id !== itemId) } : c
        )
      );
      toast.success('Ítem eliminado.');
      void cargarPlan();
    } catch (err: unknown) {
      console.error("No se pudo eliminar artículo:", err);
      setErrorMsg(getErrorMessage(err, 'Error al eliminar artículo.'));
    }
  };

  const toggleDisponible = async (catId: string, item: ArticuloAPI) => {
    const nuevoEstado = !(item.activo !== false);
    try {
      await api.actualizarArticulo(item.id, { activo: nuevoEstado });
      setCategorias(prev =>
        prev.map(c =>
          c.id === catId
            ? {
                ...c,
                items: c.items.map(i =>
                  i.id === item.id ? { ...i, activo: nuevoEstado } : i
                ),
              }
            : c
        )
      );
      toast.success(
        item.activo !== false
          ? 'Ítem ocultado del menú público.'
          : 'Ítem visible en el menú público.'
      );
    } catch (err: unknown) {
      console.error("No se pudo actualizar visibilidad:", err);
      setErrorMsg(getErrorMessage(err, 'Error al actualizar visibilidad.'));
    }
  };

  const asignarFranjaCategoria = async (categoriaId: string, franjaId: string) => {
    try {
      setGuardandoHorarioId(categoriaId);
      const actualizada = await api.asignarFranjaCategoria(categoriaId, franjaId || null);
      setCategorias(prev => prev.map(categoria => (
        categoria.id === categoriaId
          ? { ...categoria, franja_horaria_id: actualizada.franja_horaria_id }
          : categoria
      )));
      setMenuHorarioCategoriaAbierto(null);
      toast.success(franjaId ? "Horario aplicado a la categoría." : "La categoría quedó disponible todo el día.");
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "No se pudo actualizar el horario."));
    } finally {
      setGuardandoHorarioId(null);
    }
  };

  const asignarIconoCategoria = async (categoriaId: string, icono: string) => {
    if (estadoPlan?.plan !== 'pro') {
      setModalUpgradeRecurso('personalizacion');
      setModalUpgradeOpen(true);
      return;
    }
    try {
      setGuardandoHorarioId(categoriaId);
      const actualizada = await api.asignarIconoCategoria(categoriaId, icono);
      setCategorias(prev => prev.map(categoria => categoria.id === categoriaId
        ? { ...categoria, icono: actualizada.icono }
        : categoria));
      setMenuIconoCategoriaAbierto(null);
      toast.success('Ícono de categoría actualizado.');
    } catch (error: unknown) {
      if (esPlanLimitReached(error)) {
        setModalUpgradeRecurso('personalizacion');
        setModalUpgradeOpen(true);
        return;
      }
      toast.error(getErrorMessage(error, 'No se pudo actualizar el ícono.'));
    } finally {
      setGuardandoHorarioId(null);
    }
  };

  const cargarImagenNueva = async (file?: File | null) => {
    if (!file) return;
    try {
      setProcesandoImagen(true);
      const fotoUrl = await prepararImagenProducto(file);
      setNuevoItem(prev => ({ ...prev, fotoUrl }));
      toast.success('Imagen preparada y lista para guardar.');
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, 'No se pudo procesar la imagen.'));
    } finally {
      setProcesandoImagen(false);
    }
  };

  const actualizarFotoArticulo = async (catId: string, item: ArticuloAPI, file?: File | null) => {
    if (!file) return;
    try {
      setActualizandoFotoId(item.id);
      const fotoUrl = await prepararImagenProducto(file);
      const actualizado = await api.actualizarArticulo(item.id, { foto_url: fotoUrl });
      setCategorias(prev => prev.map(categoria => categoria.id === catId
        ? { ...categoria, items: categoria.items.map(actual => actual.id === item.id ? { ...actual, foto_url: actualizado.foto_url } : actual) }
        : categoria));
      toast.success(`Foto de ${item.nombre} actualizada.`);
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, 'No se pudo actualizar la foto.'));
    } finally {
      setActualizandoFotoId(null);
    }
  };

  const quitarFotoArticulo = async (catId: string, item: ArticuloAPI) => {
    try {
      setActualizandoFotoId(item.id);
      await api.actualizarArticulo(item.id, { foto_url: '' });
      setCategorias(prev => prev.map(categoria => categoria.id === catId
        ? { ...categoria, items: categoria.items.map(actual => actual.id === item.id ? { ...actual, foto_url: undefined } : actual) }
        : categoria));
      setMenuItemAbierto(null);
      toast.success('Foto eliminada. Se mostrará el ícono de la categoría.');
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, 'No se pudo quitar la foto.'));
    } finally {
      setActualizandoFotoId(null);
    }
  };

  const asignarFranjaArticulo = async (categoriaId: string, articuloId: string, franjaId: string) => {
    try {
      setGuardandoHorarioId(articuloId);
      const actualizado = await api.asignarFranjaArticulo(articuloId, franjaId || null);
      setCategorias(prev => prev.map(categoria => (
        categoria.id === categoriaId
          ? {
              ...categoria,
              items: categoria.items.map(item => item.id === articuloId
                ? { ...item, franja_horaria_id: actualizado.franja_horaria_id }
                : item),
            }
          : categoria
      )));
      setMenuItemAbierto(null);
      toast.success(franjaId ? "Horario propio aplicado al producto." : "El producto vuelve a heredar el horario de su categoría.");
    } catch (error: unknown) {
      toast.error(getErrorMessage(error, "No se pudo actualizar el horario."));
    } finally {
      setGuardandoHorarioId(null);
    }
  };

  const agregarVariante = async (catId: string, articuloId: string) => {
    if (!nuevaVariante.nombre.trim()) {
      toast.error('Ingresá el nombre de la opción.');
      return;
    }
    const precio = parseFloat(nuevaVariante.precioAdicional || '0');
    if (isNaN(precio) || precio < 0) {
      toast.error('El precio adicional debe ser 0 o mayor.');
      return;
    }

    try {
      setGuardandoVariante(true);
      const creada = await api.crearVariante(articuloId, {
        nombre: nuevaVariante.nombre.trim(),
        grupo: nuevaVariante.grupo.trim() || undefined,
        precio_adicional: precio,
        seleccion_unica: nuevaVariante.seleccionUnica,
      });

      setCategorias(prev =>
        prev.map(c =>
          c.id === catId
            ? {
                ...c,
                items: c.items.map(i =>
                  i.id === articuloId
                    ? { ...i, variantes: [...(i.variantes || []), creada] }
                    : i
                ),
              }
            : c
        )
      );

      setNuevaVariante(prev => ({
        nombre: '',
        grupo: prev.grupo,
        precioAdicional: '0',
        seleccionUnica: prev.seleccionUnica,
      }));
      toast.success('Opción agregada.');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, 'Error al agregar opción.'));
    } finally {
      setGuardandoVariante(false);
    }
  };

  const eliminarVariante = async (catId: string, articuloId: string, varianteId: string) => {
    const ok = await confirmar({
      titulo: 'Eliminar opción',
      mensaje: '¿Eliminás esta opción de personalización?',
      labelAceptar: 'Sí, eliminar',
      variante: 'danger',
    });
    if (!ok) return;
    try {
      await api.eliminarVariante(varianteId);
      setCategorias(prev =>
        prev.map(c =>
          c.id === catId
            ? {
                ...c,
                items: c.items.map(i =>
                  i.id === articuloId
                    ? { ...i, variantes: (i.variantes || []).filter(v => v.id !== varianteId) }
                    : i
                ),
              }
            : c
        )
      );
      toast.success('Opción eliminada.');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, 'Error al eliminar opción.'));
    }
  };

  const totalItems = categorias.reduce((n, c) => n + c.items.length, 0);

  const limiteProductosAlcanzado = Boolean(
    estadoPlan &&
      estadoPlan.plan === "free" &&
      (estadoPlan.alcanzado?.productos === true ||
        (estadoPlan.disponibles?.productos !== undefined && estadoPlan.disponibles.productos <= 0))
  );

  const handleAbrirCrearItem = (catId?: string) => {
    if (limiteProductosAlcanzado) {
      setModalUpgradeRecurso("productos");
      setModalLimiteInfo({
        limite: estadoPlan?.limites?.productos ?? 30,
        uso: estadoPlan?.uso?.productos ?? totalItems,
      });
      setModalUpgradeOpen(true);
      return;
    }
    if (categorias.length === 0) {
      setMostrarFormCat(true);
      toast.error('Primero creá una categoría para poder agregar artículos.');
      return;
    }
    setMostrarFormItem(catId || categorias[0].id);
    setNuevoItem({ nombre: '', descripcion: '', precio: '', fotoUrl: '' });
    setNuevoItemErrors({});
  };

  const handleAbrirImportar = () => {
    if (estadoPlan && estadoPlan.plan === "free") {
      setModalUpgradeRecurso("carga_masiva");
      setModalLimiteInfo({
        limite: 0,
        uso: 0,
      });
      setModalUpgradeOpen(true);
      return;
    }
    setModalImportarAbierto(true);
  };

  const handlePreciosActualizados = async (actualizados: number) => {
    await cargarCarta();
    toast.success(
      `Se ${actualizados === 1 ? "actualizó" : "actualizaron"} ${actualizados} ${
        actualizados === 1 ? "producto" : "productos"
      }.`
    );
  };

  return (
    <div className="h-full space-y-24 overflow-y-auto bg-ghost-fog/45 p-16 font-inter sm:p-24 md:p-32">
      <div className="flex flex-col gap-14">
        <div className="min-w-0">
          <h2 className="text-24 font-semibold tracking-[-0.02em] text-ash-graphite sm:text-32">
            Gestión de Carta
          </h2>
          <div className="mt-4 flex flex-wrap items-center gap-8">
            <p className="text-13 text-sage-green sm:text-14">
              {loading ? 'Cargando carta...' : `${categorias.length} categorías · ${totalItems} ítems`}
            </p>
            {limiteProductosAlcanzado && (
              <span
                className="inline-flex items-center gap-4 rounded-full border border-stone/30 bg-ghost-fog px-10 py-2 text-11 font-mono font-medium text-stone"
                title="Límite del plan Free alcanzado (30 productos)"
              >
                <span className="material-symbols-outlined text-14">lock</span>
                Límite alcanzado
              </span>
            )}
          </div>
        </div>
        <div className="grid grid-cols-2 gap-6 sm:flex sm:flex-wrap sm:items-center sm:gap-8">
          <button
            type="button"
            onClick={() => setMostrarFranjasHorarias(true)}
            className="group flex h-40 min-w-0 items-center justify-center gap-6 whitespace-nowrap rounded-full border border-concrete bg-canvas-white px-10 pr-14 text-11 font-semibold text-ash-graphite shadow-xs transition-all hover:border-stone hover:bg-vanilla-cream sm:px-12 sm:pr-16 sm:text-12"
          >
            <span className="flex h-28 w-28 shrink-0 items-center justify-center rounded-full bg-ghost-fog text-plain-green transition-colors group-hover:bg-canvas-white">
              <span className="material-symbols-outlined text-16">schedule</span>
            </span>
            Horarios
          </button>
          <button
            type="button"
            onClick={handleAbrirImportar}
            className="group flex h-40 min-w-0 items-center justify-center gap-6 whitespace-nowrap rounded-full border border-concrete bg-canvas-white px-10 pr-14 text-11 font-semibold text-ash-graphite shadow-xs transition-all hover:border-stone hover:bg-vanilla-cream sm:px-12 sm:pr-16 sm:text-12"
          >
            <span className="flex h-28 w-28 shrink-0 items-center justify-center rounded-full bg-ghost-fog text-plain-green transition-colors group-hover:bg-canvas-white">
              <span className="material-symbols-outlined text-16">upload_file</span>
            </span>
            Importar CSV/Excel
          </button>
          <button
            type="button"
            onClick={() => setMostrarAjustePrecios(true)}
            disabled={loading || totalItems === 0}
            className="group flex h-40 min-w-0 items-center justify-center gap-6 whitespace-nowrap rounded-full border border-concrete bg-canvas-white px-10 pr-14 text-11 font-semibold text-ash-graphite shadow-xs transition-all hover:border-stone hover:bg-vanilla-cream disabled:cursor-not-allowed disabled:opacity-40 sm:px-12 sm:pr-16 sm:text-12"
            title={totalItems === 0 ? "Agregá productos antes de ajustar precios" : undefined}
          >
            <span className="flex h-28 w-28 shrink-0 items-center justify-center rounded-full bg-ghost-fog text-plain-green transition-colors group-hover:bg-canvas-white">
              <span className="material-symbols-outlined text-16">percent</span>
            </span>
            Ajustar precios
          </button>
          <button
            type="button"
            onClick={() => setMostrarFormCat(true)}
            className="group flex h-40 min-w-0 items-center justify-center gap-6 whitespace-nowrap rounded-full border border-plain-green bg-canvas-white px-10 pr-14 text-11 font-semibold text-plain-green shadow-xs transition-all hover:bg-vanilla-cream sm:px-12 sm:pr-16 sm:text-12"
          >
            <span className="flex h-28 w-28 shrink-0 items-center justify-center rounded-full bg-ghost-fog text-plain-green transition-colors group-hover:bg-canvas-white">
              <span className="material-symbols-outlined text-16">add</span>
            </span>
            Nueva categoría
          </button>
          <button
            type="button"
            onClick={() => handleAbrirCrearItem()}
            className="col-span-2 flex h-40 min-w-0 items-center justify-center gap-6 whitespace-nowrap rounded-full bg-plain-green px-12 pr-16 text-12 font-semibold text-canvas-white shadow-sm transition-all hover:bg-plain-green-muted sm:col-span-1 sm:px-14 sm:pr-20 sm:text-13"
          >
            <span className="flex h-28 w-28 shrink-0 items-center justify-center rounded-full bg-canvas-white/10 text-canvas-white">
              <span className="material-symbols-outlined text-16">add</span>
            </span>
            Nuevo artículo
          </button>
        </div>
      </div>

      <ImportarCartaModal
        isOpen={modalImportarAbierto}
        onClose={() => setModalImportarAbierto(false)}
        onImportacionCompletada={() => void cargarCarta()}
      />

      {mostrarAjustePrecios && (
        <AjustePreciosModal
          categorias={categorias}
          onClose={() => setMostrarAjustePrecios(false)}
          onApplied={handlePreciosActualizados}
        />
      )}

      {mostrarFranjasHorarias && (
        <FranjasHorariasModal
          categorias={categorias}
          franjas={franjas}
          onClose={() => setMostrarFranjasHorarias(false)}
          onChanged={cargarCarta}
        />
      )}

      {errorMsg && (
        <div className="p-10 bg-red-50 border border-alert-red/30 rounded text-12 text-alert-red flex items-center justify-between gap-12">
          <span>{errorMsg}</span>
          <button
            onClick={() => void cargarCarta()}
            className="text-12 font-medium underline hover:no-underline shrink-0"
          >
            Reintentar
          </button>
        </div>
      )}

      {mostrarFormCat && (
        <div className="space-y-14 rounded-xl border border-concrete bg-canvas-white p-16 shadow-sm sm:p-20">
          <div className="flex items-start justify-between gap-12">
            <div>
              <h3 className="text-15 font-semibold text-ash-graphite">Configuración de categorías</h3>
              <p className="mt-2 text-11 text-sage-green">Creá categorías y definí cuándo estarán disponibles.</p>
            </div>
            <button
              onClick={() => setMostrarFormCat(false)}
              className="flex h-44 w-44 shrink-0 items-center justify-center rounded-lg text-sage-green hover:bg-ghost-fog hover:text-ash-graphite"
              aria-label="Cerrar configuración de categorías"
            >
              <span className="material-symbols-outlined text-20">close</span>
            </button>
          </div>

          <div className="flex flex-col items-stretch gap-8 sm:flex-row sm:items-center">
            <input
              className="h-44 flex-1 rounded-lg border border-concrete bg-canvas-white px-12 text-13 outline-none focus:border-plain-green"
              placeholder="Nombre de la nueva categoría"
              value={nuevaCatNombre}
              onChange={e => setNuevaCatNombre(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && agregarCategoria()}
              autoFocus
            />
            <button
              onClick={agregarCategoria}
              className="h-44 rounded-lg bg-plain-green px-16 text-12 font-semibold text-canvas-white hover:bg-plain-green-muted"
            >
              Crear categoría
            </button>
          </div>

          {categorias.length > 0 && (
            <div className="divide-y divide-ghost-fog rounded-lg border border-concrete">
              {categorias.map(categoria => (
                <div key={categoria.id} className="flex flex-col gap-10 px-12 py-10 sm:flex-row sm:items-center">
                  <div className="relative" onClick={(event) => event.stopPropagation()}>
                    <button
                      type="button"
                      onClick={() => {
                        if (estadoPlan?.plan !== 'pro') {
                          setModalUpgradeRecurso('personalizacion');
                          setModalUpgradeOpen(true);
                          return;
                        }
                        setMenuIconoCategoriaAbierto(actual => actual === categoria.id ? null : categoria.id);
                        setMenuHorarioCategoriaAbierto(null);
                      }}
                      className="relative flex h-44 w-44 shrink-0 items-center justify-center rounded-lg border border-concrete bg-ghost-fog text-ash-graphite transition-colors hover:border-stone hover:bg-canvas-white"
                      title={estadoPlan?.plan === 'pro' ? 'Elegir ícono de categoría' : 'Íconos de categoría disponibles con Pro'}
                      aria-label={`Elegir ícono para ${categoria.nombre}`}
                    >
                      <span className="material-symbols-outlined text-21">{categoria.icono || 'restaurant'}</span>
                      {estadoPlan?.plan !== 'pro' && (
                        <span className="material-symbols-outlined absolute -bottom-4 -right-4 rounded-full bg-ash-graphite p-2 text-9 text-canvas-white">lock</span>
                      )}
                    </button>
                    {menuIconoCategoriaAbierto === categoria.id && (
                      <div className="absolute left-0 top-[50px] z-50 w-[250px] rounded-xl border border-concrete bg-canvas-white p-12 shadow-xl sm:left-auto sm:right-0">
                        <p className="mb-8 text-10 font-semibold uppercase tracking-[0.08em] text-sage-green">Ícono de la categoría</p>
                        <div className="grid grid-cols-5 gap-6">
                          {ICONOS_CATEGORIA.map(icono => (
                            <button
                              key={icono}
                              type="button"
                              disabled={guardandoHorarioId === categoria.id}
                              onClick={() => void asignarIconoCategoria(categoria.id, icono)}
                              className={`flex h-40 w-40 items-center justify-center rounded-lg border transition-colors ${categoria.icono === icono ? 'border-plain-green bg-plain-green/10 text-plain-green' : 'border-concrete text-sage-green hover:border-stone hover:text-ash-graphite'}`}
                              aria-label={`Usar ícono ${icono}`}
                            >
                              <span className="material-symbols-outlined text-20">{icono}</span>
                            </button>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-12 font-semibold text-ash-graphite">{categoria.nombre}</p>
                    <p className="mt-2 text-10 text-sage-green">
                      {categoria.items.length} {categoria.items.length === 1 ? 'artículo' : 'artículos'}
                    </p>
                  </div>
                  <div className="flex items-center gap-8">
                    <SelectorFranjaCategoria
                      categoria={categoria}
                      franjas={franjas}
                      abierto={menuHorarioCategoriaAbierto === categoria.id}
                      guardando={guardandoHorarioId === categoria.id}
                      onToggle={() => {
                        setMenuHorarioCategoriaAbierto(actual => actual === categoria.id ? null : categoria.id);
                        setMenuItemAbierto(null);
                        setMenuIconoCategoriaAbierto(null);
                      }}
                      onSelect={franjaId => void asignarFranjaCategoria(categoria.id, franjaId)}
                    />
                    <button
                      type="button"
                      onClick={() => void eliminarCategoria(categoria.id)}
                      className="flex h-44 w-44 shrink-0 items-center justify-center rounded-lg text-sage-green transition-colors hover:bg-warm-pink/20 hover:text-alert-red"
                      title={`Eliminar categoría ${categoria.nombre}`}
                      aria-label={`Eliminar categoría ${categoria.nombre}`}
                    >
                      <span className="material-symbols-outlined text-18">delete</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {mostrarFormItem && (
        <div className="space-y-12 rounded-xl border border-concrete bg-canvas-white p-16 shadow-sm sm:p-20">
          <div className="flex items-start justify-between gap-12">
            <div>
              <h3 className="text-15 font-semibold text-ash-graphite">Nuevo artículo</h3>
              <p className="mt-2 text-11 text-sage-green">Completá sus datos y elegí la categoría a la que pertenece.</p>
            </div>
            <button
              type="button"
              onClick={() => {
                setMostrarFormItem(null);
                setNuevoItemErrors({});
              }}
              className="flex h-44 w-44 shrink-0 items-center justify-center rounded-lg text-sage-green hover:bg-ghost-fog hover:text-ash-graphite"
              aria-label="Cerrar formulario de artículo"
            >
              <span className="material-symbols-outlined text-20">close</span>
            </button>
          </div>

          <div className="grid grid-cols-1 gap-8 md:grid-cols-[minmax(0,1.4fr)_minmax(180px,0.8fr)_minmax(140px,0.5fr)]">
            <div className="space-y-4">
              <input
                className={`h-44 w-full rounded-lg border bg-canvas-white px-10 text-13 outline-none focus:border-plain-green ${
                  nuevoItemErrors.nombre ? "border-alert-red" : "border-concrete"
                }`}
                placeholder="Nombre del artículo *"
                value={nuevoItem.nombre}
                onChange={e => {
                  setNuevoItem(p => ({ ...p, nombre: e.target.value }));
                  setNuevoItemErrors(p => ({ ...p, nombre: undefined }));
                }}
                aria-invalid={Boolean(nuevoItemErrors.nombre)}
                autoFocus
              />
              {nuevoItemErrors.nombre && (
                <p className="text-11 text-alert-red">{nuevoItemErrors.nombre}</p>
              )}
            </div>
            <select
              className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-10 text-13 text-ash-graphite outline-none focus:border-plain-green"
              value={mostrarFormItem}
              onChange={event => setMostrarFormItem(event.target.value)}
              aria-label="Categoría del artículo"
            >
              {categorias.map(categoria => (
                <option key={categoria.id} value={categoria.id}>{categoria.nombre}</option>
              ))}
            </select>
            <div className="space-y-4">
              <input
                type="number"
                min="0.01"
                step="0.01"
                className={`h-44 w-full rounded-lg border bg-canvas-white px-12 text-13 outline-none focus:border-plain-green ${
                  nuevoItemErrors.precio ? "border-alert-red" : "border-concrete"
                }`}
                placeholder="Precio *"
                value={nuevoItem.precio}
                onChange={e => {
                  setNuevoItem(p => ({ ...p, precio: e.target.value }));
                  setNuevoItemErrors(p => ({ ...p, precio: undefined }));
                }}
                aria-invalid={Boolean(nuevoItemErrors.precio)}
              />
              {nuevoItemErrors.precio && (
                <p className="text-11 text-alert-red">{nuevoItemErrors.precio}</p>
              )}
            </div>
          </div>
          <div className="space-y-4">
            <input
              className="h-44 w-full rounded-lg border border-concrete bg-canvas-white px-10 text-13 outline-none focus:border-plain-green"
              placeholder="Descripción (opcional)"
              value={nuevoItem.descripcion}
              maxLength={MAX_DESCRIPCION_ARTICULO}
              aria-describedby="descripcion-articulo-contador"
              onChange={e => setNuevoItem(p => ({ ...p, descripcion: e.target.value }))}
            />
            <p id="descripcion-articulo-contador" className="text-right text-10 text-sage-green" aria-live="polite">
              {Array.from(nuevoItem.descripcion).length}/{MAX_DESCRIPCION_ARTICULO} caracteres
            </p>
          </div>
          <div className="flex flex-col gap-10 rounded-xl border border-dashed border-concrete bg-ghost-fog/35 p-12 sm:flex-row sm:items-center">
            <div className="flex h-[88px] w-full shrink-0 items-center justify-center overflow-hidden rounded-lg border border-concrete bg-canvas-white sm:w-[116px]">
              {nuevoItem.fotoUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={nuevoItem.fotoUrl} alt="Vista previa del plato" className="h-full w-full object-cover" />
              ) : (
                <span className="material-symbols-outlined text-28 text-sage-green">add_photo_alternate</span>
              )}
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-12 font-semibold text-ash-graphite">Foto del plato</p>
              <p className="mt-2 text-10 leading-relaxed text-sage-green">PNG o JPG, hasta 5 MB. La recortamos automáticamente para que todas las tarjetas mantengan la misma proporción.</p>
            </div>
            <label className="flex h-44 shrink-0 cursor-pointer items-center justify-center gap-6 rounded-lg border border-concrete bg-canvas-white px-14 text-11 font-semibold text-ash-graphite hover:border-stone">
              <span className="material-symbols-outlined text-17">upload</span>
              {procesandoImagen ? 'Procesando...' : nuevoItem.fotoUrl ? 'Cambiar foto' : 'Subir foto'}
              <input
                type="file"
                accept="image/png,image/jpeg"
                disabled={procesandoImagen}
                onChange={(event) => void cargarImagenNueva(event.target.files?.[0])}
                className="sr-only"
              />
            </label>
          </div>
          <div className="flex items-center justify-end gap-8">
            <button
              type="button"
              onClick={() => {
                setMostrarFormItem(null);
                setNuevoItemErrors({});
              }}
              className="h-44 rounded-lg px-12 text-12 text-sage-green hover:bg-ghost-fog hover:text-ash-graphite"
            >
              Cancelar
            </button>
            <button
              type="button"
              onClick={() => agregarItem(mostrarFormItem)}
              className="h-44 rounded-lg bg-plain-green px-16 text-12 font-semibold text-canvas-white hover:bg-plain-green-muted"
            >
              Crear artículo
            </button>
          </div>
        </div>
      )}

      {/* Skeleton de carga */}
      {loading && (
        <div className="space-y-16">
          {[1, 2, 3].map(i => (
            <div key={i} className="overflow-hidden rounded-xl border border-concrete bg-canvas-white shadow-sm">
              <Skeleton className="h-72 w-full rounded-none" />
              <div>
                {[1, 2].map(j => (
                  <div key={j} className="flex items-center justify-between px-20 py-16">
                    <Skeleton className="h-12 w-1/3" />
                    <Skeleton className="h-12 w-20" />
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Catálogo en formato tabla, con agrupación operativa por categoría */}
      {!loading && (
        <div className="rounded-2xl border border-concrete bg-canvas-white shadow-sm">
          {categorias.length > 0 && (
            <div className="hidden grid-cols-[minmax(260px,2.2fr)_minmax(120px,0.85fr)_100px_minmax(125px,0.85fr)_44px] items-center gap-16 rounded-t-2xl border-b border-concrete bg-ghost-fog/55 px-20 py-12 text-10 font-semibold uppercase tracking-[0.08em] text-sage-green lg:grid">
              <span>Artículo</span>
              <span>Categoría</span>
              <span>Precio</span>
              <span>Variantes</span>
              <span className="sr-only">Acciones</span>
            </div>
          )}
          {categorias.flatMap(cat =>
            cat.items.map(item => {
                  const visible = item.activo !== false;
                  const gruposVariantes = new Set(
                    (item.variantes || []).map(variante => variante.grupo?.trim() || 'General')
                  ).size;

                  return (
                  <div key={item.id} className="border-b border-ghost-fog last:border-b-0">
                    <div className={`relative grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-14 gap-y-12 px-16 py-14 transition-colors hover:bg-ghost-fog/35 sm:px-20 lg:grid-cols-[minmax(260px,2.2fr)_minmax(120px,0.85fr)_100px_minmax(125px,0.85fr)_44px] lg:gap-16 ${!visible ? 'bg-ghost-fog/15' : ''}`}>
                      <div className="col-span-2 flex min-w-0 items-center gap-12 pr-44 lg:col-span-1 lg:pr-0">
                        <label
                          role={item.foto_url ? 'img' : undefined}
                          aria-label={item.foto_url ? `Foto de ${item.nombre}` : undefined}
                          className={`group/photo relative flex h-52 w-52 shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-xl border border-concrete bg-ghost-fog bg-cover bg-center text-sage-green ${!visible ? 'grayscale' : ''}`}
                          style={item.foto_url ? { backgroundImage: `url(${item.foto_url})` } : undefined}
                          title="Subir o reemplazar foto"
                        >
                          {!item.foto_url && (
                            <span className="material-symbols-outlined text-22">{cat.icono || 'restaurant'}</span>
                          )}
                          <span className="absolute inset-0 flex items-center justify-center bg-system-black/55 text-canvas-white opacity-0 transition-opacity group-hover/photo:opacity-100">
                            <span className="material-symbols-outlined text-19">photo_camera</span>
                          </span>
                          {actualizandoFotoId === item.id && (
                            <span className="absolute inset-0 flex items-center justify-center bg-canvas-white/80">
                              <span className="material-symbols-outlined animate-spin text-19 text-plain-green">progress_activity</span>
                            </span>
                          )}
                          <input
                            type="file"
                            accept="image/png,image/jpeg"
                            disabled={actualizandoFotoId === item.id}
                            onChange={(event) => void actualizarFotoArticulo(cat.id, item, event.target.files?.[0])}
                            className="sr-only"
                          />
                        </label>
                        <div className="min-w-0">
                          <div className="flex min-w-0 items-center gap-6">
                            <span className="truncate text-14 font-semibold text-ash-graphite sm:text-15">{item.nombre}</span>
                            {item.franja_horaria_id && (
                              <span
                                className="material-symbols-outlined shrink-0 text-15 text-sage-green"
                                title={franjas.find(franja => franja.id === item.franja_horaria_id)?.nombre || 'Horario propio'}
                              >
                                schedule
                              </span>
                            )}
                          </div>
                          <p className="mt-3 truncate text-11 text-sage-green" title={item.descripcion || undefined}>
                            {visible ? 'Visible en la carta digital' : 'Oculto de la carta digital'}
                            {item.descripcion ? ` · ${item.descripcion}` : ''}
                          </p>
                        </div>
                      </div>

                      <div className="min-w-0">
                        <span className="inline-flex max-w-full items-center rounded-md bg-ghost-fog px-8 py-4 text-10 font-medium text-sage-green">
                          <span className="truncate">{cat.nombre}</span>
                        </span>
                      </div>

                      <span className="whitespace-nowrap text-right text-14 font-semibold text-ash-graphite lg:text-left lg:text-15">
                        ${item.precio.toLocaleString('es-AR')}
                      </span>

                      <button
                        type="button"
                        onClick={() => {
                          if (articuloVariantesAbierto === item.id) {
                            setArticuloVariantesAbierto(null);
                          } else {
                            setArticuloVariantesAbierto(item.id);
                            setNuevaVariante({ nombre: '', grupo: '', precioAdicional: '0', seleccionUnica: false });
                          }
                        }}
                        className={`inline-flex min-h-44 items-center gap-4 justify-self-start rounded-lg px-4 text-11 font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-plain-green/30 ${
                          articuloVariantesAbierto === item.id
                            ? "text-plain-green"
                            : "text-sage-green hover:text-ash-graphite"
                        }`}
                        title="Gestionar opciones y variantes"
                        aria-label={`Gestionar opciones de ${item.nombre}`}
                        aria-expanded={articuloVariantesAbierto === item.id}
                      >
                        <span>{gruposVariantes > 0 ? `${gruposVariantes} ${gruposVariantes === 1 ? 'grupo' : 'grupos'}` : 'Sin variantes'}</span>
                        <span className={`material-symbols-outlined text-18 transition-transform ${articuloVariantesAbierto === item.id ? 'rotate-90' : ''}`}>chevron_right</span>
                      </button>

                      <div className="absolute right-10 top-10 lg:static" onClick={event => event.stopPropagation()}>
                          <button
                            type="button"
                            onClick={event => {
                              event.stopPropagation();
                              setMenuItemAbierto(actual => actual === item.id ? null : item.id);
                              setMenuHorarioCategoriaAbierto(null);
                            }}
                            className="flex h-44 w-44 items-center justify-center rounded-lg text-sage-green transition-colors hover:bg-ghost-fog hover:text-ash-graphite"
                            title="Más acciones del ítem"
                            aria-label={`Más acciones de ${item.nombre}`}
                            aria-expanded={menuItemAbierto === item.id}
                          >
                            <span className="material-symbols-outlined text-24">more_horiz</span>
                          </button>
                          {menuItemAbierto === item.id && (
                            <div className="absolute right-0 top-[48px] z-30 w-[250px] rounded-lg border border-concrete bg-canvas-white p-6 shadow-xl">
                              <button
                                type="button"
                                onClick={() => {
                                  setMenuItemAbierto(null);
                                  void toggleDisponible(cat.id, item);
                                }}
                                className="flex min-h-44 w-full items-center gap-8 rounded-md px-10 text-left text-12 font-medium text-ash-graphite transition-colors hover:bg-ghost-fog"
                              >
                                <span className="material-symbols-outlined text-18">{visible ? 'visibility_off' : 'visibility'}</span>
                                {visible ? 'Ocultar de la carta' : 'Mostrar en la carta'}
                              </button>
                              {item.foto_url && (
                                <button
                                  type="button"
                                  onClick={() => void quitarFotoArticulo(cat.id, item)}
                                  className="flex min-h-44 w-full items-center gap-8 rounded-md px-10 text-left text-12 font-medium text-ash-graphite transition-colors hover:bg-ghost-fog"
                                >
                                  <span className="material-symbols-outlined text-18">hide_image</span>
                                  Quitar foto
                                </button>
                              )}
                              <div className="my-4 border-t border-ghost-fog" />
                              <div className="px-6 pb-8 pt-6">
                                <p className="px-4 text-10 font-semibold uppercase tracking-[0.08em] text-sage-green">
                                  Horario del producto
                                </p>
                                <p className="mb-6 mt-2 px-4 text-10 leading-relaxed text-sage-green">
                                  Podés reemplazar el horario de la categoría solo para este producto.
                                </p>
                                <button
                                  type="button"
                                  disabled={guardandoHorarioId === item.id}
                                  onClick={() => void asignarFranjaArticulo(cat.id, item.id, '')}
                                  className={`flex min-h-44 w-full items-center gap-8 rounded-lg px-8 py-6 text-left transition-colors disabled:cursor-wait disabled:opacity-60 ${
                                    !item.franja_horaria_id ? 'bg-plain-green/[0.08]' : 'hover:bg-ghost-fog'
                                  }`}
                                >
                                  <span className="material-symbols-outlined text-18 text-sage-green">
                                    {cat.franja_horaria_id ? 'account_tree' : 'all_inclusive'}
                                  </span>
                                  <span className="min-w-0 flex-1 text-11 font-semibold text-ash-graphite">
                                    {cat.franja_horaria_id ? 'Heredar de la categoría' : 'Siempre disponible'}
                                  </span>
                                  {!item.franja_horaria_id && (
                                    <span className="material-symbols-outlined text-18 text-plain-green">check</span>
                                  )}
                                </button>
                                {franjas.map(franja => {
                                  const seleccionada = item.franja_horaria_id === franja.id;
                                  return (
                                    <button
                                      key={franja.id}
                                      type="button"
                                      disabled={guardandoHorarioId === item.id}
                                      onClick={() => void asignarFranjaArticulo(cat.id, item.id, franja.id)}
                                      className={`mt-2 flex min-h-44 w-full items-center gap-8 rounded-lg px-8 py-6 text-left transition-colors disabled:cursor-wait disabled:opacity-60 ${
                                        seleccionada ? 'bg-plain-green/[0.08]' : 'hover:bg-ghost-fog'
                                      }`}
                                    >
                                      <span className="material-symbols-outlined text-18 text-sage-green">schedule</span>
                                      <span className="min-w-0 flex-1">
                                        <span className="block truncate text-11 font-semibold text-ash-graphite">{franja.nombre}</span>
                                        <span className="block font-mono text-9 text-sage-green">{franja.hora_inicio} — {franja.hora_fin}</span>
                                      </span>
                                      {seleccionada && (
                                        <span className="material-symbols-outlined text-18 text-plain-green">check</span>
                                      )}
                                    </button>
                                  );
                                })}
                              </div>
                              <div className="my-4 border-t border-ghost-fog" />
                              <button
                                type="button"
                                onClick={() => {
                                  setMenuItemAbierto(null);
                                  void eliminarItem(cat.id, item.id);
                                }}
                                className="flex h-44 w-full items-center gap-8 rounded-md px-10 text-left text-12 font-medium text-alert-red transition-colors hover:bg-warm-pink/20"
                              >
                                <span className="material-symbols-outlined text-18">delete</span>
                                Eliminar ítem
                              </button>
                            </div>
                          )}
                      </div>
                    </div>

                    {/* Subpanel de variantes y personalización */}
                    {articuloVariantesAbierto === item.id && (
                      <div className="space-y-12 border-t border-ghost-fog bg-ghost-fog/70 px-16 py-16 sm:px-20">
                        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
                          <div className="flex items-center gap-6">
                            <span className="material-symbols-outlined text-16 text-ash-graphite">tune</span>
                            <h4 className="text-12 font-semibold text-ash-graphite">
                              Opciones y Variantes de &ldquo;{item.nombre}&rdquo;
                            </h4>
                          </div>
                          <span className="text-11 text-sage-green">
                            {item.variantes?.length ?? 0} {item.variantes?.length === 1 ? 'opción configurada' : 'opciones configuradas'}
                          </span>
                        </div>

                        {/* Lista de variantes existentes */}
                        {item.variantes && item.variantes.length > 0 ? (
                          <div className="space-y-6">
                            {item.variantes.map(v => (
                              <div
                                key={v.id}
                                className="flex flex-col gap-8 rounded-lg border border-concrete bg-canvas-white px-12 py-10 text-12 sm:flex-row sm:items-center sm:justify-between"
                              >
                                <div className="flex min-w-0 flex-wrap items-center gap-8">
                                  <span className="rounded-full border border-concrete bg-ghost-fog px-8 py-2 text-10 font-medium text-ash-graphite">
                                    {v.grupo || 'General'}
                                  </span>
                                  <span className="font-medium text-ash-graphite truncate">{v.nombre}</span>
                                  <span className="text-10 text-sage-green">
                                    ({v.seleccion_unica ? 'Radio · Única' : 'Checkbox · Múltiple'})
                                  </span>
                                </div>
                                <div className="flex shrink-0 items-center justify-between gap-10 sm:justify-end">
                                  <span className="font-mono font-medium text-ash-graphite">
                                    {v.precio_adicional > 0 ? `+$${v.precio_adicional.toLocaleString()}` : 'Sin cargo'}
                                  </span>
                                  <button
                                    type="button"
                                    onClick={() => eliminarVariante(cat.id, item.id, v.id)}
                                    className="flex h-44 w-44 items-center justify-center rounded-lg text-alert-red transition-colors hover:bg-warm-pink/20"
                                    title="Eliminar opción"
                                  >
                                    ✕
                                  </button>
                                </div>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <p className="text-12 text-sage-green italic">
                            No hay opciones agregadas aún. Añadí términos de cocción, tamaños o extras debajo.
                          </p>
                        )}

                        {/* Formulario para agregar variante */}
                        <div className="space-y-10 rounded-lg border border-concrete bg-canvas-white p-12">
                          <div className="text-11 font-semibold text-ash-graphite">
                            + Agregar nueva opción
                          </div>
                          <div className="grid grid-cols-1 sm:grid-cols-3 gap-8">
                            <input
                              className="h-44 rounded-lg border border-concrete bg-canvas-white px-10 text-12 outline-none focus:border-plain-green"
                              placeholder="Nombre (ej: Jugoso, Cheddar) *"
                              value={nuevaVariante.nombre}
                              onChange={e => setNuevaVariante(p => ({ ...p, nombre: e.target.value }))}
                            />
                            <input
                              className="h-44 rounded-lg border border-concrete bg-canvas-white px-10 text-12 outline-none focus:border-plain-green"
                              placeholder="Grupo (ej: Término, Extras)"
                              value={nuevaVariante.grupo}
                              onChange={e => setNuevaVariante(p => ({ ...p, grupo: e.target.value }))}
                            />
                            <input
                              type="number"
                              min="0"
                              step="0.01"
                              className="h-44 rounded-lg border border-concrete bg-canvas-white px-10 text-12 outline-none focus:border-plain-green"
                              placeholder="Precio adicional ($)"
                              value={nuevaVariante.precioAdicional}
                              onChange={e => setNuevaVariante(p => ({ ...p, precioAdicional: e.target.value }))}
                            />
                          </div>
                          <div className="flex flex-col justify-between gap-8 pt-2 sm:flex-row sm:items-center">
                            <button
                              type="button"
                              role="switch"
                              aria-checked={nuevaVariante.seleccionUnica}
                              onClick={() => setNuevaVariante(p => ({ ...p, seleccionUnica: !p.seleccionUnica }))}
                              className="group flex min-h-44 select-none items-center gap-10 rounded-lg px-8 text-left transition-colors hover:bg-canvas-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-plain-green/30 focus-visible:ring-offset-2"
                            >
                              <span
                                aria-hidden="true"
                                className={`relative h-24 w-44 shrink-0 rounded-full border transition-colors after:absolute after:left-[3px] after:top-[3px] after:h-[16px] after:w-[16px] after:rounded-full after:bg-canvas-white after:shadow-sm after:transition-transform ${
                                  nuevaVariante.seleccionUnica
                                    ? 'border-plain-green bg-plain-green after:translate-x-20'
                                    : 'border-concrete bg-stone/45'
                                }`}
                              />
                              <span className="min-w-0">
                                <span className="block text-12 font-semibold text-ash-graphite">Selección única</span>
                                <span className="mt-2 block text-10 leading-snug text-sage-green">Una sola opción permitida dentro del grupo</span>
                              </span>
                            </button>
                            <button
                              type="button"
                              disabled={guardandoVariante}
                              onClick={() => agregarVariante(cat.id, item.id)}
                              className="h-44 self-end whitespace-nowrap rounded-lg bg-plain-green px-14 text-12 font-semibold text-canvas-white hover:bg-plain-green-muted disabled:opacity-50 sm:self-auto"
                            >
                              {guardandoVariante ? 'Guardando...' : 'Agregar opción'}
                            </button>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                  );
            })
          )}

          {categorias.length === 0 && (
            <EmptyState
              icon="menu_book"
              title="Tu carta está vacía"
              description="Creá tu primera categoría para empezar a cargar productos."
              actionLabel="+ Nueva categoría"
              onAction={() => setMostrarFormCat(true)}
            />
          )}

          {categorias.length > 0 && totalItems === 0 && (
            <EmptyState
              icon="restaurant_menu"
              title="Todavía no hay artículos"
              description="Creá el primero y elegí a qué categoría pertenece."
              actionLabel="+ Nuevo artículo"
              onAction={() => handleAbrirCrearItem()}
            />
          )}
        </div>
      )}

      <UpgradeModal
        isOpen={modalUpgradeOpen}
        onClose={() => setModalUpgradeOpen(false)}
        recurso={modalUpgradeRecurso}
        limite={modalLimiteInfo.limite ?? (modalUpgradeRecurso === "productos" ? (estadoPlan?.limites?.productos ?? 30) : 0)}
        uso={modalLimiteInfo.uso ?? (modalUpgradeRecurso === "productos" ? (estadoPlan?.uso?.productos ?? totalItems) : 0)}
        onUpgradeSolicitado={() => void cargarPlan()}
      />
    </div>
  );
}
