import { CAPABILITIES } from "@/lib/site";

export function CapabilityGrid() {
  return (
    <section className="border-b border-line-soft py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <p className="eyebrow">▸ capabilities</p>
        <h2 className="mt-4 max-w-[24ch] text-balance text-[clamp(1.5rem,2.6vw,2rem)] font-semibold leading-[1.2] tracking-[-0.03em] text-strong">
          What the scan already knows about you.
        </h2>
        <div className="mt-10 grid grid-cols-1 gap-px overflow-hidden rounded-md border border-line-soft bg-line-soft sm:grid-cols-2 lg:grid-cols-3">
          {CAPABILITIES.map((cap) => (
            <div key={cap.title} className="bg-panel p-6">
              <h3 className="text-[15px] font-semibold tracking-[-0.01em] text-strong">
                {cap.title}
              </h3>
              <p className="mt-2 text-[13.5px] leading-relaxed text-muted">
                {cap.body}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}