import type { MDXComponents } from "mdx/types";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { Tab, Tabs } from "fumadocs-ui/components/tabs";

/**
 * Root MDX component convention — Next picks this up for fumadocs-mdx pages.
 * Base on fumadocs-ui defaults so docs pages get the shared prose styling,
 * then register the interactive components used in content.
 */
export function useMDXComponents(components: MDXComponents): MDXComponents {
  return {
    ...defaultMdxComponents,
    ...components,
    Tabs,
    Tab,
  };
}