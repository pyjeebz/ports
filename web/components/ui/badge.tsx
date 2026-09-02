"use client";

import * as React from "react";
import { DrawablyBadge } from "drawably/react";

import { cn } from "@/lib/utils";

/**
 * Napkin tags: mono metadata — framework names, container ids, statuses —
 * on a highlighter wash with a drawn border. Rotate slightly when
 * several badges sit together.
 */
const COLORS = {
  yellow: "bg-highlighter-yellow",
  pink: "bg-highlighter-pink",
  blue: "bg-highlighter-blue",
  green: "bg-highlighter-green",
  white: "bg-white",
  transparent: "bg-transparent",
} as const;

const ROTATIONS = {
  "-2": "-rotate-2",
  "-1": "-rotate-1",
  "0": "rotate-0",
  "1": "rotate-1",
  "2": "rotate-2",
} as const;

type BadgeProps = React.ComponentProps<typeof DrawablyBadge> & {
  color?: keyof typeof COLORS;
  rotate?: keyof typeof ROTATIONS;
};

function Badge({
  className,
  color = "yellow",
  rotate = "0",
  ...props
}: BadgeProps) {
  return (
    <DrawablyBadge
      data-slot="badge"
      variant="outline"
      className={cn(
        "inline-flex items-center px-2.5 py-0.5 font-mono text-[12px] leading-relaxed text-charcoal",
        COLORS[color],
        ROTATIONS[rotate],
        className,
      )}
      {...props}
    />
  );
}

export { Badge };