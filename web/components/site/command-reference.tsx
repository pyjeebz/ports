import { CopyButton } from "./copy-button";

const REF = [
  {
    cmd: "ports",
    desc: "List every listening port with its framework, project, and uptime.",
  },
  {
    cmd: "ports find <name>",
    desc: "Filter by framework, process, project, or container. Case-insensitive.",
  },
  { cmd: "ports free <port>", desc: "Kill whatever holds a port." },
  {
    cmd: "ports kill <name>",
    desc: "Stop every process in a project or service — containers included.",
  },
  { cmd: "--json", desc: "Machine-readable output for scripts and coding agents." },
  { cmd: "--force", desc: "Skip SIGTERM and kill immediately." },
];

export function CommandReference() {
  return (
    <div className="reference gx">
      <div className="col">
        <div className="sec-head">
          <span className="eyebrow">Reference</span>
          <h2 className="sec-title">Four commands and two flags.</h2>
        </div>
        <div className="ref-table">
          {REF.map((r) => (
            <div className="ref-row" key={r.cmd}>
              <code className="ref-cmd">{r.cmd}</code>
              <span className="ref-desc">{r.desc}</span>
              <CopyButton text={r.cmd} />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
