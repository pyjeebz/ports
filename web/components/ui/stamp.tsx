"use client";

import * as React from "react";
import { DrawablyBadge } from "drawably/react";

import { cn } from "@/lib/utils";

/**
 * The rubber stamp: mono uppercase ink inside a scribble frame — for
 * declarations like NO DAEMON or ZERO CONFIG. `ink` picks the pad color;
 * stamps are always tilted and never subtle.
 */
const INKS = {
  red: { text: "text-danger", fill: "var(--danger)" },
  blue: { text: "text-accent", fill: "var(--accent)" },
  charcoal: { text: "text-charcoal", fill: "var(--charcoal)" },
} as const;

const ROTATIONS = {
  "-3": "-rotate-3",
  "-2": "-rotate-2",
  "-1": "-rotate-1",
  "0": "rotate-0",
  "1": "rotate-1",
  "2": "rotate-2",
  "3": "rotate-3",
} as const;

type StampProps = React.ComponentProps<typeof DrawablyBadge> & {
  ink?: keyof typeof INKS;
  rotate?: keyof typeof ROTATIONS;
};

function Stamp({
  className,
  ink = "red",
  rotate = "-2",
  ...props
}: StampProps) {
  return (
    <DrawablyBadge
      data-slot="stamp"
      variant="scribble"
      fill={INKS[ink].fill}
      className={cn(
        "inline-flex items-center px-3 py-1 font-mono text-[12px] font-bold tracking-[0.14em] uppercase",
        INKS[ink].text,
        ROTATIONS[rotate],
        className,
      )}
      {...props}
    />
  );
}

export { Stamp };