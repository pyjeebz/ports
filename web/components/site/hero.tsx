"use client";

import Link from "next/link";
import { useState } from "react";

import { CONTACTS, SITE } from "@/lib/site";

import { RadarBoard } from "./radar/radar-board";
import { ContactsLegend } from "./radar/contacts";

export function Hero() {
  const [selected, setSelected] = useState<number | null>(null);
  const ct = CONTACTS.find((c) => c.port === selected) ?? null;

  return (
    <section className="relative overflow-hidden border-b border-line-soft">
      {/* static plotting grid behind the radar panel only */}
      <div className="plot absolute inset-0 opacity-70" aria-hidden="true" />
      <div
        className="relative mx-auto grid max-w-[1160px] items-center gap-14 px-8 pb-20 pt-20 lg:grid-cols-[1.05fr_0.95fr] lg:pb-24 lg:pt-24"
      >
        <div>
          <p className="eyebrow">
            ▸ radar · localhost
            <span className="ml-3 text-faint">CONTACTS {CONTACTS.length}</span>
            <span className="ml-3 text-faint">SWEEP 6.0S</span>
          </p>
          <h1 className="mt-5 max-w-[16ch] text-balance font-sans text-[clamp(2.4rem,5.2vw,4rem)] font-semibold leading-[1.04] tracking-[-0.035em] text-strong">
            See what&apos;s running on your machine, by project.
          </h1>
          <p className="mt-6 max-w-[52ch] text-[16px] leading-relaxed text-muted">
            ports is a zero-config CLI that shows every listening port with its
            framework, project, and uptime — and lets you stop things by name.
            No PID hunting, no <code className="font-mono text-[0.85em] text-foreground">lsof</code> incantations.
          </p>
          <div className="mt-9 flex flex-wrap items-center gap-4">
            <Link
              href="/docs/installation"
              className="rounded-md bg-accent px-5 py-2.5 text-[14px] font-medium text-[#04150b] transition-colors hover:bg-accent-bright"
            >
              Install ports
            </Link>
            <Link
              href="/docs"
              className="rounded-md border border-line px-5 py-2.5 font-mono text-[13px] text-muted transition-colors hover:border-faint hover:text-foreground"
            >
              Read the docs →
            </Link>
          </div>
          <CopyPill />
        </div>

        <div className="flex flex-col items-center gap-6">
          <RadarBoard contacts={CONTACTS} selected={selected} onSelect={setSelected} />
          <p className="readout h-[1.2em] tabular-nums" aria-live="polite">
            {ct ? (
              <span>
                <b>CONTACT {ct.port}</b> · {ct.framework} · {ct.project} · {ct.uptime}
              </span>
            ) : (
              <span>
                <b>CONTACTS {CONTACTS.length}</b> · hover a blip or the legend
              </span>
            )}
          </p>
          <ContactsLegend contacts={CONTACTS} selected={selected} onSelect={setSelected} />
        </div>
      </div>
    </section>
  );
}

const CMD = SITE.installCurl;

function CopyPill() {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(CMD);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {
      /* clipboard unavailable */
    }
  }

  return (
    <div className="mt-8 flex h-10 w-fit max-w-full items-center gap-3.5 rounded-md border border-line bg-panel pl-4 pr-2.5 font-mono text-[13px]">
      <span className="select-none text-accent">$</span>
      <span className="truncate">{CMD}</span>
      <button
        type="button"
        onClick={copy}
        className="shrink-0 cursor-pointer rounded-[5px] border border-line bg-panel-2 px-[9px] py-1 text-[11px] text-muted transition-colors hover:text-foreground"
      >
        {copied ? "copied" : "copy"}
      </button>
    </div>
  );
}