import { CopyButton } from "@/components/site/copy-button";

export function CodeBlock({
  head,
  copy,
  html,
}: {
  head: string;
  copy: string;
  html: string;
}) {
  return (
    <div className="code">
      <div className="code-head">
        <span>{head}</span>
        <CopyButton text={copy} />
      </div>
      <pre dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  );
}
