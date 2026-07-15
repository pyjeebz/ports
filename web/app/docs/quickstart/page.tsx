import { CodeBlock } from "@/components/docs/code-block";
import { Toc } from "@/components/docs/toc";

export const metadata = { title: "Quickstart" };

export default function Page() {
  return (
    <>
      <main className="doc">
        <article>
          <nav className="crumbs">
            Docs <span>/</span> <b>Quickstart</b>
          </nav>
          <h1>Quickstart</h1>
          <p className="subtitle">
            See everything listening on your machine, find the service you care
            about, and stop it — all without touching a PID.
          </p>

          <ol className="steps">
            <li className="step">
              <h3 id="list-running-ports">List running ports</h3>
              <p>
                Run <code>ports</code> with no arguments. Rows are sorted and
                grouped by project, and every process shows a framework or
                service name — not a raw process name.
              </p>
              <CodeBlock
                head="terminal"
                copy="ports"
                html={`<span class="t-prompt">$ </span><span class="t-cmd">ports</span>\n<span class="t-head">PORT     PROCESS       PROJECT                UPTIME</span>\n3000     <span class="t-cmd">Next.js</span>       ~/projects/shop        14m\n5432     <span class="t-cmd">PostgreSQL</span>    docker: shop-db        2d\n8000     <span class="t-cmd">FastAPI</span>       ~/projects/shop        14m`}
              />
            </li>
            <li className="step">
              <h3 id="find-a-service">Find a service</h3>
              <p>
                The search is case-insensitive and matches substrings —
                framework, process, project, or container.
              </p>
              <CodeBlock
                head="terminal"
                copy="ports find postgres"
                html={`<span class="t-prompt">$ </span><span class="t-cmd">ports find postgres</span>\n5432     <span class="t-cmd">PostgreSQL</span>    docker: shop-db        2d`}
              />
            </li>
            <li className="step">
              <h3 id="free-a-port">Free a port</h3>
              <p>Kill whatever holds a port, by number.</p>
              <CodeBlock
                head="terminal"
                copy="ports free 3000"
                html={`<span class="t-prompt">$ </span><span class="t-cmd">ports free 3000</span>\n<span class="t-ok">✓ killed Next.js (pid 48391) — port 3000 is free</span>`}
              />
            </li>
            <li className="step">
              <h3 id="stop-a-project">Stop a project</h3>
              <p>
                Stops all processes that belong to a named project or service —
                dev servers and containers alike.
              </p>
              <CodeBlock
                head="terminal"
                copy="ports kill shop"
                html={`<span class="t-prompt">$ </span><span class="t-cmd">ports kill shop</span>\n<span class="t-ok">✓ stopped FastAPI (:8000)</span>\n<span class="t-ok">✓ stopped PostgreSQL (container shop-db)</span>`}
              />
            </li>
          </ol>

          <div className="callout">
            <span className="ico">✓</span>
            <p>
              ports sends SIGTERM first and only escalates to SIGKILL if the
              process does not exit. Docker-backed ports stop the container
              through the daemon — never by killing docker-proxy.
            </p>
          </div>

          <div className="rn">
            <h2 id="read-next-quick">Read next</h2>
            <div className="rn-cards">
              <a className="rn-card" href="/docs/commands">
                <span className="t">Commands</span>
                <span className="d">
                  Four commands and two flags. That&apos;s the whole tool.
                </span>
              </a>
              <a className="rn-card" href="/docs">
                <span className="t">Introduction</span>
                <span className="d">
                  What ports is, and the context it infers for you.
                </span>
              </a>
            </div>
          </div>
        </article>
      </main>
      <Toc
        items={[
          { href: "#list-running-ports", label: "List running ports" },
          { href: "#find-a-service", label: "Find a service" },
          { href: "#free-a-port", label: "Free a port" },
          { href: "#stop-a-project", label: "Stop a project" },
        ]}
      />
    </>
  );
}
