import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium leading-none cursor-pointer transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-faint focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-[#fafafa] text-[#18181b] hover:bg-[#e4e4e7]",
        outline:
          "border border-line text-foreground hover:bg-panel-2 hover:border-faint",
        ghost: "text-muted hover:text-foreground",
      },
      size: {
        default: "h-10 px-[18px]",
        sm: "h-9 px-[14px] text-[13.5px]",
      },
    },
    defaultVariants: { variant: "default", size: "default" },
  },
);

function Button({
  className,
  variant,
  size,
  ...props
}: React.ComponentProps<"button"> & VariantProps<typeof buttonVariants>) {
  return (
    <button
      data-slot="button"
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  );
}

export { Button, buttonVariants };
