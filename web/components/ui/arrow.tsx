import { cn } from "@/lib/utils";

/**
 * A hand-drawn arrow between pipeline nodes. A static squiggle — one
 * path, drawn in ink — so it works from server components, unlike
 * drawably's live arrow which needs refs to two mounted elements. Flip
 * with -scale-x-100; color via text-*.
 */
function Arrow({ className, ...props }: React.ComponentProps<"svg">) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 56 14"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn("h-3.5 w-14 text-charcoal-light", className)}
      {...props}
    >
      {/* the shaft: two not-quite-even waves, the way a pen hurries */}
      <path d="M2 9c7-5 13 3 20-1s13 2 22-3" />
      {/* the head, a touch off-angle */}
      <path d="M38 1l6 5-7 4" />
    </svg>
  );
}

export { Arrow };