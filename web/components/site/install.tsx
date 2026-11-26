"use client";

import { useState } from "react";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { SITE } from "@/lib/site";
import { CopyButton } from "./copy-button";

const METHODS = [
  { id: "curl", label: "curl", command: SITE.installCurl },
  { id: "npm", label: "npm", command: SITE.installNpm },
];

/** The install command: curl first, npm beside it. One line to copy,
 *  whichever is chosen — switching tabs swaps the line without motion,
 *  since it's what the reader came to copy. */
export function Install() {
  const [method, setMethod] = useState(METHODS[0]);
  return (
    <Tabs
      value={method.id}
      onValueChange={(v) => setMethod(METHODS.find((m) => m.id === v) ?? METHODS[0])}
      className="not-prose my-6 gap-0 overflow-hidden rounded-lg border bg-card"
    >
      <div className="flex h-10 items-center justify-between border-b bg-chrome pr-1.5 pl-2">
        <TabsList variant="line" className="h-10 bg-transparent p-0">
          {METHODS.map((m) => (
            <TabsTrigger key={m.id} value={m.id} className="flex-none px-2.5 text-[13px] font-normal">
              {m.label}
            </TabsTrigger>
          ))}
        </TabsList>
        <CopyButton text={method.command} label="Copy install command" />
      </div>
      <code className="block overflow-x-auto px-4 py-3 font-mono text-[13px] whitespace-nowrap">
        <span className="text-muted-foreground select-none">$ </span>
        {method.command}
      </code>
    </Tabs>
  );
}
