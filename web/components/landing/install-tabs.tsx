"use client";

import { useLayoutEffect, useRef, useState } from "react";
import { INSTALL_METHODS } from "@/lib/site";
import { CopyButton } from "./copy-button";

/**
 * The install command, one tab per channel. The underline slides to the
 * chosen tab — a spatial hint of where the content came from — on a
 * 200ms ease-out; the command itself swaps without motion, since it's
 * what the reader came to copy.
 */
export function InstallTabs() {
  const [active, setActive] = useState(0);
  const tabs = useRef<(HTMLButtonElement | null)[]>([]);
  const [bar, setBar] = useState({ x: 0, w: 0 });

  useLayoutEffect(() => {
    const el = tabs.current[active];
    if (el) setBar({ x: el.offsetLeft, w: el.offsetWidth });
  }, [active]);

  const method = INSTALL_METHODS[active];
  return (
    <div className="my-5 overflow-hidden rounded-lg border bg-card">
      <div className="flex items-center justify-between border-b bg-muted/40 pr-1">
        <div role="tablist" aria-label="Install with" className="relative flex px-1.5">
          {INSTALL_METHODS.map((m, i) => (
            <button
              key={m.id}
              ref={(el) => {
                tabs.current[i] = el;
              }}
              role="tab"
              aria-selected={i === active}
              onClick={() => setActive(i)}
              className={`h-9 px-2.5 text-xs transition-colors duration-150 ${i === active ? "text-foreground" : "text-muted-foreground hover:text-foreground"}`}
            >
              {m.label}
            </button>
          ))}
          <span
            aria-hidden
            className="absolute bottom-0 left-0 h-px bg-foreground transition-[transform,width] duration-200 ease-out motion-reduce:transition-none"
            style={{ width: bar.w, transform: `translateX(${bar.x}px)` }}
          />
        </div>
        <CopyButton text={method.lines.join("\n")} />
      </div>
      <pre role="tabpanel" className="overflow-x-auto py-3 font-mono text-[13px] leading-6">
        <code className="grid min-w-max">
          {method.lines.map((l) => (
            <span key={l} className="px-4">
              <span className="text-muted-foreground select-none">$ </span>
              {l}
            </span>
          ))}
        </code>
      </pre>
    </div>
  );
}
