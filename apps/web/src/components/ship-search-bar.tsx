"use client";

import { useForm } from "react-hook-form";
import { useShallow } from "zustand/react/shallow";
import { useShipSearchStore } from "@/stores/ship-search-store";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Search, RotateCcw, Filter } from "lucide-react";

interface SearchFormValues {
  q: string;
  imo: string;
  mmsi: string;
  shipType: string;
}

const SHIP_TYPES = [
  { value: "", label: "Todos los tipos" },
  { value: "Cargo", label: "Cargo (Carga general)" },
  { value: "Tanker", label: "Tanker (Tanquero)" },
  { value: "Passenger", label: "Passenger (Pasajeros)" },
  { value: "Fishing", label: "Fishing (Pesquero)" },
  { value: "Tug", label: "Tug (Remolcador)" },
  { value: "Pleasure", label: "Pleasure Craft (Recreo)" },
  { value: "High Speed Craft", label: "High Speed Craft (HSC)" },
  { value: "Other", label: "Otros" },
];

export function ShipSearchBar() {
  const { filters, setFilters, searchShips, clearFilters, isLoading } = useShipSearchStore(
    useShallow((state) => ({
      filters: state.filters,
      setFilters: state.setFilters,
      searchShips: state.searchShips,
      clearFilters: state.clearFilters,
      isLoading: state.isLoading,
    }))
  );

  const { register, handleSubmit, reset } = useForm<SearchFormValues>({
    defaultValues: {
      q: filters.q || "",
      imo: filters.imo ? String(filters.imo) : "",
      mmsi: filters.mmsi ? String(filters.mmsi) : "",
      shipType: filters.shipType || "",
    },
  });

  const onSubmit = (data: SearchFormValues) => {
    setFilters({
      q: data.q || undefined,
      imo: data.imo ? parseInt(data.imo, 10) : undefined,
      mmsi: data.mmsi ? parseInt(data.mmsi, 10) : undefined,
      shipType: data.shipType || undefined,
    });
    void searchShips(1);
  };

  const handleReset = () => {
    reset({ q: "", imo: "", mmsi: "", shipType: "" });
    clearFilters();
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="bg-card border rounded-lg p-5 shadow-sm space-y-4"
    >
      <div className="flex items-center gap-2 border-b pb-3 text-sm font-semibold text-foreground">
        <Filter className="w-4 h-4 text-blue-500" />
        <span>Filtros de Búsqueda Facetada de Buques</span>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Búsqueda por Nombre Parcial */}
        <div className="space-y-1.5">
          <Label htmlFor="q" className="text-xs font-medium">Nombre de la Embarcación</Label>
          <Input
            id="q"
            placeholder="Ej. EVER GIVEN o parcial"
            {...register("q")}
          />
        </div>

        {/* MMSI */}
        <div className="space-y-1.5">
          <Label htmlFor="mmsi" className="text-xs font-medium">MMSI (9 dígitos)</Label>
          <Input
            id="mmsi"
            type="number"
            placeholder="Ej. 244670000"
            {...register("mmsi")}
          />
        </div>

        {/* IMO */}
        <div className="space-y-1.5">
          <Label htmlFor="imo" className="text-xs font-medium">IMO (7 dígitos)</Label>
          <Input
            id="imo"
            type="number"
            placeholder="Ej. 9241061"
            {...register("imo")}
          />
        </div>

        {/* Tipo de Buque */}
        <div className="space-y-1.5">
          <Label htmlFor="shipType" className="text-xs font-medium">Tipo de Buque</Label>
          <select
            id="shipType"
            {...register("shipType")}
            className="flex h-8 w-full rounded-md border border-input bg-transparent px-2.5 py-1 text-xs shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring dark:bg-input/30"
          >
            {SHIP_TYPES.map((t) => (
              <option key={t.value} value={t.value} className="bg-background text-foreground">
                {t.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="flex items-center justify-end gap-2 pt-2 border-t">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleReset}
          disabled={isLoading}
        >
          <RotateCcw className="w-3.5 h-3.5 mr-1" />
          Limpiar
        </Button>
        <Button type="submit" size="sm" disabled={isLoading}>
          <Search className="w-3.5 h-3.5 mr-1" />
          {isLoading ? "Buscando..." : "Buscar Buques"}
        </Button>
      </div>
    </form>
  );
}
