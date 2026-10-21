"use client";

import React, { useEffect, useState } from "react";

interface StepBusinessProps {
  data: {
    nombreNegocio: string;
    nombreFantasia: string;
    slug: string;
    rubro: string;
    descripcion: string;
  };
  errors?: Partial<Record<"nombreNegocio" | "slug" | "rubro", string>>;
  onChange: (fields: Partial<{
    nombreNegocio: string;
    nombreFantasia: string;
    slug: string;
    rubro: string;
    descripcion: string;
  }>) => void;
  onNext: () => void;
}

function slugify(value: string) {
  return value
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export default function StepBusiness({ data, errors = {}, onChange, onNext }: StepBusinessProps) {
  // El prefijo del slug tiene que ser el dominio donde realmente corre la app
  // (localhost:3000, el dominio de Render, etc.), no un dominio fijo de ejemplo.
  // Se resuelve después del montaje porque `window` no existe en el render del servidor.
  const [host, setHost] = useState("");

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      setHost(window.location.host);
    }, 0);

    return () => window.clearTimeout(timeoutId);
  }, []);

  const handleNombreChange = (val: string) => {
    const slugActualSugerido = slugify(data.nombreNegocio);
    const nuevoSlugSugerido = slugify(val);
    const usuarioEditoSlug = data.slug !== "" && data.slug !== slugActualSugerido;

    onChange({
      nombreNegocio: val,
      slug: usuarioEditoSlug ? data.slug : nuevoSlugSugerido,
    });
  };

  const handleSlugChange = (val: string) => {
    onChange({ slug: slugify(val) });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!data.nombreNegocio.trim()) return;
    onNext();
  };

  return (
    <div className="space-y-24 font-inter">
      <form onSubmit={handleSubmit} className="space-y-20">
        <div className="bg-vanilla-cream p-20 rounded-lg border border-system-black shadow-sm space-y-16">
          <div className="flex items-center gap-8 text-ash-graphite mb-4">
            <span className="material-symbols-outlined text-20">storefront</span>
            <h3 className="font-medium text-11 font-mono uppercase tracking-wider">
              Identidad del Negocio
            </h3>
          </div>

          <div className="space-y-14">
            <div className="space-y-6">
              <label className="block text-11 font-bold">
                Nombre de la Empresa o Negocio
              </label>
              <input
                type="text"
                required
                value={data.nombreNegocio}
                onChange={(e) => handleNombreChange(e.target.value)}
                placeholder="Ej: Café Bar Central"
                aria-invalid={Boolean(errors.nombreNegocio)}
                className={`w-full h-40 px-12 bg-canvas-white border rounded-md focus:border-plain-green outline-none transition-all text-14 ${
                  errors.nombreNegocio ? "border-alert-red" : "border-ash-graphite"
                }`}
              />
              {errors.nombreNegocio && (
                <p className="text-11 text-alert-red px-1">{errors.nombreNegocio}</p>
              )}
            </div>

            <div className="space-y-6">
              <label className="block text-11 font-bold">
                Link público del negocio
              </label>
              <div className="grid grid-cols-1 sm:grid-cols-[minmax(0,1fr)_minmax(160px,220px)] gap-8 min-w-0">
                <div className="min-w-0">
                  <span className="block text-10 font-mono text-sage-green uppercase tracking-wider px-1 mb-4">
                    Dominio
                  </span>
                  <div className="h-40 px-12 bg-ghost-fog border border-ash-graphite rounded-md text-12 font-mono text-sage-green flex items-center min-w-0">
                    <span className="truncate">{host ? `${host}/` : "fudd/"}</span>
                  </div>
                </div>
                <div>
                  <span className="block text-10 font-mono text-sage-green uppercase tracking-wider px-1 mb-4">
                    Nombre en URL
                  </span>
                  <input
                    type="text"
                    required
                    value={data.slug}
                    onChange={(e) => handleSlugChange(e.target.value)}
                    placeholder={slugify(data.nombreNegocio) || "tu-negocio"}
                    aria-invalid={Boolean(errors.slug)}
                    className={`w-full h-40 px-12 bg-canvas-white border rounded-md focus:border-plain-green outline-none transition-all text-14 font-mono ${
                      errors.slug ? "border-alert-red" : "border-ash-graphite"
                    }`}
                  />
                  {errors.slug && (
                    <p className="text-11 text-alert-red px-1 mt-4">{errors.slug}</p>
                  )}
                </div>
              </div>
              <p className="text-11 text-sage-green font-mono break-all px-1">
                {host ? `${host}/${data.slug || "tu-negocio"}` : data.slug || "tu-negocio"}
              </p>
            </div>

            <div className="space-y-6">
              <label className="block text-11 font-bold">
                Rubro Principal
              </label>
              <select 
                value={data.rubro || "cafeteria"}
                onChange={(e) => onChange({ rubro: e.target.value })}
                aria-invalid={Boolean(errors.rubro)}
                className={`w-full h-40 px-12 bg-canvas-white border rounded-md focus:border-plain-green outline-none transition-all text-14 ${
                  errors.rubro ? "border-alert-red" : "border-ash-graphite"
                }`}
              >
                <option value="cafeteria">Cafetería</option>
                <option value="restaurante">Restaurante</option>
                <option value="bar">Bar / Pub / Cervecería</option>
                <option value="comida_rapida">Comida Rápida</option>
                <option value="otro">Otro</option>
              </select>
              {errors.rubro && (
                <p className="text-11 text-alert-red px-1">{errors.rubro}</p>
              )}
            </div>
          </div>
        </div>

        <div className="bg-vanilla-cream p-20 rounded-lg border border-system-black shadow-sm space-y-8">
          <label className="block text-11 font-bold">
            Descripción o Slogan (Opcional)
          </label>
          <textarea
            rows={3}
            value={data.descripcion}
            onChange={(e) => onChange({ descripcion: e.target.value })}
            placeholder="Especialistas en café de especialidad y pastelería artesanal..."
            className="w-full p-12 bg-canvas-white border border-ash-graphite rounded-md focus:border-plain-green outline-none transition-all text-14 resize-none"
          />
        </div>

        <button
          type="submit"
          className="landing-cta mt-24 w-full disabled:cursor-not-allowed disabled:opacity-40"
        >
          Continuar
        </button>
      </form>
    </div>
  );
}
