import type { MDXComponents } from "mdx/types";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { Tab, Tabs } from "fumadocs-ui/components/tabs";

/**
 * Root MDX component convention — Next picks this up for fumadocs-mdx pages.
 * Base on fumadocs-ui defaults so docs pages get the shared prose styling,
 * then register the interactive components used in content.
 */
function resolveMdxComponents(components: MDXComponents): MDXComponents {
  return {
    ...defaultMdxComponents,
    ...components,
    Tabs,
    Tab,
  };
}

/** Root MDX component convention — Next picks this up for fumadocs-mdx pages. */
export const useMDXComponents: typeof resolveMdxComponents =
  resolveMdxComponents;

/** The `[[...slug]]` page renders `page.data.body` with these components. */
export function getMDXComponents(): MDXComponents {
  return resolveMdxComponents({});
}