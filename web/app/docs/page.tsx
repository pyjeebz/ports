import { CodeBlock } from "@/components/docs/code-block";
import { Toc } from "@/components/docs/toc";

export const metadata = { title: "Introduction" };

const LIST_HTML = `<span class="t-prompt">$ </span><span class="t-cmd">ports</span>
<span class="t-head">PORT     PROCESS       PROJECT                UPTIME</span>
3000     <span class="t-cmd">Next.js</span>       ~/projects/shop        14m
5432     <span class="t-cmd">PostgreSQL</span>    docker: shop-db        2d
8000     <span class="t-cmd">FastAPI</span>       ~/projects/shop        14m
5173     <span class="t-cmd">Vite</span>          ~/projects/admin       3h
11434    <span class="t-cmd">Ollama</span>        windows                ?`;

export default function Page() {
  return (
    <>
      <main className="doc">
        <article>
          <nav className="crumbs">
            Docs <span>/</span> <b>Introduction</b>
          </nav>
          <h1>Introduction</h1>
          <p className="subtitle">
            ports is a zero-config CLI that shows every listening port with its
            framework, project, and uptime — and lets you stop things by name.
          </p>

          <h2 id="what-is-ports">
            What is ports
            <a className="hash" href="#what-is-ports">
              #
            </a>
          </h2>
          <p>
            ports answers the question every developer hits multiple times a
            day: <strong>what is actually running on this machine right now, and
            how do I stop it?</strong>
          </p>
          <p>
            Instead of raw PIDs and process names, you get framework names,
            project directories, and uptime — grouped so a frontend, API, and
            database that belong together read as one application.
          </p>

          <CodeBlock head="terminal" copy="ports" html={LIST_HTML} />

          <div className="callout">
            <span className="ico">✓</span>
            <p>
              ports requires no configuration and no setup beyond installation.
              All context is inferred from the machine itself.
            </p>
          </div>

          <h2 id="highlights">
            Highlights
            <a className="hash" href="#highlights">
              #
            </a>
          </h2>
          <div className="cards">
            <div className="card">
              <div className="ico">
                <svg width="18" height="18" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <circle cx="10" cy="10" r="7" />
                  <path d="M10 6v4l2.5 2.5" />
                </svg>
              </div>
              <h4>Framework detection</h4>
              <p>
                A human-readable name for every service: framework detection
                first, then container image names, then raw process names.
              </p>
            </div>
            <div className="card">
              <div className="ico">
                <svg width="18" height="18" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <rect x="3" y="7" width="14" height="8" rx="1.5" />
                  <path d="M6.5 7V5.5h7V7M6.5 10.5h1.5M10 10.5h1.5" />
                </svg>
              </div>
              <h4>Docker unwrap</h4>
              <p>
                Shows real container and Compose service names instead of
                docker-proxy.
              </p>
            </div>
            <div className="card">
              <div className="ico">
                <svg width="18" height="18" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <rect x="3" y="4" width="14" height="9" rx="1.5" />
                  <path d="M3 8.5h14M8.5 4v9" />
                </svg>
              </div>
              <h4>WSL2-aware</h4>
              <p>
                Distinguishes Windows-side listeners from WSL processes in the
                same table.
              </p>
            </div>
            <div className="card">
              <div className="ico">
                <svg width="18" height="18" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M7 5.5c-1.5 0-2 1-2 2v1c0 1-.7 1.5-1.5 1.5.8 0 1.5.5 1.5 1.5v1c0 1 .5 2 2 2M13 5.5c1.5 0 2 1 2 2v1c0 1 .7 1.5 1.5 1.5-.8 0-1.5.5-1.5 1.5v1c0 1-.5 2-2 2" />
                </svg>
              </div>
              <h4>JSON output</h4>
              <p>
                Pass <code>--json</code> to any list or find command for
                scripting and automation.
              </p>
            </div>
          </div>

          <div className="rn">
            <h2 id="read-next-intro">Read next</h2>
            <div className="rn-cards">
              <a className="rn-card" href="/docs/installation">
                <span className="t">Installation</span>
                <span className="d">
                  Homebrew, Scoop, or curl. One static binary, no runtime.
                </span>
              </a>
              <a className="rn-card" href="/docs/quickstart">
                <span className="t">Quickstart</span>
                <span className="d">
                  List, find, and stop — all without touching a PID.
                </span>
              </a>
            </div>
          </div>
        </article>
      </main>
      <Toc
        items={[
          { href: "#what-is-ports", label: "What is ports" },
          { href: "#highlights", label: "Highlights" },
        ]}
      />
    </>
  );
}
