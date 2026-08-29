"use client";

import { useState } from "react";

import { INSTALL_METHODS, SITE } from "@/lib/site";

export function InstallSection() {
  const [activeId, setActiveId] = useState(INSTALL_METHODS[0].id);
  const [copied, setCopied] = useState(false);
  const method = INSTALL_METHODS.find((m) => m.id === activeId) ?? INSTALL_METHODS[0];

  async function copy() {
    try {
      await navigator.clipboard.writeText(method.lines.join("\n"));
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {
      /* clipboard unavailable */
    }
  }

  return (
    <section id="install" className="border-b border-line-soft py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <p className="eyebrow">▸ install</p>
        <h2 className="mt-4 text-balance text-[clamp(1.5rem,2.6vw,2rem)] font-semibold leading-[1.2] tracking-[-0.03em] text-strong">
          One static binary, no runtime.
        </h2>
        <p className="mt-3 max-w-[52ch] text-[15px] leading-relaxed text-muted">
          Useful sixty seconds after install, on a machine in whatever state
          it is in.
        </p>

        <div className="mt-8 max-w-[560px]">
          <div className="flex" role="tablist" aria-label="Install method">
            {INSTALL_METHODS.map((m) => (
              <button
                key={m.id}
                type="button"
                role="tab"
                aria-selected={m.id === activeId}
                onClick={() => setActiveId(m.id)}
                className={`-mb-px cursor-pointer border-b-2 px-3.5 py-2 font-mono text-[12.5px] transition-colors ${
                  m.id === activeId
                    ? "border-accent text-accent"
                    : "border-transparent text-muted hover:text-foreground"
                }`}
              >
                {m.label}
              </button>
            ))}
          </div>
          <div className="code-shell mt-3.5" role="tabpanel">
            <div className="flex items-center justify-between border-b border-line-soft bg-panel-2 px-3.5 py-2">
              <span className="readout">{method.os}</span>
              <button
                type="button"
                onClick={copy}
                className="cursor-pointer rounded-[5px] border border-line px-2 py-0.5 text-[11px] font-mono text-muted transition-colors hover:text-foreground"
              >
                {copied ? "copied" : "copy"}
              </button>
            </div>
            <pre className="text-muted">
              {method.lines.map((line) => (
                <span key={line} className="block">
                  <span className="mr-2 select-none text-faint">$</span>
                  {line}
                </span>
              ))}
            </pre>
          </div>
          <p className="mt-4 text-[12px] leading-relaxed text-faint">
            No daemons, no shell profiles, no autoloaders. Or read the{" "}
            <a href={SITE.links.docs} className="text-muted underline-offset-4 hover:text-foreground hover:underline">
              installation guide
            </a>{" "}
            for the details.
          </p>
        </div>
      </div>
    </section>
  );
}