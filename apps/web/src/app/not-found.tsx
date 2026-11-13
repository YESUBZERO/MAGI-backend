"use client";

import * as React from "react";
import Link from "next/link";
import { Compass, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import * as k from "@/components/figures/iso/iso-kit";

const WIDTH = 640;
const HEIGHT = 400;

// Geometría calibrada para presentación minimalista
const BEDPLATE: k.Plan = { x: -130, y: -70, w: 260, d: 140, r: 8 };
const BED_H = 10;
const SOUNDER_XY: [number, number] = [-65, 8];
const DIAL_R = 22;
const RADAR_XY: [number, number] = [65, 0];
const SCANNER_SPAN = 46;

const PROJECTION = k.fitProjection(
  [
    ...k.boxCorners(BEDPLATE, -BED_H - 12, 0),
    [SOUNDER_XY[0] - DIAL_R, SOUNDER_XY[1] - DIAL_R, 30],
    [SOUNDER_XY[0] + DIAL_R, SOUNDER_XY[1] + DIAL_R, 30],
    [RADAR_XY[0] - SCANNER_SPAN, RADAR_XY[1] - SCANNER_SPAN, 50],
    [RADAR_XY[0] + SCANNER_SPAN, RADAR_XY[1] + SCANNER_SPAN, 50],
  ],
  WIDTH,
  HEIGHT,
  { pad: 28, azimuth: 45, elevation: 30 }
);

const at = (point: k.Point3): k.Flat => k.iso(point, PROJECTION);

function buildStaticRig(): string {
  const parts: string[] = [];

  // Halo suave bajo el banco
  parts.push(`<path class="iso-halo" d="${k.haloOf(BEDPLATE, -BED_H - 12, BED_H, PROJECTION)}"/>`);

  // Bancada de precisión (sólido extruido con sombreado de 4 tonos)
  parts.push(k.solidSvg(k.slabOf(BEDPLATE, -BED_H, BED_H, PROJECTION, 6, 1.2), { tone: "lo" }));
  parts.push(k.lineSvg(k.planOutline(k.insetPlan(BEDPLATE, 4), 0, PROJECTION), { tone: "faint" }));

  // Regla tenue frontal con marcas
  const ruleStrip: k.Plan = { x: -105, y: BEDPLATE.y + BEDPLATE.d - 10, w: 210, d: 6, r: 1 };
  parts.push(k.solidSvg(k.slabOf(ruleStrip, 0, 0.6, PROJECTION, 2), { tone: "lo", crease: "none" }));
  const ticks = k.topTicks(ruleStrip.x + 4, ruleStrip.x + ruleStrip.w - 4, 3, 5, ruleStrip.y + 1, 0.6, [1.5, 3], PROJECTION, "x");
  parts.push(k.lineSvg(ticks.minor, { tone: "faint" }));
  parts.push(k.lineSvg(ticks.major, { tone: "mid" }));

  // Cuadrante azimutal de la sonda náutica (izquierda)
  parts.push(k.solidSvg(k.cylinder(SOUNDER_XY[0], SOUNDER_XY[1], DIAL_R + 3, 0, 16, PROJECTION, 32, 0.5), { tone: "mid" }));
  parts.push(k.solidSvg(k.cylinder(SOUNDER_XY[0], SOUNDER_XY[1], DIAL_R, 16, 1, PROJECTION, 32), { tone: "hi", crease: "none" }));
  const compassTicks = k.radialTicks(SOUNDER_XY[0], SOUNDER_XY[1], DIAL_R - 1, 32, 4, 17, [1.2, 2.8], PROJECTION);
  parts.push(k.lineSvg(compassTicks.minor, { tone: "faint" }));
  parts.push(k.lineSvg(compassTicks.major, { tone: "mid" }));

  // Pedestal del radar (derecha)
  parts.push(k.solidSvg(k.cylinder(RADAR_XY[0], RADAR_XY[1], 22, 0, 24, PROJECTION, 28, 0.6), { tone: "mid" }));
  parts.push(k.solidSvg(k.cylinder(RADAR_XY[0], RADAR_XY[1], 12, 24, 2, PROJECTION, 20), { tone: "hi" }));

  // Conduit sutil entre módulos
  const cFrom = at([SOUNDER_XY[0] + DIAL_R, SOUNDER_XY[1], 1]);
  const cMid = at([0, -6, 0.5]);
  const cTo = at([RADAR_XY[0] - 18, RADAR_XY[1], 1]);
  parts.push(k.wireSvg(cFrom, cTo, { via: [cMid], tone: "lo", flow: false }));

  return parts.join("");
}

function buildDynamicRig(radarAngleDeg: number, needleJitterDeg: number): string {
  const parts: string[] = [];
  const rad = (radarAngleDeg * Math.PI) / 180;

  // Aguja azimutal acústica
  const needleAngle = rad * 1.5 + (needleJitterDeg * Math.PI) / 180;
  const tipX = SOUNDER_XY[0] + Math.sin(needleAngle) * (DIAL_R - 3);
  const tipY = SOUNDER_XY[1] - Math.cos(needleAngle) * (DIAL_R - 3);
  const tailX = SOUNDER_XY[0] - Math.sin(needleAngle) * 4;
  const tailY = SOUNDER_XY[1] + Math.cos(needleAngle) * 4;
  const needleZ = 18.5;

  const pTip = at([tipX, tipY, needleZ]);
  const pTail = at([tailX, tailY, needleZ]);
  parts.push(k.lineSvg(k.pathOf([pTail, pTip]), { tone: "lit" }));
  parts.push(k.dotsSvg([pTip], { size: 0.6, tone: "hi" }));

  // Barra de radar náutico oscilante
  const cosR = Math.cos(rad);
  const sinR = Math.sin(rad);
  const spanX = -sinR * SCANNER_SPAN;
  const spanY = cosR * SCANNER_SPAN;
  const pivotZ = 34;

  const cA: k.Point3 = [RADAR_XY[0] - spanX, RADAR_XY[1] - spanY, pivotZ - 2];
  const cB: k.Point3 = [RADAR_XY[0] + spanX, RADAR_XY[1] + spanY, pivotZ - 2];
  const cC: k.Point3 = [RADAR_XY[0] + spanX, RADAR_XY[1] + spanY, pivotZ + 2];
  const cD: k.Point3 = [RADAR_XY[0] - spanX, RADAR_XY[1] - spanY, pivotZ + 2];

  const flatCA = at(cA), flatCB = at(cB), flatCC = at(cC), flatCD = at(cD);
  const facePath = k.pathOf([flatCA, flatCB, flatCC, flatCD], true);

  parts.push(`<polygon class="iso-fill" d="${facePath}"/>`);
  parts.push(`<path class="iso-line" data-tone="hi" d="${facePath}"/>`);

  // Haz explorador minimalista
  const hornCenter = at([RADAR_XY[0], RADAR_XY[1], pivotZ]);
  const bLeft = at([RADAR_XY[0] + Math.cos(rad - 0.2) * 55, RADAR_XY[1] + Math.sin(rad - 0.2) * 55, pivotZ]);
  const bRight = at([RADAR_XY[0] + Math.cos(rad + 0.2) * 55, RADAR_XY[1] + Math.sin(rad + 0.2) * 55, pivotZ]);
  const bMid = at([RADAR_XY[0] + Math.cos(rad) * 65, RADAR_XY[1] + Math.sin(rad) * 65, pivotZ]);

  parts.push(k.lineSvg(k.pathOf([hornCenter, bLeft]), { tone: "faint", dashed: true }));
  parts.push(k.lineSvg(k.pathOf([hornCenter, bRight]), { tone: "faint", dashed: true }));
  parts.push(k.lineSvg(k.pathOf([bLeft, bMid, bRight]), { tone: "lit", dotted: true }));
  parts.push(k.dotsSvg([bMid], { size: 1.1, tone: "hi", pulse: true }));

  return parts.join("");
}

export default function NotFound() {
  const staticSvg = React.useMemo(() => buildStaticRig(), []);
  const [readout, setReadout] = React.useState("AZ 045.0° · SIN ECO");
  const [dynamicSvg, setDynamicSvg] = React.useState(() => buildDynamicRig(45, 0));

  const targetAngleRef = React.useRef(45);
  const currentAngleRef = React.useRef(45);
  const isHoveredRef = React.useRef(false);
  const stageRef = React.useRef<HTMLDivElement>(null);
  const lastTimeRef = React.useRef<number>(0);
  const animFrameRef = React.useRef<number>(0);

  const handlePointerMove = React.useCallback((e: React.PointerEvent) => {
    isHoveredRef.current = true;
    const stage = stageRef.current;
    if (!stage) return;
    const rect = stage.getBoundingClientRect();
    const nx = (e.clientX - rect.left) / rect.width;
    targetAngleRef.current = (nx * 360) % 360;
  }, []);

  const handlePointerLeave = React.useCallback(() => {
    isHoveredRef.current = false;
  }, []);

  const handleRecalibrate = React.useCallback(() => {
    targetAngleRef.current = 0;
  }, []);

  React.useEffect(() => {
    const loop = (now: number) => {
      if (!lastTimeRef.current) lastTimeRef.current = now;
      const dt = Math.min((now - lastTimeRef.current) / 1000, 0.05);
      lastTimeRef.current = now;

      if (!isHoveredRef.current) {
        targetAngleRef.current = (targetAngleRef.current + dt * 22) % 360;
      }

      const diff = targetAngleRef.current - currentAngleRef.current;
      const delta = ((diff + 540) % 360) - 180;
      currentAngleRef.current += delta * (1 - Math.exp(-dt * 5));
      currentAngleRef.current = (currentAngleRef.current + 360) % 360;

      const jitter = Math.sin(now * 0.01) * 3;
      const currentDeg = currentAngleRef.current;
      setDynamicSvg(buildDynamicRig(currentDeg, jitter));

      const azStr = currentDeg.toFixed(1).padStart(5, "0");
      setReadout(`AZ ${azStr}° · SIN ECO`);

      animFrameRef.current = requestAnimationFrame(loop);
    };

    animFrameRef.current = requestAnimationFrame(loop);
    return () => cancelAnimationFrame(animFrameRef.current);
  }, []);

  return (
    <div className="iso relative flex flex-col items-center justify-center min-h-[calc(100vh-4rem)] p-4 sm:p-6 bg-slate-50/60 dark:bg-background text-foreground" data-theme="light">
      <style dangerouslySetInnerHTML={{ __html: k.ISO_CSS }} />

      <div className="w-full max-w-xl flex flex-col items-center text-center space-y-6">
        {/* Minimal Header (shadcn style) */}
        <div className="space-y-2 flex flex-col items-center">
          <div className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-mono font-medium border border-border bg-card text-muted-foreground shadow-xs">
            <span className="size-1.5 rounded-full bg-rose-500" />
            ERROR 404 // COORDENADAS FUERA DE CARTA
          </div>
          <h1 className="text-2xl sm:text-3xl font-semibold tracking-tight text-foreground">
            Página no encontrada
          </h1>
          <p className="text-sm text-muted-foreground max-w-md leading-normal">
            El buque o la ruta que buscas no existe en la carta náutica ni en el registro activo de telemetría.
          </p>
        </div>

        {/* Anatomy Minimal Card */}
        <div
          ref={stageRef}
          className="iso-plate relative w-full select-none cursor-crosshair overflow-hidden border border-border/80 bg-card rounded-xl shadow-xs"
          onPointerMove={handlePointerMove}
          onPointerLeave={handlePointerLeave}
        >
          <div className="iso-plate-corners pointer-events-none flex justify-between p-3 text-[11px] font-mono text-muted-foreground z-10" aria-hidden="true">
            <span className="font-semibold text-foreground/80">FIG. 404</span>
            <span className="text-sky-600 dark:text-sky-400 font-medium">{readout}</span>
          </div>

          <svg className="iso-svg w-full h-auto block" viewBox={`0 0 ${WIDTH} ${HEIGHT}`} role="img" aria-label="Radar náutico isométrico minimalista">
            <g className="iso-static-assembly" dangerouslySetInnerHTML={{ __html: staticSvg }} />
            <g className="iso-dynamic-assembly" dangerouslySetInnerHTML={{ __html: dynamicSvg }} />
          </svg>
        </div>

        {/* Minimal Navigation Actions (shadcn Buttons) */}
        <div className="flex items-center justify-center gap-3 pt-1">
          <Button
            variant="outline"
            size="default"
            className="cursor-pointer gap-2 bg-card hover:bg-muted text-foreground"
            onClick={handleRecalibrate}
          >
            <RotateCcw className="size-3.5" />
            Recalibrar radar
          </Button>

          <Link href="/">
            <Button
              variant="default"
              size="default"
              className="cursor-pointer gap-2 font-medium"
            >
              <Compass className="size-3.5" />
              Volver al Centro de Mando
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
