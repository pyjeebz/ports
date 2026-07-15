"use client";

import { useState } from "react";

import { CopyButton } from "./copy-button";

const TABS = [
  {
    id: "brew",
    label: "Homebrew",
    head: "macOS",
    copy: "brew install pyjeebz/tap/ports",
    show: "brew install pyjeebz/tap/ports",
  },
  {
    id: "curl",
    label: "curl",
    head: "Linux, macOS",
    copy: "curl -fsSL https://ports.tools/install | sh",
    show: "curl -fsSL https://ports.tools/install | sh",
  },
  {
    id: "scoop",
    label: "Scoop",
    head: "Windows",
    copy: "scoop bucket add pyjeebz https://github.com/pyjeebz/scoop-bucket && scoop install ports",
    show: "scoop install ports",
  },
];

export function Install() {
  const [active, setActive] = useState("brew");
  return (
    <div className="install-sec gx">
      <div className="col">
        <div className="sec-head">
          <span className="eyebrow">Install</span>
          <h2 className="sec-title">One static binary. No runtime.</h2>
          <p className="sec-lead">
            Homebrew, Scoop, or curl — then run <code>ports</code>.
          </p>
        </div>
        <div className="install">
          <div className="tabs" role="tablist">
            {TABS.map((t) => (
              <button
                key={t.id}
                className="tab"
                role="tab"
                aria-selected={active === t.id}
                onClick={() => setActive(t.id)}
              >
                {t.label}
              </button>
            ))}
          </div>
          {TABS.map((t) => (
            <div
              key={t.id}
              className={`tabpanel${active === t.id ? " active" : ""}`}
            >
              <div className="code">
                <div className="code-head">
                  <span>{t.head}</span>
                  <CopyButton text={t.copy} />
                </div>
                <pre>
                  <span className="t-prompt">$ </span>
                  <span className="t-cmd">{t.show}</span>
                </pre>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
