"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ConexionMcp, getErrorMessage } from "@/lib/api";
import { EmptyState, Skeleton, useConfirm, useToast } from "@/components/ui";

function formatearFecha(iso: string): string {
  return new Date(iso).toLocaleString("es-AR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function AsistentesIaTab() {
  const router = useRouter();
  const toast = useToast();
  const confirmar = useConfirm();

  const [conexiones, setConexiones] = useState<ConexionMcp[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorCarga, setErrorCarga] = useState("");
  const [revocandoId, setRevocandoId] = useState<string | null>(null);

  // setState solo dentro de callbacks de la promesa (no sincrónico en el efecto).
  const cargar = useCallback(() => {
    api
      .listarConexionesMcp()
      .then(setConexiones)
      .catch((err: unknown) => {
        setErrorCarga(getErrorMessage(err, "No pudimos cargar los asistentes conectados."));
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    cargar();
  }, [cargar]);

  const reintentar = () => {
    setLoading(true);
    setErrorCarga("");
    cargar();
  };

  const revocar = async (conexion: ConexionMcp) => {
    const ok = await confirmar({
      titulo: "Revocar acceso",
      mensaje: `${conexion.client_name} dejará de poder consultar los datos de tu negocio. Para volver a usarlo vas a tener que conectarlo de nuevo.`,
      labelAceptar: "Revocar",
      variante: "danger",
    });
    if (!ok) return;

    setRevocandoId(conexion.id);
    try {
      await api.revocarConexionMcp(conexion.id);
      setConexiones((actuales) => actuales.filter((c) => c.id !== conexion.id));
      toast.success("Acceso revocado");
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, "No pudimos revocar el acceso."));
    } finally {
      setRevocandoId(null);
    }
  };

  return (
    <section className="space-y-20">
      <header>
        <p className="text-11 font-mono uppercase tracking-[0.14em] text-sage-green">
          Apps conectadas
        </p>
        <h2 className="mt-4 text-24 font-semibold text-ash-graphite sm:text-32">
          Asistentes de IA
        </h2>
        <p className="mt-4 text-13 text-sage-green">
          Apps de IA (Claude, ChatGPT, Gemini, Perplexity…) que conectaste con tu cuenta. Pueden
          ver el estado del salón, la cocina y el turno, sin modificar nada.
        </p>
        <Link
          href="/ia"
          className="mt-8 inline-block text-13 font-semibold text-ash-graphite underline"
        >
          Cómo conectar un asistente
        </Link>
      </header>

      {loading ? (
        <div className="space-y-12">
          <Skeleton className="h-64 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      ) : errorCarga ? (
        <div role="alert" className="rounded-xl border border-concrete bg-canvas-white p-16">
          <p className="text-13 text-alert-red">{errorCarga}</p>
          <button
            type="button"
            onClick={reintentar}
            className="mt-12 h-44 rounded-lg border border-concrete bg-canvas-white px-16 text-12 font-semibold text-ash-graphite hover:bg-ghost-fog"
          >
            Reintentar
          </button>
        </div>
      ) : conexiones.length === 0 ? (
        <div className="rounded-xl border border-concrete bg-canvas-white">
          <EmptyState
            icon="smart_toy"
            title="Todavía no conectaste ningún asistente"
            description="Conectá Claude, ChatGPT u otro asistente para consultar tu negocio conversando."
            actionLabel="Cómo conectar"
            onAction={() => router.push("/ia")}
          />
        </div>
      ) : (
        <ul className="divide-y divide-concrete rounded-xl border border-concrete bg-canvas-white">
          {conexiones.map((c) => (
            <li
              key={c.id}
              className="flex flex-col gap-12 p-16 sm:flex-row sm:items-center sm:justify-between"
            >
              <div className="min-w-0">
                <p className="truncate text-15 font-semibold text-ash-graphite">{c.client_name}</p>
                <p className="mt-2 text-13 text-sage-green">
                  Conectada el {formatearFecha(c.creado_at)}
                </p>
                <p className="text-13 text-sage-green">
                  Último uso: {c.ultimo_uso_at ? formatearFecha(c.ultimo_uso_at) : "todavía no se usó"}
                </p>
              </div>
              <button
                type="button"
                onClick={() => void revocar(c)}
                disabled={revocandoId === c.id}
                className="h-44 shrink-0 rounded-lg border border-concrete bg-canvas-white px-16 text-12 font-semibold text-alert-red transition-colors hover:bg-ghost-fog disabled:cursor-not-allowed disabled:opacity-40"
              >
                {revocandoId === c.id ? "Revocando…" : "Revocar"}
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
