import { CodeBlock } from "@/components/docs/code-block";
import { Toc } from "@/components/docs/toc";

export const metadata = { title: "Commands" };

export default function Page() {
  return (
    <>
      <main className="doc">
        <article>
          <nav className="crumbs">
            Docs <span>/</span> <b>Commands</b>
          </nav>
          <h1>Commands</h1>
          <p className="subtitle">
            Four commands and two flags. That&apos;s the whole tool.
          </p>

          <h2 id="cmd-list">
            ports · ports list
            <a className="hash" href="#cmd-list">
              #
            </a>
          </h2>
          <p>
            Display every listening TCP port with its framework, project
            directory, and uptime. Supports <code>--json</code> output.
          </p>
          <div className="params">
            <div className="param">
              <span className="k">PORT</span>
              <span className="v">The TCP port number.</span>
            </div>
            <div className="param">
              <span className="k">PROCESS</span>
              <span className="v">
                The human-readable name of the service — framework detection
                first, then container image names, then raw process names.
              </span>
            </div>
            <div className="param">
              <span className="k">PROJECT</span>
              <span className="v">
                A local path, a <code>docker:</code> container reference,{" "}
                <code>windows</code>, or <code>—</code> when unknown.
              </span>
            </div>
            <div className="param">
              <span className="k">UPTIME</span>
              <span className="v">
                Duration with s, m, h, d suffixes — or <code>?</code> when
                unknown.
              </span>
            </div>
          </div>

          <h2 id="cmd-find">
            ports find &lt;name&gt;
            <a className="hash" href="#cmd-find">
              #
            </a>
          </h2>
          <p>
            Filter the table. Case-insensitive, matches substrings across
            framework, process, project, and container names.
          </p>
          <CodeBlock
            head="terminal"
            copy="ports find postgres"
            html={`<span class="t-prompt">$ </span><span class="t-cmd">ports find postgres</span>\n5432     <span class="t-cmd">PostgreSQL</span>    docker: shop-db        2d`}
          />

          <h2 id="cmd-free">
            ports free &lt;port&gt;
            <a className="hash" href="#cmd-free">
              #
            </a>
          </h2>
          <p>Kill whatever holds a port.</p>
          <CodeBlock
            head="terminal"
            copy="ports free 3000"
            html={`<span class="t-prompt">$ </span><span class="t-cmd">ports free 3000</span>\n<span class="t-ok">✓ killed Next.js (pid 48391) — port 3000 is free</span>`}
          />

          <h2 id="cmd-kill">
            ports kill &lt;name&gt;
            <a className="hash" href="#cmd-kill">
              #
            </a>
          </h2>
          <p>
            Stop all processes that belong to a named project or service.
            Docker-backed ports stop the container via the API — never
            docker-proxy.
          </p>
          <CodeBlock
            head="terminal"
            copy="ports kill shop"
            html={`<span class="t-prompt">$ </span><span class="t-cmd">ports kill shop</span>\n<span class="t-ok">✓ stopped Next.js (:3000)</span>\n<span class="t-ok">✓ stopped FastAPI (:8000)</span>\n<span class="t-ok">✓ stopped PostgreSQL (container shop-db)</span>`}
          />

          <h2 id="flags">
            Flags
            <a className="hash" href="#flags">
              #
            </a>
          </h2>
          <div className="params">
            <div className="param">
              <span className="k">--json</span>
              <span className="v">
                Receive a JSON array instead of the table. Use this when you
                want to pipe ports output into other tools or scripts.
              </span>
            </div>
            <div className="param">
              <span className="k">--force</span>
              <span className="v">Skip SIGTERM and kill immediately.</span>
            </div>
          </div>

          <h2 id="permissions">
            Permissions
            <a className="hash" href="#permissions">
              #
            </a>
          </h2>
          <div className="callout">
            <span className="ico">✓</span>
            <p>
              Some rows may show <code>?</code> in the PROCESS column — this
              happens when the socket belongs to a process owned by a different
              user and the OS does not expose its details without elevated
              privileges. Run <code>sudo ports</code> to reveal those processes.
            </p>
          </div>

          <div className="rn">
            <h2 id="read-next-cmd">Read next</h2>
            <div className="rn-cards">
              <a className="rn-card" href="/docs/quickstart">
                <span className="t">Quickstart</span>
                <span className="d">
                  List, find, and stop — all without touching a PID.
                </span>
              </a>
              <a className="rn-card" href="https://github.com/pyjeebz/ports">
                <span className="t">GitHub ↗</span>
                <span className="d">Source, issues, and releases.</span>
              </a>
            </div>
          </div>
        </article>
      </main>
      <Toc
        items={[
          { href: "#cmd-list", label: "ports list" },
          { href: "#cmd-find", label: "ports find" },
          { href: "#cmd-free", label: "ports free" },
          { href: "#cmd-kill", label: "ports kill" },
          { href: "#flags", label: "Flags" },
          { href: "#permissions", label: "Permissions" },
        ]}
      />
    </>
  );
}
