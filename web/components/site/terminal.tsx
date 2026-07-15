"use client";

import { useEffect, useRef } from "react";

const TERM_HTML = `<span class="t-line"><span class="t-prompt">$ </span><span class="t-proc" data-type>ports</span></span><span class="t-line"><span class="t-head">PORT     PROCESS       PROJECT                UPTIME</span></span><span class="t-line"><span>3000</span>     <span class="t-proc">Next.js</span>       <span class="t-dim">~/projects/shop</span>        14m</span><span class="t-line"><span>5432</span>     <span class="t-proc">PostgreSQL</span>    <span class="t-dim">docker: shop-db</span>        2d</span><span class="t-line"><span>8000</span>     <span class="t-proc">FastAPI</span>       <span class="t-dim">~/projects/shop</span>        14m</span><span class="t-line"><span>5173</span>     <span class="t-proc">Vite</span>          <span class="t-dim">~/projects/admin</span>       3h</span><span class="t-line"><span>11434</span>    <span class="t-proc">Ollama</span>        <span class="t-dim">windows</span>                ?</span><span class="t-line t-gap">&nbsp;</span><span class="t-line"><span class="t-prompt">$ </span><span class="t-proc" data-type>ports find postgres</span></span><span class="t-line"><span>5432</span>     <span class="t-proc">PostgreSQL</span>    <span class="t-dim">docker: shop-db</span>        2d</span><span class="t-line t-gap">&nbsp;</span><span class="t-line"><span class="t-prompt">$ </span><span class="t-proc" data-type>ports free 3000</span></span><span class="t-line t-ok">✓ killed Next.js (pid 48391) — port 3000 is free</span><span class="t-line t-gap">&nbsp;</span><span class="t-line"><span class="t-prompt">$ </span><span class="t-proc" data-type>ports kill shop</span></span><span class="t-line t-ok">✓ stopped FastAPI (:8000)</span><span class="t-line t-ok">✓ stopped PostgreSQL (container shop-db)</span><span class="t-line"><span class="t-prompt">$ </span><span class="cursor"></span></span>`;

export function Terminal() {
  const preRef = useRef<HTMLPreElement>(null);
  const replayRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const pre = preRef.current;
    if (!pre) return;
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
    const lines = Array.from(pre.querySelectorAll<HTMLElement>(".t-line"));
    let playing = false;

    async function typeInto(span: HTMLElement, text: string) {
      span.textContent = "";
      for (const ch of text) {
        span.textContent += ch;
        await sleep(42 + Math.random() * 40);
      }
    }

    async function play() {
      if (playing || reduced) return;
      playing = true;
      const cmds = new Map<HTMLElement, { span: HTMLElement; text: string }>();
      lines.forEach((l) => {
        const t = l.querySelector<HTMLElement>("[data-type]");
        if (t) cmds.set(l, { span: t, text: t.textContent || "" });
        l.classList.add("hide");
      });
      for (const l of lines) {
        l.classList.remove("hide");
        const c = cmds.get(l);
        if (c) {
          await typeInto(c.span, c.text);
          await sleep(380);
        } else if (l.classList.contains("t-gap")) await sleep(700);
        else if (l.classList.contains("t-ok")) await sleep(260);
        else await sleep(95);
      }
      playing = false;
    }

    const replay = replayRef.current;
    const onReplay = () => play();
    replay?.addEventListener("click", onReplay);

    let io: IntersectionObserver | null = new IntersectionObserver(
      (es) => {
        if (es[0].isIntersecting) {
          play();
          io?.disconnect();
          io = null;
        }
      },
      { threshold: 0.3 },
    );
    io.observe(pre);

    return () => {
      io?.disconnect();
      replay?.removeEventListener("click", onReplay);
    };
  }, []);

  return (
    <div className="lifecycle gx">
      <div className="col">
        <div className="sec-head">
          <span className="eyebrow">The tool</span>
          <h2 className="sec-title">See it work in one session.</h2>
          <p className="sec-lead">
            List what&apos;s running, find a service, free a port, stop a
            project — the four commands, back to back.
          </p>
        </div>
        <div className="term-wrap">
          <div className="terminal">
            <div className="term-bar">
              <span className="term-title">~/projects — zsh</span>
              <button
                ref={replayRef}
                className="replay"
                aria-label="replay animation"
              >
                ↻ replay
              </button>
            </div>
            <div className="term-body">
              <pre ref={preRef} dangerouslySetInnerHTML={{ __html: TERM_HTML }} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
