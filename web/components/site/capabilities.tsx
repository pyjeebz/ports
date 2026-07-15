const CAPS = [
  {
    k: "01",
    h: "Framework detection",
    p: "Next.js, Vite, FastAPI, Postgres, Redis — named the way you know them, not as PIDs.",
  },
  {
    k: "02",
    h: "Project grouping",
    p: "A frontend, API, and database that belong together read — and stop — as one application.",
  },
  {
    k: "03",
    h: "Docker unwrap",
    p: "Real container and Compose service names instead of docker-proxy.",
  },
  {
    k: "04",
    h: "WSL2-aware",
    p: "Windows-side listeners and WSL processes, distinguished in the same table.",
  },
  {
    k: "05",
    h: "JSON output",
    p: "Pass --json to any list or find command for scripting and automation.",
  },
  {
    k: "06",
    h: "Safe stops",
    p: "SIGTERM first, SIGKILL only if it won't exit. Never a reused PID.",
  },
];

export function Capabilities() {
  return (
    <div className="capabilities gx">
      <div className="col">
        <div className="sec-head">
          <span className="eyebrow">What it knows</span>
          <h2 className="sec-title">Context the OS won&apos;t give you.</h2>
        </div>
        <div className="cap-grid">
          {CAPS.map((c) => (
            <div className="cap" key={c.k}>
              <span className="k">{c.k}</span>
              <h3>{c.h}</h3>
              <p>{c.p}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
