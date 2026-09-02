import { cn } from "@/lib/utils";

/**
 * A strip of washi tape, laid by hand. Position it over a Card corner
 * with absolute classes (e.g. "-top-3 left-8 -rotate-3"); the torn-edge
 * clip keeps it from reading as a rectangle. Pure decoration — never
 * carries content, always aria-hidden.
 */
const COLORS = {
  clear: "bg-white/60",
  yellow: "bg-highlighter-yellow/70",
  pink: "bg-highlighter-pink/60",
} as const;

type TapeProps = React.ComponentProps<"span"> & {
  color?: keyof typeof COLORS;
};

function Tape({ className, color = "clear", ...props }: TapeProps) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "pointer-events-none absolute block h-6 w-24 opacity-80 [clip-path:polygon(1%_8%,12%_0,55%_5%,99%_0,97%_92%,88%_100%,45%_93%,2%_100%)]",
        COLORS[color],
        className,
      )}
      {...props}
    />
  );
}

export { Tape };