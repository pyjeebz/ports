"use client";

import { useState } from "react";

const CMD = "curl -fsSL https://ports.tools/install | sh";

export function CopyPill() {
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
    <div className="flex h-10 items-center gap-3.5 rounded-md border border-line bg-panel pl-[18px] pr-2.5 font-mono text-[13.5px]">
      <span className="select-none text-accent">$</span>
      <span>{CMD}</span>
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
