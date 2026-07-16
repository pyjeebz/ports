import { readFileSync } from "fs";
import { join } from "path";

// Serve the install script at https://ports.tools/install so
// `curl -fsSL https://ports.tools/install | sh` works. Read at build time
// and baked into static output. Keep script.sh in sync with the repo-root
// install.sh (which the GitHub raw URL serves as a fallback).
export const dynamic = "force-static";

const script = readFileSync(join(process.cwd(), "app/install/script.sh"), "utf8");

export function GET() {
  return new Response(script, {
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "cache-control": "public, max-age=3600",
    },
  });
}
