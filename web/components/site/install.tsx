import { SITE } from "@/lib/site";
import { CopyButton } from "./copy-button";

/** The install command as one snippet: the line to run, and a button to
 *  copy it. One channel for now — curl — so there's nothing to choose. */
export function Install() {
  return (
    <div className="not-prose my-6 flex items-center justify-between gap-3 rounded-lg border bg-card py-1.5 pr-1.5 pl-4 font-mono text-[13px]">
      <code className="overflow-x-auto whitespace-nowrap">
        <span className="text-muted-foreground select-none">$ </span>
        {SITE.installCurl}
      </code>
      <CopyButton text={SITE.installCurl} label="Copy install command" />
    </div>
  );
}
