"use client";

import * as React from "react";
import { DrawablyCheckbox } from "drawably/react";

import { cn } from "@/lib/utils";

/**
 * Napkin checkbox: the real input stays in the DOM inside the sketch
 * host span; drawably draws the box and the pen checkmark.
 */
function Checkbox({ className, ...props }: React.ComponentProps<typeof DrawablyCheckbox>) {
  return (
    <DrawablyCheckbox
      data-slot="checkbox"
      className={cn(
        "inline-flex items-center [&>input]:size-4 [&>input]:appearance-none [&>input]:bg-transparent [&>input]:outline-none [&>input]:checked:bg-transparent",
        className,
      )}
      {...props}
    />
  );
}

export { Checkbox };