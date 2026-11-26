import { EyeOff, Lock, Play, UserRound } from "lucide-react";
import { Card } from "@/components/ui/card";

const PROMISES = [
  {
    icon: Lock,
    title: "Listen beyond this machine",
    body: "The router binds 127.0.0.1 and ::1, and answers only the .localhost names it knows.",
  },
  {
    icon: Play,
    title: "Run what you didn't ask for",
    body: "Only ports run starts a process, and only the command you gave it. No restarts, no supervising.",
  },
  {
    icon: UserRound,
    title: "Route someone else's server",
    body: "It routes the dev servers of the user it runs as, and stops only yours.",
  },
  {
    icon: EyeOff,
    title: "Phone home",
    body: "One static binary. No account, no cloud, no telemetry.",
  },
];

/** The four things ports won't do, as a grid of small cards: each a
 *  promise with the fact that keeps it. */
export function Promises() {
  return (
    <div className="not-prose my-6 grid gap-3 sm:grid-cols-2">
      {PROMISES.map(({ icon: Icon, title, body }) => (
        <Card key={title} className="gap-2 rounded-lg p-4 shadow-none">
          <Icon className="size-4 text-muted-foreground" strokeWidth={1.75} />
          <p className="mt-1 text-[15px] font-medium tracking-[-0.01em]">{title}</p>
          <p className="text-sm leading-6 text-muted-foreground">{body}</p>
        </Card>
      ))}
    </div>
  );
}
