import Link from "next/link";
import { Footer, Nav, Toc, type TocItem } from "@/components/landing/chrome";
import { C, Terminal } from "@/components/landing/code";
import { InstallTabs } from "@/components/landing/install-tabs";
import { PortMove } from "@/components/landing/port-move";
import { RouteFlow } from "@/components/landing/route-flow";
import { SITE } from "@/lib/site";

const SECTIONS: TocItem[] = [
  { id: "install", title: "Install" },
  { id: "names", title: "Every server, by name" },
  { id: "how-it-works", title: "How it works" },
  { id: "whats-running", title: "What's running" },
  { id: "run", title: "Start on a free port" },
  { id: "stop", title: "Stop it safely" },
  { id: "agents", title: "For coding agents" },
  { id: "resolution", title: "Names for every client" },
  { id: "promises", title: "What ports won't do" },
];

/**
 * The landing page reads like the first page of the docs: what ports is,
 * how to install it, and each thing it does with the command that does
 * it. Two diagrams explain the one idea that needs a picture — a name,
 * not a port — and everything else is prose and real terminal output.
 */
export default function Home() {
  return (
    <>
      <Nav />
      <div className="mx-auto max-w-[1040px] px-4 sm:px-6 lg:grid lg:grid-cols-[minmax(0,1fr)_180px] lg:gap-16">
        <main className="min-w-0 max-w-[700px] pt-12 pb-24 sm:pt-16">
          <h1 className="text-[2.1rem] leading-[1.15] font-semibold tracking-[-0.025em] text-balance sm:text-[2.5rem]">
            Every dev server on your machine, by name
          </h1>
          <P className="mt-4 text-[17px] text-muted-foreground">
            ports replaces port numbers with stable <C>.localhost</C> names, for every dev server on your machine,
            whether ports started it or not. See what&apos;s running, reach it by name, stop it safely. For
            people and coding agents.
          </P>

          <Terminal
            title="Your app"
            lines={[
              { del: "http://localhost:3000", note: "shop, or whichever app got there first" },
              { add: "http://shop.localhost", note: "shop, on any port, after any restart" },
            ]}
          />

          <P>
            Nothing about how you start servers changes. Run <C>npm run dev</C> the way you always do — from a
            terminal, an editor or <C>docker compose</C> — and <C>shop.localhost</C> works.
          </P>

          <Section id="install" title="Install">
            <InstallTabs />
            <P>
              One static binary for macOS, Linux and Windows, WSL included. No runtime to install, and no
              account.
            </P>
          </Section>

          <Section id="names" title="Every server, by name">
            <P>Start the router once. It runs in the background from then on, and again at login.</P>
            <Terminal
              lines={[
                { cmd: "ports proxy start" },
                { out: "✓ routing *.localhost on :80 — a systemd user unit, started at login", tone: "ok" },
              ]}
            />
            <P>
              Every web dev server now has a name. It comes from the project — its <C>package.json</C> name, or
              its folder — and the app&apos;s place in it:
            </P>
            <Table
              head={["Where it runs", "Its name"]}
              rows={[
                ["~/shop", "shop.localhost"],
                ["~/shop/apps/api", "api.shop.localhost"],
                ["a worktree of shop on fix/ui", "ui.shop.localhost"],
                ["the web service of a compose project in ~/shop", "web.shop.localhost"],
                ["anything else, named by you", "ports alias stripe 4242"],
              ]}
            />
            <P>
              Two projects that are both called shop are told apart once — the second becomes{" "}
              <C>shop-client</C>, after its parent folder — and remembered, so they never swap. Each app gets its
              own origin, so cookies and local storage stop leaking between them.
            </P>
          </Section>

          <Section id="how-it-works" title="How it works">
            <P>
              The router listens on this machine&apos;s loopback, and nowhere else. Every <C>*.localhost</C> name
              already resolves there in browsers, so nothing needs configuring. For each request it reads the
              Host header, looks the name up, and passes the request on — WebSockets and event streams included,
              so hot reload works as before.
            </P>
            <RouteFlow />
            <P>
              The route table isn&apos;t declared anywhere. ports builds it from the machine itself, every two
              seconds: which processes listen, which project each belongs to, which port is the app and which is
              its debugger. A name comes from the project, never the port, so a server that restarts somewhere
              else keeps it.
            </P>
            <PortMove />
            <P>
              A Host that isn&apos;t one of its names is refused with a <C>421</C>. That&apos;s what keeps a
              website from reaching your dev servers by pointing its own domain at <C>127.0.0.1</C>.
            </P>
          </Section>

          <Section id="whats-running" title="What's running">
            <P>
              <C>ports</C> lists every listener with what it is, the project it belongs to, and how long
              it&apos;s been up — the context <C>lsof</C> leaves out.
            </P>
            <Terminal
              lines={[
                { cmd: "ports" },
                { out: "PORT   PROCESS      PROJECT           UPTIME   URL", tone: "muted" },
                { out: "3000   Next.js      shop              14m      http://shop.localhost" },
                { out: "8000   FastAPI      shop              14m      http://api.shop.localhost" },
                { out: "5432   PostgreSQL   docker: shop-db   2d" },
                { out: "5173   Vite         admin             3h       http://admin.localhost" },
              ]}
            />
            <P>
              <C>ports here</C> narrows it to the project you&apos;re in, <C>ports find postgres</C> searches,
              and <C>ports doctor 3000</C> explains a port that won&apos;t free.
            </P>
          </Section>

          <Section id="run" title="Start on a free port">
            <P>
              <C>ports run</C> picks a free port, hands it over the way the framework takes one — <C>PORT</C>,
              or <C>--port</C> for Vite, Astro and Angular — and names the server for as long as it runs.
            </P>
            <Terminal
              lines={[
                { cmd: "ports run -- npm run dev" },
                { out: "→ http://shop.localhost (:4123)" },
                { out: "" },
                { cmd: "ports run -- npm run dev", note: "again, in another terminal" },
                {
                  out: "ports: shop is taken by another ports run (pid 48211, :4123) — name this one: ports run shop-2 -- …",
                  tone: "warn",
                },
              ]}
            />
            <P>
              When the project is already up, it says where instead of starting a second copy on the next free
              port.
            </P>
          </Section>

          <Section id="stop" title="Stop it safely">
            <Terminal
              lines={[
                { cmd: "ports stop api.shop --expect fastapi" },
                { out: "✓ stopped FastAPI (:8000)", tone: "ok" },
              ]}
            />
            <P>
              <C>--expect</C> is the guard: if anything the stop would reach isn&apos;t what you said, nothing is
              stopped. Set <C>PORTS_REQUIRE_EXPECT=1</C> and every stop must carry it. A stop sends SIGTERM,
              waits, then SIGKILL; another user&apos;s process is never touched.
            </P>
          </Section>

          <Section id="agents" title="For coding agents">
            <P>
              Agents start a second dev server on the next free port, then test the wrong one. With ports they
              check first, reuse what&apos;s running, and reach it by name:
            </P>
            <Terminal
              lines={[
                { cmd: "ports here || (ports run -- npm run dev &)" },
                { cmd: "ports wait shop --http" },
                { cmd: 'curl "$(ports url shop)/api/health"' },
              ]}
            />
            <P>
              Every command speaks JSON with <C>--json</C>, and exits <C>0</C> when it found something, <C>1</C>{" "}
              when it didn&apos;t, <C>2</C> when it was called wrong. Or give the agent the commands as tools:
            </P>
            <Terminal lines={[{ cmd: "claude mcp add ports -- ports mcp" }]} />
            <P>
              The MCP server offers <C>list</C>, <C>here</C>, <C>find</C>, <C>url</C>, <C>wait</C>,{" "}
              <C>stop</C> and <C>doctor</C>. It starts nothing, and its <C>stop</C> always needs{" "}
              <C>expect</C>.
            </P>
          </Section>

          <Section id="resolution" title="Names for every client">
            <P>
              Browsers and curl send every <C>*.localhost</C> name to this machine on their own. Node&apos;s{" "}
              <C>fetch</C>, Python and Go ask the system instead, and on Windows — or a Linux that skips
              systemd-resolved — the system doesn&apos;t know them.
            </P>
            <Terminal
              lines={[
                { cmd: "ports doctor" },
                { out: "· this machine's resolver doesn't find shop, api.shop", tone: "muted" },
                { out: "  → `ports hosts sync` writes them to the hosts file" },
                { cmd: "ports hosts sync" },
                { out: "✓ /etc/hosts resolves 2 name(s) for every client: api.shop, shop", tone: "ok" },
              ]}
            />
            <P>
              The names go in one marked block, and nothing else in the file is touched. <C>ports hosts clean</C>{" "}
              takes them out again.
            </P>
          </Section>

          <Section id="promises" title="What ports won't do">
            <ul className="my-4 grid gap-2.5 text-[15px] leading-7">
              <Li>
                <b className="font-medium">Listen beyond this machine.</b> The router binds <C>127.0.0.1</C> and{" "}
                <C>::1</C>, and serves only the <C>.localhost</C> names it knows.
              </Li>
              <Li>
                <b className="font-medium">Run anything you didn&apos;t ask for.</b> Only <C>ports run</C> starts a
                process, and only the command you gave it. No restarts, no log collection, no supervision.
              </Li>
              <Li>
                <b className="font-medium">Route someone else&apos;s server.</b> The router routes the dev servers
                of the user it runs as.
              </Li>
              <Li>
                <b className="font-medium">Phone home.</b> One binary, local-first: no account, no cloud, no
                telemetry.
              </Li>
            </ul>
            <P>
              Read the <Link href={SITE.links.docs} className="underline underline-offset-4 hover:text-foreground">docs</Link>{" "}
              for every command and flag, or the source on{" "}
              <a href={SITE.links.github} className="underline underline-offset-4 hover:text-foreground">
                GitHub
              </a>
              .
            </P>
          </Section>
        </main>
        <aside className="hidden lg:block lg:pt-16">
          <Toc items={SECTIONS} />
        </aside>
      </div>
      <Footer />
    </>
  );
}

