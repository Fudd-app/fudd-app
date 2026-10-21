"use client";

import { useState, useRef, useCallback } from "react";
import { api, ResultadoImportacion, ErrorFila, getErrorMessage } from "@/lib/api";

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onImportacionCompletada: () => void;
}

type Paso = "plantilla" | "subir" | "resultado";

// ─── Generador de plantilla CSV en el browser ─────────────────────────────────

const CABECERA_CSV = "categoria,nombre,precio,descripcion";
const EJEMPLO_CSV = [
  CABECERA_CSV,
  "Bebidas,Agua mineral,500,",
  "Bebidas,Gaseosa 500ml,900,Coca-Cola o Pepsi",
  "Comidas,Hamburguesa clásica,2500,Con lechuga tomate y papas fritas",
  "Comidas,Veggie burger,2800,100% vegetal con cheddar vegano",
  "Postres,Brownie con helado,1800,Tibio con dos bochas de helado a elección",
].join("\n");

function descargarArchivo(nombre: string, contenido: string, tipo: string) {
  const blob = new Blob([contenido], { type: tipo });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = nombre;
  a.click();
  URL.revokeObjectURL(url);
}

// ─── Componente principal ─────────────────────────────────────────────────────

export default function ImportarCartaModal({ isOpen, onClose, onImportacionCompletada }: Props) {
  const [paso, setPaso] = useState<Paso>("plantilla");
  const [archivoSeleccionado, setArchivoSeleccionado] = useState<File | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const [importando, setImportando] = useState(false);
  const [resultado, setResultado] = useState<ResultadoImportacion | null>(null);
  const [errorGlobal, setErrorGlobal] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const resetear = useCallback(() => {
    setPaso("plantilla");
    setArchivoSeleccionado(null);
    setDragOver(false);
    setImportando(false);
    setResultado(null);
    setErrorGlobal("");
  }, []);

  const cerrar = useCallback(() => {
    resetear();
    onClose();
  }, [resetear, onClose]);

  const validarArchivo = (file: File): string | null => {
    const ext = file.name.split(".").pop()?.toLowerCase();
    if (ext !== "csv" && ext !== "xlsx") {
      return "Solo se aceptan archivos .csv y .xlsx";
    }
    if (file.size > 5 * 1024 * 1024) {
      return "El archivo no puede superar los 5 MB";
    }
    return null;
  };

  const seleccionarArchivo = (file: File) => {
    const error = validarArchivo(file);
    if (error) {
      setErrorGlobal(error);
      return;
    }
    setErrorGlobal("");
    setArchivoSeleccionado(file);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file) seleccionarArchivo(file);
  };

  const handleImportar = async () => {
    if (!archivoSeleccionado) return;
    setImportando(true);
    setErrorGlobal("");
    try {
      const res = await api.importarCarta(archivoSeleccionado);
      setResultado(res);
      setPaso("resultado");
      onImportacionCompletada(); // Refresca la lista de la carta en el dashboard
    } catch (err: unknown) {
      setErrorGlobal(getErrorMessage(err, "Error al importar la carta. Revisá el archivo e intentá nuevamente."));
    } finally {
      setImportando(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-16"
      role="dialog"
      aria-modal="true"
      aria-labelledby="importar-carta-title"
      onClick={(e) => { if (e.target === e.currentTarget) cerrar(); }}
    >
      <div
        className="flex w-full max-w-lg flex-col overflow-hidden rounded-2xl bg-canvas-white shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* ── Header ─────────────────────────────────────────── */}
        <div className="flex items-center justify-between border-b border-concrete px-24 py-20">
          <div>
            <h2 id="importar-carta-title" className="text-16 font-bold text-ash-graphite">
              Importar catálogo
            </h2>
            <p className="text-12 text-stone mt-2">
              {paso === "plantilla" && "Descargá la plantilla oficial para empezar"}
              {paso === "subir" && "Subí tu archivo con el catálogo completo"}
              {paso === "resultado" && "Importación finalizada"}
            </p>
          </div>
          <button
            onClick={cerrar}
            aria-label="Cerrar"
            className="flex h-36 w-36 items-center justify-center rounded-full text-stone hover:text-ash-graphite transition-colors text-20"
          >
            ✕
          </button>
        </div>

        {/* ── Pasos indicadores ──────────────────────────────── */}
        <div className="flex border-b border-concrete">
          {(["plantilla", "subir", "resultado"] as Paso[]).map((p, i) => (
            <div
              key={p}
              className={`flex-1 py-10 text-center text-11 font-bold uppercase tracking-wide transition-colors ${
                paso === p
                  ? "border-b-2 border-ash-graphite text-ash-graphite"
                  : "text-stone"
              }`}
            >
              {i + 1}. {p === "plantilla" ? "Plantilla" : p === "subir" ? "Subir archivo" : "Resultado"}
            </div>
          ))}
        </div>

        {/* ── Contenido ─────────────────────────────────────── */}
        <div className="p-24 space-y-20">

          {/* PASO 1: Descargar plantilla */}
          {paso === "plantilla" && (
            <div className="space-y-16">
              <p className="text-14 text-deep-forest leading-relaxed">
                Antes de subir tu carta, descargá la plantilla oficial con el formato correcto y datos de ejemplo. Podés editarla en Excel, Google Sheets o cualquier editor de planillas.
              </p>

              <div className="rounded-xl border border-concrete bg-ghost-fog p-16 space-y-8">
                <div className="text-11 font-bold uppercase tracking-wide text-stone">Columnas del template</div>
                <div className="grid grid-cols-2 gap-6">
                  {[
                    { col: "categoria", req: true, desc: "Nombre de la categoría" },
                    { col: "nombre", req: true, desc: "Nombre del artículo" },
                    { col: "precio", req: true, desc: "Precio sin símbolos (ej: 1500)" },
                    { col: "descripcion", req: false, desc: "Descripción visible al cliente" },
                  ].map(({ col, req, desc }) => (
                    <div key={col} className="flex gap-6 items-start">
                      <span className={`mt-1 h-6 w-6 rounded-full shrink-0 ${req ? "bg-ash-graphite" : "bg-concrete"}`} />
                      <div>
                        <code className="text-12 font-mono font-semibold text-ash-graphite">{col}</code>
                        {req && <span className="ml-4 text-10 text-stone uppercase">obligatorio</span>}
                        <p className="text-11 text-stone leading-tight">{desc}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex gap-10">
                <button
                  onClick={() => descargarArchivo("plantilla-carta.csv", EJEMPLO_CSV, "text/csv;charset=utf-8")}
                  className="flex-1 flex items-center justify-center gap-8 rounded-lg border-2 border-ash-graphite bg-canvas-white py-12 text-13 font-bold text-ash-graphite hover:bg-ghost-fog transition-colors"
                >
                  <span>⬇</span> Descargar CSV
                </button>
                <a
                  href="/plantilla-carta.xlsx"
                  download="plantilla-carta.xlsx"
                  className="flex-1 flex items-center justify-center gap-8 rounded-lg border-2 border-plain-green bg-canvas-white py-12 text-13 font-bold text-plain-green hover:bg-success/10 transition-colors"
                >
                  <span>⬇</span> Descargar Excel
                </a>
              </div>

              <button
                onClick={() => setPaso("subir")}
                className="w-full rounded-full bg-ash-graphite py-14 text-13 font-bold text-canvas-white hover:opacity-85 transition-opacity"
              >
                Ya tengo mi archivo → Continuar
              </button>
            </div>
          )}

          {/* PASO 2: Subir archivo */}
          {paso === "subir" && (
            <div className="space-y-16">
              {/* Zona de drop */}
              <div
                onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
                onDragLeave={() => setDragOver(false)}
                onDrop={handleDrop}
                onClick={() => inputRef.current?.click()}
                className={`flex cursor-pointer flex-col items-center justify-center gap-10 rounded-xl border-2 border-dashed py-32 transition-colors ${
                  dragOver
                    ? "border-ash-graphite bg-ghost-fog"
                    : archivoSeleccionado
                    ? "border-success bg-success/5"
                    : "border-concrete hover:border-ash-graphite"
                }`}
              >
                <span className="text-32">{archivoSeleccionado ? "✅" : "📄"}</span>
                {archivoSeleccionado ? (
                  <>
                    <p className="text-14 font-semibold text-ash-graphite">{archivoSeleccionado.name}</p>
                    <p className="text-12 text-stone">
                      {(archivoSeleccionado.size / 1024).toFixed(1)} KB — hacé clic para cambiar
                    </p>
                  </>
                ) : (
                  <>
                    <p className="text-14 font-semibold text-ash-graphite">
                      Arrastrá tu archivo o hacé clic para seleccionar
                    </p>
                    <p className="text-12 text-stone">CSV o XLSX · máx. 5 MB</p>
                  </>
                )}
                <input
                  ref={inputRef}
                  type="file"
                  accept=".csv,.xlsx"
                  className="hidden"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) seleccionarArchivo(file);
                  }}
                />
              </div>

              {errorGlobal && (
                <div className="rounded-lg border border-alert-red/30 bg-warm-pink/20 p-12 text-13 text-alert-red">
                  {errorGlobal}
                </div>
              )}

              <div className="flex gap-10">
                <button
                  onClick={() => setPaso("plantilla")}
                  className="flex-1 rounded-full border border-concrete py-12 text-13 font-semibold text-stone hover:text-ash-graphite transition-colors"
                >
                  ← Volver
                </button>
                <button
                  onClick={handleImportar}
                  disabled={!archivoSeleccionado || importando}
                  className="flex-1 flex items-center justify-center gap-8 rounded-full bg-ash-graphite py-12 text-13 font-bold text-canvas-white hover:opacity-85 transition-opacity disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {importando ? (
                    <><span className="animate-spin inline-block">⟳</span> Importando...</>
                  ) : (
                    "Importar catálogo"
                  )}
                </button>
              </div>
            </div>
          )}

          {/* PASO 3: Resultado */}
          {paso === "resultado" && resultado && (
            <div className="space-y-16">
              {/* Resumen */}
              <div className="grid grid-cols-2 gap-10">
                <div className="rounded-xl border border-concrete bg-success/10 p-16 text-center">
                  <p className="text-32 font-bold text-ash-graphite">{resultado.creados}</p>
                  <p className="text-12 text-stone mt-4">artículos importados</p>
                </div>
                <div className={`rounded-xl border p-16 text-center ${resultado.omitidos > 0 ? "border-alert-red/30 bg-warm-pink/10" : "border-concrete bg-ghost-fog"}`}>
                  <p className="text-32 font-bold text-ash-graphite">{resultado.omitidos}</p>
                  <p className="text-12 text-stone mt-4">filas omitidas</p>
                </div>
              </div>

              {resultado.creados > 0 && resultado.omitidos === 0 && (
                <div className="flex items-center gap-10 rounded-lg bg-success/15 px-16 py-12 text-13 font-semibold text-ash-graphite">
                  <span>✅</span> ¡Todo importado correctamente!
                </div>
              )}

              {/* Tabla de errores */}
              {resultado.errores.length > 0 && (
                <div className="space-y-8">
                  <p className="text-12 font-bold uppercase tracking-wide text-stone">
                    Detalle de filas rechazadas
                  </p>
                  <div className="max-h-[200px] overflow-y-auto rounded-lg border border-concrete">
                    <table className="w-full text-12">
                      <thead className="sticky top-0 bg-ghost-fog">
                        <tr>
                          <th className="px-12 py-8 text-left font-semibold text-stone w-16">Fila</th>
                          <th className="px-12 py-8 text-left font-semibold text-stone">Motivo</th>
                        </tr>
                      </thead>
                      <tbody>
                        {resultado.errores.map((e: ErrorFila, i: number) => (
                          <tr key={i} className="border-t border-concrete">
                            <td className="px-12 py-8 font-mono text-stone">{e.fila}</td>
                            <td className="px-12 py-8 text-deep-forest">{e.motivo}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              <div className="flex gap-10">
                {resultado.omitidos > 0 && (
                  <button
                    onClick={resetear}
                    className="flex-1 rounded-full border border-concrete py-12 text-13 font-semibold text-stone hover:text-ash-graphite transition-colors"
                  >
                    Subir otro archivo
                  </button>
                )}
                <button
                  onClick={cerrar}
                  className="flex-1 rounded-full bg-ash-graphite py-12 text-13 font-bold text-canvas-white hover:opacity-85 transition-opacity"
                >
                  Cerrar
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
