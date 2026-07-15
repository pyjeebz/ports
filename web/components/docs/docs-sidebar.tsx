"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

import { PortsLogo } from "@/components/site/ports-logo";

const GUIDES = [
  { href: "/docs", label: "Introduction" },
  { href: "/docs/installation", label: "Installation" },
  { href: "/docs/quickstart", label: "Quickstart" },
];
const REFERENCE = [{ href: "/docs/commands", label: "Commands" }];

function Group({
  title,
  items,
  path,
}: {
  title: string;
  items: { href: string; label: string }[];
  path: string;
}) {
  const [open, setOpen] = useState(true);
  return (
    <div className={`sb-group${open ? "" : " closed"}`}>
      <button className="sb-head" onClick={() => setOpen((o) => !o)}>
        {title} <span className="chev">▾</span>
      </button>
      <div className="sb-items">
        {items.map((it) => (
          <Link
            key={it.href}
            href={it.href}
            className={`nav-item${path === it.href ? " active" : ""}`}
          >
            {it.label}
          </Link>
        ))}
      </div>
    </div>
  );
}

export function DocsSidebar() {
  const path = usePathname();
  return (
    <aside className="sidebar">
      <Link className="brand" href="/">
        <PortsLogo className="h-[22px] w-[22px]" maskId="ports-slash-docs" />
        ports <span className="docs-tag">docs</span>
      </Link>
      <div className="search">
        <svg
          width="14"
          height="14"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
        >
          <circle cx="9" cy="9" r="6" />
          <path d="M13.5 13.5L17 17" />
        </svg>
        Search docs…
        <span className="kbd">⌘K</span>
      </div>
      <div className="sb-nav">
        <Group title="Guides" items={GUIDES} path={path} />
        <Group title="Reference" items={REFERENCE} path={path} />
      </div>
      <div className="sb-project">
        <h5>Project</h5>
        <a href="https://github.com/pyjeebz/ports">
          GitHub <span>↗</span>
        </a>
        <a href="https://github.com/pyjeebz/ports/releases">
          Releases <span>↗</span>
        </a>
        <Link href="/">
          ports.tools <span>↗</span>
        </Link>
      </div>
    </aside>
  );
}
