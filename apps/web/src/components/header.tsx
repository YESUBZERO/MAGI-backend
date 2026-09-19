"use client";

import Link from "next/link";
import { signIn, signOut, useSession } from "next-auth/react";
import { ModeToggle } from "./mode-toggle";
import { Button } from "./ui/button";

export default function Header() {
  const { data: session, status } = useSession();

  return (
    <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-14 items-center justify-between px-4">
        <div className="flex items-center gap-6">
          <Link href="/" className="flex items-center gap-2 font-bold tracking-tight text-lg">
            <span className="text-blue-600 dark:text-blue-400">AIRES</span>
            <span>MAGI</span>
          </Link>
          <nav className="flex items-center gap-4 text-sm font-medium">
            <Link href="/" className="text-foreground transition-colors hover:text-foreground/80">
              Dashboard AIS
            </Link>
            <a
              href="http://localhost:8080/health"
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
            <span className="text-xs text-muted-foreground">Cargando...</span>
          ) : session ? (
            <div className="flex items-center gap-3">
              <span className="text-xs text-muted-foreground">
                {session.user?.name || session.user?.email || "Usuario Zitadel"}
              </span>
              <Button variant="outline" size="sm" onClick={() => signOut()}>
                Cerrar sesión
              </Button>
            </div>
          ) : (
            <Button variant="default" size="sm" onClick={() => signIn("zitadel")}>
              Iniciar sesión (Zitadel)
            </Button>
          )}
          <ModeToggle />
        </div>
      </div>
    </header>
  );
}
