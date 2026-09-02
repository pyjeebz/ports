"use client";

import * as React from "react";
import { DrawablyCard } from "drawably/react";

import { cn } from "@/lib/utils";

/**
 * The Napkin panel: a real div with a hand-drawn border sketched around
 * it by drawably. `rotate` tilts the whole sheet; when placing multiple
 * cards together, vary the rotation — nothing on a desk sits perfectly
 * straight. `bg` picks the paper. Padding and layout stay with the
 * caller via className.
 */
const ROTATIONS = {
  "-3": "-rotate-3",
  "-2": "-rotate-2",
  "-1": "-rotate-1",
  "0": "rotate-0",
  "1": "rotate-1",
  "2": "rotate-2",
  "3": "rotate-3",
} as const;

const BGS = {
  white: "bg-white",
  paper: "bg-paper",
  muted: "bg-panel-2",
} as const;

type CardProps = React.ComponentProps<typeof DrawablyCard> & {
  rotate?: keyof typeof ROTATIONS;
  bg?: keyof typeof BGS;
};

function Card({ className, rotate = "0", bg = "white", ...props }: CardProps) {
  return (
    <DrawablyCard
      data-slot="card"
      className={cn(ROTATIONS[rotate], BGS[bg], className)}
      {...props}
    />
  );
}

export { Card };