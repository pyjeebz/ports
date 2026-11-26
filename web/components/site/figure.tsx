"use client";

import { RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

/** Figure frames a diagram the way the page frames code: a card with a
 *  quiet header naming what it shows, and a button to play it again. */
export function Figure({
  title,
  onReplay,
  children,
  ref,
}: {
  title: string;
  onReplay: () => void;
  children: React.ReactNode;
  ref?: React.Ref<HTMLDivElement>;
}) {
  return (
    <Card ref={ref} className="not-prose my-6 gap-0 overflow-hidden rounded-lg py-0 shadow-none">
      <div className="flex h-10 items-center justify-between border-b bg-chrome pr-1.5 pl-4 text-[13px] text-muted-foreground">
        <span>{title}</span>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Replay"
              onClick={onReplay}
              className="size-8 text-muted-foreground transition-[transform,color,background-color] duration-150 ease-out hover:text-foreground active:scale-[0.96]"
            >
              <RotateCcw className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="top">Replay</TooltipContent>
        </Tooltip>
      </div>
      {children}
    </Card>
  );
}
