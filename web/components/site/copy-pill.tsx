"use client";

import { useState } from "react";

import { SITE } from "@/lib/site";

/** A command line in a copy-to-clipboard pill — defaults to the install curl. */
export function CopyPill({
  text = SITE.installCurl,
  ariaLabel = "Copy the install command",
}: {
  text?: string;
  ariaLabel?: string;
}) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {
      /* clipboard unavailable */
    }
  }

  return (
    <button
      type="button"
      onClick={copy}
      aria-label={ariaLabel}
      className="flex h-9 cursor-pointer items-center gap-2.5 rounded-md border border-line bg-panel px-3.5 font-mono text-[12.5px] text-muted transition-colors hover:border-strong/40 hover:text-foreground"
    >
      <span className="text-foreground">{text}</span>
      <span className={copied ? "text-accent" : "text-faint"}>
        {copied ? "copied" : "copy"}
      </span>
    </button>
  );
}