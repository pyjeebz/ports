"use client";

import { useEffect, useRef, useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

/**
 * Copies text, and says so twice: the icon cross-fades to a check through
 * a short blur — two icons swapping read as one shape changing — and the
 * tooltip changes its word. Both under 200ms.
 */
export function CopyButton({ text, label = "Copy" }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => () => clearTimeout(timer.current), []);

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          aria-label={copied ? "Copied" : label}
          className="relative size-8 text-muted-foreground transition-[transform,color,background-color] duration-150 ease-out hover:text-foreground active:scale-[0.96]"
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
        >
          <Copy
            className={`absolute size-3.5 transition-[opacity,filter,transform] duration-200 ease-out ${copied ? "scale-90 opacity-0 blur-[2px]" : ""}`}
          />
          <Check
            className={`absolute size-3.5 text-green transition-[opacity,filter,transform] duration-200 ease-out ${copied ? "" : "scale-90 opacity-0 blur-[2px]"}`}
          />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top">{copied ? "Copied" : label}</TooltipContent>
    </Tooltip>
  );
}
