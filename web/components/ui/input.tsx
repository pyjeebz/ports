"use client";

import * as React from "react";
import { DrawablyInput } from "drawably/react";

import { cn } from "@/lib/utils";

/**
 * Napkin form field: the real input stays in the DOM (labels, forms and
 * screen readers untouched); drawably sketches the field border around
 * it. The host span carries layout classes, the inner input the rest.
 */
function Input({ className, ...props }: React.ComponentProps<typeof DrawablyInput>) {
  return (
    <DrawablyInput
      data-slot="input"
      className={cn(
        "inline-block bg-white [&>input]:w-full [&>input]:bg-transparent [&>input]:px-3 [&>input]:py-2 [&>input]:font-sans [&>input]:text-[15px] [&>input]:text-charcoal [&>input]:outline-none [&>input]:placeholder:text-faint",
        className,
      )}
      {...props}
    />
  );
}

export { Input };