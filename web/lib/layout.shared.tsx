import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";
import { PortsMark } from "@/components/site/mark";
import { SITE } from "@/lib/site";

// The top bar every page shares: the mark, Docs, Releases, search,
// GitHub and the theme switch.
export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: (
        <span className="flex items-center gap-2 text-[15px] font-semibold tracking-[-0.02em]">
          <PortsMark className="size-[13px]" />
          ports
        </span>
      ),
      url: "/",
    },
    githubUrl: SITE.links.github,
    links: [
      { text: "Docs", url: SITE.links.docs, active: "nested-url" },
      { text: "Releases", url: SITE.links.releases, external: true },
    ],
  };
}
