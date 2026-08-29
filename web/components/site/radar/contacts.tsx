"use client";

import type { Contact } from "@/lib/site";

/**
 * The visible contact legend — the real UI for the radar. The canvas is
 * decorative (aria-hidden); every fact on screen lives here as real text,
 * reachable by keyboard, and any change announces itself via :focus-visible.
 */
export function ContactsLegend({
  contacts,
  selected,
  onSelect,
}: {
  contacts: readonly Contact[];
  selected: number | null;
  onSelect: (port: number | null) => void;
}) {
  return (
    <ul className="grid w-full grid-cols-2 gap-px overflow-hidden rounded-md border border-line-soft bg-line-soft sm:grid-cols-3">
      {contacts.map((ct) => {
        const active = selected === ct.port;
        return (
          <li key={ct.port} className="min-w-0">
            <button
              type="button"
              onMouseEnter={() => onSelect(ct.port)}
              onFocus={() => onSelect(ct.port)}
              onMouseLeave={() => onSelect(null)}
              onBlur={() => onSelect(null)}
              onClick={() => onSelect(active ? null : ct.port)}
              aria-pressed={active}
              className={`block w-full cursor-pointer px-3.5 py-3 text-left font-mono text-xs transition-colors ${
                active
                  ? "bg-accent-dim text-accent-bright"
                  : "bg-panel text-muted hover:bg-panel-2 hover:text-foreground"
              }`}
            >
              <span className="block font-medium text-foreground">{ct.framework}</span>
              <span className="block tabular-nums">
                {ct.port} · {ct.project}
              </span>
              <span className="block text-faint">{ct.uptime} up</span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}