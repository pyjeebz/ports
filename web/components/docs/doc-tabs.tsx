"use client";

import { useState, type ReactNode } from "react";

export function DocTabs({
  tabs,
}: {
  tabs: { id: string; label: string; panel: ReactNode }[];
}) {
  const [active, setActive] = useState(tabs[0].id);
  return (
    <>
      <div className="tabs" role="tablist">
        {tabs.map((t) => (
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
      {tabs.map((t) => (
        <div
          key={t.id}
          className={`tabpanel${active === t.id ? " active" : ""}`}
        >
          {t.panel}
        </div>
      ))}
    </>
  );
}
