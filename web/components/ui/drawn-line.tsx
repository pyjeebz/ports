"use client";

import * as React from "react";
import { DrawablyDivider } from "drawably/react";

import { cn } from "@/lib/utils";

/**
 * The drawn divider — a pen rule between sections, drawn once per mount
 * and never two rows alike. Give it breathing room via className.
 */
function DrawnLine({ className, ...props }: React.ComponentProps<typeof DrawablyDivider>) {
  return (
    <DrawablyDivider
      data-slot="drawn-line"
      className={cn("my-0 border-0", className)}
      {...props}
    />
  );
}

export { DrawnLine };