function Section({ id, title, children }: { id: string; title: string; children: React.ReactNode }) {
  return (
    <section id={id} className="mt-14 scroll-mt-20">
      <h2 className="group flex items-baseline gap-2 text-xl font-semibold tracking-[-0.015em]">
        {title}
        <a
          href={`#${id}`}
          aria-label={`Link to ${title}`}
          className="text-muted-foreground opacity-0 transition-opacity duration-150 group-hover:opacity-100 focus:opacity-100"
        >
          #
        </a>
      </h2>
      <div className="mt-3">{children}</div>
    </section>
  );
}

function P({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <p className={`my-4 text-[15px] leading-7 text-foreground/85 ${className}`}>{children}</p>;
}

function Li({ children }: { children: React.ReactNode }) {
  return (
    <li className="relative pl-5 text-foreground/85 before:absolute before:top-[0.8em] before:left-1 before:size-1 before:rounded-full before:bg-muted-foreground/60">
      {children}
    </li>
  );
}

function Table({ head, rows }: { head: string[]; rows: string[][] }) {
  return (
    <div className="my-5 overflow-x-auto rounded-lg border">
      <table className="w-full text-left text-sm">
        <thead className="bg-muted/40 text-xs text-muted-foreground">
          <tr>
            {head.map((h) => (
              <th key={h} className="px-3.5 py-2 font-normal">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r[0]} className="border-t">
              <td className="px-3.5 py-2 text-foreground/85">{r[0]}</td>
              <td className="px-3.5 py-2 font-mono text-[13px]">{r[1]}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
