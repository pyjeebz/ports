import { CodeBlock } from "@/components/docs/code-block";
import { DocTabs } from "@/components/docs/doc-tabs";
import { Toc } from "@/components/docs/toc";

export const metadata = { title: "Installation" };

export default function Page() {
  return (
    <>
      <main className="doc">
        <article>
          <nav className="crumbs">
            Docs <span>/</span> <b>Installation</b>
          </nav>
          <h1>Installation</h1>
          <p className="subtitle">
            Install via Homebrew, Scoop, or curl. One static binary, no runtime.
          </p>

          <h2 id="install">
            Install
            <a className="hash" href="#install">
              #
            </a>
          </h2>
          <DocTabs
            tabs={[
              {
                id: "brew",
                label: "Homebrew",
                panel: (
                  <CodeBlock
                    head="macOS"
                    copy="brew install pyjeebz/tap/ports"
                    html={`<span class="t-prompt">$ </span><span class="t-cmd">brew install pyjeebz/tap/ports</span>`}
                  />
                ),
              },
              {
                id: "curl",
                label: "curl",
                panel: (
                  <CodeBlock
                    head="Linux, macOS"
                    copy="curl -fsSL https://ports.tools/install | sh"
                    html={`<span class="t-prompt">$ </span><span class="t-cmd">curl -fsSL https://ports.tools/install | sh</span>`}
                  />
                ),
              },
              {
                id: "scoop",
                label: "Scoop",
                panel: (
                  <CodeBlock
                    head="Windows"
                    copy="scoop bucket add pyjeebz https://github.com/pyjeebz/scoop-bucket && scoop install ports"
                    html={`<span class="t-prompt">$ </span><span class="t-cmd">scoop bucket add pyjeebz https://github.com/pyjeebz/scoop-bucket</span>\n<span class="t-prompt">$ </span><span class="t-cmd">scoop install ports</span>`}
                  />
                ),
              },
            ]}
          />

          <h2 id="verify">
            Verify
            <a className="hash" href="#verify">
              #
            </a>
          </h2>
          <CodeBlock
            head="terminal"
            copy="ports --version"
            html={`<span class="t-prompt">$ </span><span class="t-cmd">ports --version</span>\nports 0.1.0`}
          />

          <div className="callout">
            <span className="ico">✓</span>
            <p>
              That&apos;s it — there is no step two. Run <code>ports</code> and
              you&apos;ll see everything listening on your machine.
            </p>
          </div>

          <div className="rn">
            <h2 id="read-next-install">Read next</h2>
            <div className="rn-cards">
              <a className="rn-card" href="/docs/quickstart">
                <span className="t">Quickstart</span>
                <span className="d">
                  List, find, and stop — all without touching a PID.
                </span>
              </a>
              <a className="rn-card" href="/docs/commands">
                <span className="t">Commands</span>
                <span className="d">
                  Four commands and two flags. That&apos;s the whole tool.
                </span>
              </a>
            </div>
          </div>
        </article>
      </main>
      <Toc
        items={[
          { href: "#install", label: "Install" },
          { href: "#verify", label: "Verify" },
        ]}
      />
    </>
  );
}
