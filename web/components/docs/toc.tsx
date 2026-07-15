"use client";

import { useEffect, useRef } from "react";

type Item = { href: string; label: string };

export function Toc({ items }: { items: Item[] }) {
  const ref = useRef<HTMLElement>(null);

  useEffect(() => {
    const root = ref.current;
    if (!root) return;
    const links = Array.from(root.querySelectorAll<HTMLAnchorElement>("a"));
    const heads = items
      .map((i) => document.getElementById(i.href.slice(1)))
      .filter((x): x is HTMLElement => !!x);
    if (!heads.length) return;
    const io = new IntersectionObserver(
      (es) => {
        es.forEach((e) => {
          if (e.isIntersecting)
            links.forEach((l) =>
              l.classList.toggle("lit", l.getAttribute("href") === "#" + e.target.id),
            );
        });
      },
      { rootMargin: "-15% 0px -70% 0px" },
    );
    heads.forEach((h) => io.observe(h));
    return () => io.disconnect();
  }, [items]);

  return (
    <aside className="toc" ref={ref}>
      <h6>
        <svg
          width="13"
          height="13"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
        >
          <path d="M4 5h12M4 10h12M4 15h8" />
        </svg>{" "}
        On this page
      </h6>
      {items.map((i) => (
        <a key={i.href} href={i.href}>
          {i.label}
        </a>
      ))}
    </aside>
  );
}
