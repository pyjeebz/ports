"use client";

import * as React from "react";
import { DrawablyHighlight } from "drawably/react";

import { cn } from "@/lib/utils";

/**
 * Marker emphasis inside running text — the Napkin replacement for bold.
 * The wash color comes from the `fill` drawably option (the scribble's
 * stroke); keep spans short, one emphasis per thought.
 */
const FILLS = {
  yellow: "var(--highlighter-yellow)",
  pink: "var(--highlighter-pink)",
} as const;

type HighlighterProps = React.ComponentProps<typeof DrawablyHighlight> & {
  color?: keyof typeof FILLS;
};

function Highlighter({ className, color = "yellow", ...props }: HighlighterProps) {
  return (
    <DrawablyHighlight
      data-slot="highlighter"
      fill={FILLS[color]}
      className={cn("text-charcoal", className)}
      {...props}
    />
  );
}

export { Highlighter };