import { PRINCIPLES } from "@/lib/site";

export function Principles() {
  return (
    <section className="border-b border-line-soft bg-panel py-16">
      <div className="mx-auto max-w-[1160px] px-8">
        <p className="eyebrow text-faint">▸ the contract</p>
        <div className="mt-6 grid grid-cols-1 gap-10 md:grid-cols-3">
          {PRINCIPLES.map((p, i) => (
            <div key={p.title}>
              <p className="readout">
                <b>{String(i + 1).padStart(2, "0")}</b> / {String(PRINCIPLES.length).padStart(2, "0")}
              </p>
              <h3 className="mt-3 font-mono text-[13.5px] font-medium tracking-[0.02em] text-accent">
                {p.title}
              </h3>
              <p className="mt-2 max-w-[38ch] text-[13.5px] leading-relaxed text-muted">
                {p.body}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}