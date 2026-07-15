export function Workflow() {
  return (
    <div className="workflow gx">
      <div className="col wf-grid">
        <div className="wf-copy">
          <span className="eyebrow">How it works</span>
          <h2>Your whole machine, already explained.</h2>
          <p>
            Instead of raw PIDs and process names: framework names, project
            directories, and uptime — grouped so a frontend, API, and database
            that belong together read as one application.
          </p>
          <a href="/docs">Read the docs</a>
        </div>
        <div className="diagram">
          <div className="d-card">
            <span className="d-icon">⌗</span>
            <div>
              <h4>Your machine</h4>
              <p>sockets · processes · working directories</p>
            </div>
          </div>
          <span className="d-arrow">↓</span>
          <div className="d-card hot">
            <span className="d-icon">▸</span>
            <div>
              <h4>ports</h4>
              <p>one command, zero config</p>
            </div>
          </div>
          <span className="d-arrow">↓</span>
          <div className="d-rows">
            <div className="d-row">
              Local projects — git roots &amp; marker files{" "}
              <span className="arr">↗</span>
            </div>
            <div className="d-row">
              Docker — containers &amp; Compose services{" "}
              <span className="arr">↗</span>
            </div>
            <div className="d-row">
              WSL2 — Windows-side listeners <span className="arr">↗</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
