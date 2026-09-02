"use client";

import * as React from "react";
import Link from "next/link";
import { cva, type VariantProps } from "class-variance-authority";
import { DrawablyButton } from "drawably/react";
import { drawablyButton } from "drawably";

import { cn } from "@/lib/utils";

/**
 * Napkin buttons: drawably draws the pen stroke around the real control.
 * primary is the inked-solid Sharpie action, secondary the outlined one,
 * ghost the bare text link. Hover lift / press sink come from drawably.
 * Button renders a <button>; ButtonLink an <a> with the same chrome.
 */
const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap px-4 py-2 font-bold leading-none cursor-pointer focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-charcoal focus-visible:outline-offset-2 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        primary: "bg-charcoal text-paper",
        secondary: "bg-white text-charcoal",
        ghost: "text-charcoal-light hover:text-charcoal",
      },
      size: {
        default: "text-[15px]",
        sm: "text-[13px]",
      },
    },
    defaultVariants: { variant: "primary", size: "default" },
  },
);

type Variant = NonNullable<VariantProps<typeof buttonVariants>["variant"]>;

function sketch(el: HTMLElement, variant?: Variant | null) {
  // ghost stays bare — no chrome to draw around a plain text action.
  if (variant === "ghost") return null;
  return drawablyButton(el, {
    variant: variant === "primary" ? "solid" : "outline",
    fill: "var(--charcoal)",
    paper: "var(--paper)",
  });
}

function Button({
  className,
  variant,
  size,
  ...props
}: React.ComponentProps<"button"> & VariantProps<typeof buttonVariants>) {
  const classes = cn(buttonVariants({ variant, size }), className);
  if (variant === "ghost") {
    return <button data-slot="button" className={classes} {...props} />;
  }
  return (
    <DrawablyButton
      data-slot="button"
      variant={variant === "primary" ? "solid" : "outline"}
      fill="var(--charcoal)"
      paper="var(--paper)"
      className={classes}
      {...props}
    />
  );
}

function ButtonLink({
  className,
  variant,
  size,
  href,
  ...props
}: React.ComponentProps<typeof Link> & VariantProps<typeof buttonVariants>) {
  const ref = React.useRef<HTMLAnchorElement>(null);
  // One fresh sketch per mount; drawably cleans its own SVG on destroy.
  React.useEffect(() => {
    if (!ref.current) return;
    return sketch(ref.current, variant)?.destroy;
  }, [variant]);
  return (
    <Link
      ref={ref}
      href={href}
      data-slot="button-link"
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  );
}

export { Button, ButtonLink, buttonVariants };