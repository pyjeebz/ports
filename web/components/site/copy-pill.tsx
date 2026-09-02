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
      className="flex h-10 cursor-pointer items-center gap-2.5 border-[1.5px] border-charcoal bg-white px-4 font-mono text-[12.5px] text-charcoal-light transition-colors hover:bg-highlighter-yellow/50 hover:text-charcoal"
    >
      <span className="text-charcoal">{text}</span>
      <span className={copied ? "text-charcoal font-bold" : "text-faint"}>
        {copied ? "copied" : "copy"}
      </span>
    </button>
  );
}