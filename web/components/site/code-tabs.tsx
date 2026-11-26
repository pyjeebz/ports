import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

/**
 * shadcn's line tabs, for MDX: the list sits on the code block's top edge,
 * and each panel holds a fenced code block written right in the MDX, so it
 * is highlighted and copyable like any other.
 */
export function CodeTabs({ defaultValue, children }: { defaultValue: string; children: React.ReactNode }) {
  return (
    <Tabs defaultValue={defaultValue} className="not-prose my-6 gap-0 [&_figure]:my-0 [&_figure]:rounded-t-none">
      {children}
    </Tabs>
  );
}

export function CodeTabsList({ children }: { children: React.ReactNode }) {
  return (
    <TabsList variant="line" className="h-10 w-full justify-start rounded-t-lg rounded-b-none border border-b-0 bg-chrome px-2">
      {children}
    </TabsList>
  );
}

export const CodeTab = ({ value, children }: { value: string; children: React.ReactNode }) => (
  <TabsTrigger value={value} className="flex-none px-2.5 text-[13px] font-normal">
    {children}
  </TabsTrigger>
);

export const CodeTabPanel = ({ value, children }: { value: string; children: React.ReactNode }) => (
  <TabsContent value={value}>{children}</TabsContent>
);
