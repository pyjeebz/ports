import { PortsLogo } from "./ports-logo";

export function SiteFooter() {
  return (
    <>
      <footer className="gx">
        <div className="col">
          <div className="foot-grid">
            <div className="foot-brand">
              <span className="brand">
                <PortsLogo
                  className="h-[18px] w-[18px]"
                  maskId="ports-slash-foot"
                />
                ports
              </span>
              <p>
                See what&apos;s running on your machine, by project — and kill it
                by name.
              </p>
              <span className="meta">MIT · v0.1.0</span>
            </div>
            <div className="foot-col">
              <h5>Product</h5>
              <a href="/docs/installation">Install</a>
              <a href="/">The tool</a>
              <a href="https://github.com/pyjeebz/ports/releases">Releases</a>
            </div>
            <div className="foot-col">
              <h5>Documentation</h5>
              <a href="/docs">Introduction</a>
              <a href="/docs/quickstart">Quickstart</a>
              <a href="/docs/commands">Commands</a>
            </div>
            <div className="foot-col">
              <h5>Project</h5>
              <a href="https://github.com/pyjeebz/ports">GitHub</a>
              <a href="https://github.com/pyjeebz/ports/releases">Changelog</a>
              <a href="/">ports.tools</a>
            </div>
          </div>
        </div>
      </footer>
      <div className="frame-close gx" />
    </>
  );
}
