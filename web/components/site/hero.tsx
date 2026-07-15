import { buttonVariants } from "@/components/ui/button";

import { CopyPill } from "./copy-pill";

export function Hero() {
  return (
    <header className="gx mt-6 flex flex-col items-center gap-7 px-6 pb-20 pt-28 text-center">
      <h1 className="text-balance font-semibold tracking-[-0.04em] text-strong leading-[1.06] text-[clamp(2rem,5vw,4rem)]">
        See every port. Stop it by name.
      </h1>
      <p className="max-w-[620px] text-balance text-muted leading-[1.55] text-[clamp(15px,1.6vw,18px)]">
        Every listening port, with its framework, project, and uptime. Zero
        config, one static binary.
      </p>
      <div className="mt-2 flex flex-wrap items-stretch justify-center gap-3">
        <a
          href="/docs/installation"
          className={buttonVariants({ variant: "default" })}
        >
          Install ports ↗
        </a>
        <CopyPill />
      </div>
    </header>
  );
}
