import { SiteNav } from "@/components/site/site-nav";
import { Hero } from "@/components/site/hero";
import { PidHunt } from "@/components/site/pid-hunt";
import { PortAnatomy } from "@/components/site/port-anatomy";
import { SocketPipeline } from "@/components/site/socket-pipeline";
import { FourCommands } from "@/components/site/four-commands";
import { DockerTrace } from "@/components/site/docker-trace";
import { OneBinary } from "@/components/site/one-binary";
import { ObserverSplit } from "@/components/site/observer-split";
import { Faq } from "@/components/site/faq";
import { Close } from "@/components/site/close";
import { SiteFooter } from "@/components/site/site-footer";
import { DrawnLine } from "@/components/ui/drawn-line";

/**
 * The landing page as a product story: the claim with the CLI output as
 * the artwork, the problem it replaces, the model, the system, the
 * interface, the docker unwrap, the binary, the contract — then the
 * questions and the install. Between sections, a drawn rule: on paper
 * the page is separated by pen, not by borders.
 */
export default function Home() {
  return (
    <>
      <SiteNav />
      <main>
        <Hero />
        <DrawnLine className="mx-auto max-w-[1160px]" />
        <PidHunt />
        <PortAnatomy />
        <SocketPipeline />
        <DrawnLine className="mx-auto max-w-[1160px]" />
        <FourCommands />
        <DrawnLine className="mx-auto max-w-[1160px]" />
        <DockerTrace />
        <OneBinary />
        <ObserverSplit />
        <DrawnLine className="mx-auto max-w-[1160px]" />
        <Faq />
        <Close />
      </main>
      <SiteFooter />
    </>
  );
}