import type { MDXComponents } from "mdx/types";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { Tab, Tabs } from "fumadocs-ui/components/tabs";
import { CodeTab, CodeTabPanel, CodeTabs, CodeTabsList } from "@/components/site/code-tabs";
import { Install } from "@/components/site/install";
import { PortMove } from "@/components/site/port-move";
import { Promises } from "@/components/site/promises";
import { RouteFlow } from "@/components/site/route-flow";

function resolveMdxComponents(components: MDXComponents): MDXComponents {
  return {
    ...defaultMdxComponents,
    ...components,
    Tabs,
    Tab,
    CodeTabs,
    CodeTabsList,
    CodeTab,
    CodeTabPanel,
    Install,
    RouteFlow,
    PortMove,
    Promises,
  };
}

/** Root MDX component convention — Next picks this up for fumadocs-mdx pages. */
export const useMDXComponents: typeof resolveMdxComponents = resolveMdxComponents;

/** The pages render `page.data.body` with these components. */
export function getMDXComponents(): MDXComponents {
  return resolveMdxComponents({});
}
