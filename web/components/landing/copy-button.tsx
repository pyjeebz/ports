"use client";

import { useEffect, useRef, useState } from "react";
import { CheckIcon, CopyIcon } from "./icons";

/**
 * Copies text, and says so: the icon cross-fades to a check through a
 * short blur — two icons swapping read as one shape changing — and the
 * button gives under the press. Pressed a few times a session at most,
 * so the motion is allowed; it stays under 200ms.
 */
export function CopyButton({ text, label = "Copy" }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => () => clearTimeout(timer.current), []);

  return (
    <button
      type="button"
      aria-label={copied ? "Copied" : label}
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
        } catch {
          return;
        }
        setCopied(true);
        clearTimeout(timer.current);
        timer.current = setTimeout(() => setCopied(false), 1400);
      }}
      className="relative grid size-7 shrink-0 place-items-center rounded-md text-muted-foreground transition-[transform,color,background-color] duration-150 ease-out hover:bg-accent hover:text-foreground active:scale-[0.96] focus-visible:outline-2 focus-visible:outline-ring"
    >
      <CopyIcon
        className={`col-start-1 row-start-1 size-3.5 transition-[opacity,filter,transform] duration-200 ease-out ${copied ? "scale-90 opacity-0 blur-[2px]" : ""}`}
      />
      <CheckIcon
        className={`col-start-1 row-start-1 size-3.5 text-added transition-[opacity,filter,transform] duration-200 ease-out ${copied ? "" : "scale-90 opacity-0 blur-[2px]"}`}
      />
    </button>
  );
}
