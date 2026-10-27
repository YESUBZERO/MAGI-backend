"use client";

import Link from "next/link";
import { signIn, signOut, useSession } from "next-auth/react";
import { ModeToggle } from "./mode-toggle";
import { Button } from "./ui/button";
import { Shield, User, LogOut, LogIn, Activity } from "lucide-react";

export default function Header() {
  const { data: session, status } = useSession();

  const userRoles = session?.roles || session?.user?.roles || [];
  const isAdmin = userRoles.includes("admin");
  const isOperator = userRoles.includes("operator");
  const healthUrl = process.env.NEXT_PUBLIC_API_URL
    ? `${process.env.NEXT_PUBLIC_API_URL}/health`
    : "http://localhost:8080/health";

  return (
    <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 sticky top-0 z-40">
      <div className="container mx-auto flex h-14 items-center justify-between px-4">
        <div className="flex items-center gap-6">
          <Link href="/" className="flex items-center gap-2 font-bold tracking-tight text-lg">
            <span className="text-blue-600 dark:text-blue-400">AIRES</span>
            <span>MAGI</span>
          </Link>
          <nav className="flex items-center gap-4 text-xs font-medium">
            <Link href="/" className="text-foreground transition-colors hover:text-foreground/80 flex items-center gap-1.5">
              <Activity className="w-3.5 h-3.5 text-blue-500" />
              Búsqueda & Flota
            </Link>
            <a
              href={process.env.NEXT_PUBLIC_API_URL ? `${process.env.NEXT_PUBLIC_API_URL}/health` : "http://localhost:8080/health"}
              target="_blank"
              rel="noreferrer"
              className="text-muted-foreground transition-colors hover:text-foreground"
            >
              API Status
            </a>
          </nav>
        </div>

        <div className="flex items-center gap-3">
          {status === "loading" ? (
            <span className="text-xs text-muted-foreground animate-pulse">Cargando sesión...</span>
          ) : session ? (
            <div className="flex items-center gap-3">
              <div className="flex flex-col items-end text-right">
                <span className="text-xs font-medium text-foreground flex items-center gap-1">
                  <User className="w-3 h-3 text-muted-foreground" />
                  {session.user?.name || session.user?.email || "Usuario Zitadel"}
                </span>
                <div className="flex items-center gap-1 mt-0.5">
                  {isAdmin && (
                    <span className="inline-flex items-center gap-0.5 text-[9px] font-semibold bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300 px-1.5 py-0.2 rounded">
                      <Shield className="w-2.5 h-2.5" /> Admin
                    </span>
                  )}
                  {isOperator && !isAdmin && (
                    <span className="inline-flex items-center gap-0.5 text-[9px] font-semibold bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300 px-1.5 py-0.2 rounded">
                      Operador
                    </span>
                  )}
                  {!isAdmin && !isOperator && (
                    <span className="text-[10px] text-muted-foreground">Lector</span>
                  )}
                </div>
              </div>

              <Button variant="outline" size="xs" onClick={() => signOut()} className="gap-1">
                <LogOut className="w-3 h-3" />
                Salir
              </Button>
            </div>
          ) : (
            <Button variant="default" size="xs" onClick={() => signIn("zitadel")} className="gap-1">
              <LogIn className="w-3 h-3" />
              Iniciar sesión (Zitadel)
            </Button>
          )}
          <ModeToggle />
        </div>
      </div>
    </header>
  );
}
