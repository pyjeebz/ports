"use client";

import Link from "next/link";
import { useEffect, useState, useSyncExternalStore } from "react";
import { useTheme } from "next-themes";
import { SITE } from "@/lib/site";
import { GitHubIcon, MoonIcon, PortsMark, SunIcon } from "./icons";

/**
 * The bar across the top: a translucent layer the page scrolls under, so
 * it orients without taking a strip of the page for itself.
 */
export function Nav() {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/75 backdrop-blur-md backdrop-saturate-150">
      <div className="mx-auto flex h-14 max-w-[1040px] items-center justify-between px-4 sm:px-6">
        <Link href="/" className="flex items-center gap-2 font-semibold tracking-tight">
          <PortsMark className="size-5" />
          ports
        </Link>
        <nav className="flex items-center gap-1 text-sm text-muted-foreground">
          <Link href={SITE.links.docs} className="rounded-md px-2.5 py-1.5 transition-colors hover:text-foreground">
            Docs
          </Link>
          <a
            href={SITE.links.releases}
            className="hidden rounded-md px-2.5 py-1.5 transition-colors hover:text-foreground sm:inline"
          >
            Releases
          </a>
          <a
            href={SITE.links.github}
            aria-label="GitHub"
            className="grid size-8 place-items-center rounded-md transition-colors hover:text-foreground"
          >
            <GitHubIcon className="size-4" />
          </a>
          <ThemeToggle />
        </nav>
      </div>
    </header>
  );
}

function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();
  // the theme is known only in the browser: render the server's guess,
  // then the real one, without a mismatch
  const mounted = useSyncExternalStore(
    () => () => {},
    () => true,
    () => false,
  );
  const dark = mounted && resolvedTheme === "dark";
  return (
    <button
      type="button"
      aria-label={dark ? "Light theme" : "Dark theme"}
      onClick={() => setTheme(dark ? "light" : "dark")}
      className="grid size-8 place-items-center rounded-md transition-[transform,color] duration-150 ease-out hover:text-foreground active:scale-[0.96]"
    >
      {dark ? <SunIcon className="size-4" /> : <MoonIcon className="size-4" />}
    </button>
  );
}

export type TocItem = { id: string; title: string };

/**
 * "On this page", the way a doc has it: every section, with the one you're
 * reading marked. The mark moves with you, on scroll, not on a timer.
 */
export function Toc({ items }: { items: TocItem[] }) {
  const [active, setActive] = useState<string | undefined>(items[0]?.id);
  useEffect(() => {
    const io = new IntersectionObserver(
      (entries) => {
        const seen = entries.filter((e) => e.isIntersecting).map((e) => e.target.id);
        if (seen.length) setActive(items.find((i) => seen.includes(i.id))?.id);
      },
      { rootMargin: "-15% 0px -70% 0px" },
    );
    for (const i of items) {
      const el = document.getElementById(i.id);
      if (el) io.observe(el);
    }
    return () => io.disconnect();
  }, [items]);

  return (
    <nav aria-label="On this page" className="sticky top-24 text-sm">
      <p className="mb-3 text-xs font-medium text-foreground">On this page</p>
      <ul className="grid gap-0.5 border-l">
        {items.map((i) => (
          <li key={i.id}>
            <a
              href={`#${i.id}`}
              className={`-ml-px block border-l py-1 pl-3 transition-colors duration-150 ${active === i.id ? "border-foreground text-foreground" : "border-transparent text-muted-foreground hover:text-foreground"}`}
            >
              {i.title}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}

export function Footer() {
  return (
    <footer className="border-t">
      <div className="mx-auto flex max-w-[1040px] flex-col gap-3 px-4 py-8 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-6">
        <span className="flex items-center gap-2">
          <PortsMark className="size-4" />
          ports · MIT
        </span>
        <span className="flex gap-4">
          <Link href={SITE.links.docs} className="hover:text-foreground">
            Docs
          </Link>
          <a href={SITE.links.github} className="hover:text-foreground">
            GitHub
          </a>
          <a href={SITE.links.issues} className="hover:text-foreground">
            Issues
          </a>
        </span>
      </div>
    </footer>
  );
}
