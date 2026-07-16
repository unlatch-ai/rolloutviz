import { useEffect, useMemo, useRef, useState } from "react";
import { loadTrajectory } from "./api";
import { duration, eventText, json, payload, preview, time, title } from "./format";
import { sampleTrajectory } from "./sample";
import type { Trajectory, TrajectoryEvent } from "./types";

const kindMark: Record<string, string> = {
  message: "M", generation: "AI", tool: "T", environment_action: "A", observation: "O",
  state: "S", reward: "R", grader: "G", artifact: "F", error: "!", log: "L",
};
const filterKinds = ["all", "generation", "tool", "observation", "reward", "grader", "error"];

function Kind({ kind }: { kind: string }) {
  return <span className={`kind kind-${kind}`} aria-label={kind}>{kindMark[kind] || kind.slice(0, 2).toUpperCase()}</span>;
}

function Value({ value }: { value: unknown }) {
  if (value === undefined) return null;
  return <pre className="payload">{preview(value)}</pre>;
}

function TimelineCard({ event, selected, expanded, onSelect, onExpand }: {
  event: TrajectoryEvent; selected: boolean; expanded: boolean; onSelect: () => void; onExpand: () => void;
}) {
  const hasIO = event.input !== undefined || event.output !== undefined;
  return (
    <article id={`event-${event.id}`} className={`event-card kind-border-${event.kind} ${selected ? "selected" : ""}`} onClick={onSelect}>
      <div className="event-rail"><Kind kind={event.kind} /><span className="seq">{String(event.sequence).padStart(3, "0")}</span></div>
      <div className="event-body">
        <header>
          <div><span className="event-kind-label">{event.kind.replaceAll("_", " ")}</span><h2>{title(event)}</h2></div>
          <div className="event-stats">{event.reward !== undefined && <span className={event.reward < 0 ? "negative" : "positive"}>{event.reward > 0 ? "+" : ""}{event.reward}</span>}{event.duration_ms !== undefined && <span>{duration(event.duration_ms)}</span>}<span>{time(event.timestamp)}</span></div>
        </header>
        {event.summary && event.summary !== event.title && <p className="summary">{event.summary}</p>}
        {expanded ? (
          <div className="expanded-content">
            {hasIO ? <><div className="io-label">Input</div><Value value={event.input} /><div className="io-label">Output</div><Value value={event.output} /></> : <Value value={event.content} />}
          </div>
        ) : (payload(event) !== event && <pre className="preview">{preview(payload(event), 210)}</pre>)}
        <button className="expand" onClick={(e) => { e.stopPropagation(); onExpand(); }} aria-label={`${expanded ? "Collapse" : "Expand"} event ${event.sequence}`}>{expanded ? "Collapse" : "Expand"} <kbd>Space</kbd></button>
      </div>
    </article>
  );
}

function Inspector({ event, raw }: { event: TrajectoryEvent; raw: boolean }) {
  const entries = [
    ["Event ID", event.id], ["Sequence", event.sequence], ["Kind", event.kind], ["Time", event.timestamp],
    ["Duration", event.duration_ms === undefined ? undefined : duration(event.duration_ms)], ["Tokens", event.token_count],
    ["Reward", event.reward], ["Parent", event.parent_id], ["Alignment", event.alignment_key], ["State hash", event.state_hash],
  ].filter((entry) => entry[1] !== undefined) as [string, unknown][];
  return (
    <aside className="inspector">
      <div className="panel-heading"><span>Inspector</span><span className="panel-hint">x raw</span></div>
      <div className="inspector-scroll">
        <div className="selected-heading"><Kind kind={event.kind} /><div><h3>{title(event)}</h3><span>event {event.sequence}</span></div></div>
        {raw ? <section><h4>Raw normalized record</h4><pre className="raw-json">{json(event.raw ?? event)}</pre></section> : <>
          <section><h4>Properties</h4><dl>{entries.map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{String(value)}</dd></div>)}</dl></section>
          {event.source && <section><h4>Source</h4><div className="source-path">{event.source.path || "Unknown source"}</div><div className="source-detail">{event.source.line && `line ${event.source.line}`}{(event.source.byte_offset ?? event.source.byte_start) !== undefined && ` · bytes ${event.source.byte_offset ?? event.source.byte_start}–${event.source.byte_length !== undefined ? (event.source.byte_offset ?? 0) + event.source.byte_length : (event.source.byte_end ?? "?")}`}</div></section>}
          {event.input !== undefined && <section><h4>Input</h4><pre className="raw-json compact">{json(event.input)}</pre></section>}
          {event.output !== undefined && <section><h4>Output</h4><pre className="raw-json compact">{json(event.output)}</pre></section>}
          {(event.content ?? event.data) !== undefined && <section><h4>Content</h4><pre className="raw-json compact">{typeof (event.content ?? event.data) === "string" ? String(event.content ?? event.data) : json(event.content ?? event.data)}</pre></section>}
          {event.metadata && <section><h4>Metadata</h4><pre className="raw-json compact">{json(event.metadata)}</pre></section>}
        </>}
      </div>
    </aside>
  );
}

export function App({ initialTrajectory }: { initialTrajectory?: Trajectory }) {
  const [trajectory, setTrajectory] = useState(initialTrajectory || sampleTrajectory);
  const [isSample, setIsSample] = useState(!initialTrajectory);
  const [loading, setLoading] = useState(!initialTrajectory);
  const [selectedId, setSelectedId] = useState(trajectory.events[0]?.id || "");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState("all");
  const [query, setQuery] = useState("");
  const [searchOpen, setSearchOpen] = useState(false);
  const [raw, setRaw] = useState(false);
  const [help, setHelp] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (initialTrajectory) return;
    const controller = new AbortController();
    loadTrajectory(controller.signal).then(({ trajectory: next, isSample: sample }) => {
      setTrajectory(next); setIsSample(sample); setSelectedId(next.events[0]?.id || ""); setLoading(false);
    }).catch(() => setLoading(false));
    return () => controller.abort();
  }, [initialTrajectory]);

  const counts = useMemo(() => trajectory.events.reduce<Record<string, number>>((acc, e) => ({ ...acc, [e.kind]: (acc[e.kind] || 0) + 1 }), {}), [trajectory]);
  const visible = useMemo(() => trajectory.events.filter((event) => (filter === "all" || event.kind === filter) && (!query || eventText(event).includes(query.toLowerCase()))), [trajectory, filter, query]);
  const selected = trajectory.events.find((event) => event.id === selectedId) || visible[0] || trajectory.events[0];

  const move = (delta: number, predicate?: (event: TrajectoryEvent) => boolean) => {
    const candidates = predicate ? visible.filter(predicate) : visible;
    if (!candidates.length) return;
    const index = candidates.findIndex((event) => event.id === selectedId);
    const next = candidates[index < 0 ? 0 : (index + delta + candidates.length) % candidates.length];
    setSelectedId(next.id);
    requestAnimationFrame(() => document.getElementById(`event-${next.id}`)?.scrollIntoView({ block: "center", behavior: "smooth" }));
  };
  const toggleExpand = (id: string) => setExpanded((current) => { const next = new Set(current); next.has(id) ? next.delete(id) : next.add(id); return next; });

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const typing = event.target instanceof HTMLInputElement || event.target instanceof HTMLTextAreaElement;
      if (event.key === "Escape") { setSearchOpen(false); setHelp(false); searchRef.current?.blur(); return; }
      if (typing) return;
      if (event.key === "/") { event.preventDefault(); setSearchOpen(true); requestAnimationFrame(() => searchRef.current?.focus()); }
      else if (event.key === "j") move(1);
      else if (event.key === "k") move(-1);
      else if (event.key === "e") move(1, (item) => item.kind === "error");
      else if (event.key === "r") move(1, (item) => item.kind === "reward" || item.kind === "grader");
      else if (event.key === "x") setRaw((value) => !value);
      else if (event.key === "?") setHelp((value) => !value);
      else if ((event.key === " " || event.key === "Enter") && selected) { event.preventDefault(); toggleExpand(selected.id); }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });

  if (!selected) return <main className="empty">No events in this trajectory.</main>;
  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand"><span className="brand-mark">RV</span><span>RolloutViz</span></div>
        <div className="crumb"><span>{trajectory.run_id || "local run"}</span><b>/</b><strong>{trajectory.name || trajectory.id}</strong></div>
        <div className="top-actions">{loading && <span className="loading">Connecting…</span>}{isSample && !loading && <span className="demo-pill" title="The local API was unavailable">Sample data</span>}<button onClick={() => setHelp(true)} className="icon-button" aria-label="Keyboard shortcuts">?</button></div>
      </header>
      <div className="contextbar">
        <div className="status"><span className={`status-dot ${trajectory.status}`}></span>{trajectory.status || "complete"}</div>
        <div><span>MODEL</span>{trajectory.model || "—"}</div><div><span>EVENTS</span>{trajectory.events.length}</div><div><span>DURATION</span>{duration(trajectory.duration_ms)}</div><div><span>REWARD</span><strong className={(trajectory.total_reward || 0) < 0 ? "negative" : "positive"}>{trajectory.total_reward ?? "—"}</strong></div>
        <div className="context-spacer"></div><div><span>CASE</span>{trajectory.case_id || "—"}</div><div><span>GROUP</span>{trajectory.group_id || "—"}</div>
      </div>
      <div className="workspace">
        <aside className="outline">
          <div className="panel-heading"><span>Events</span><span>{visible.length}/{trajectory.events.length}</span></div>
          <div className={`search ${searchOpen ? "open" : ""}`}><span>⌕</span><input ref={searchRef} value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search events" aria-label="Search events" /><kbd>/</kbd></div>
          <div className="filters">{filterKinds.filter((kind) => kind === "all" || counts[kind]).map((kind) => <button key={kind} className={filter === kind ? "active" : ""} onClick={() => setFilter(kind)}><span>{kind}</span><b>{kind === "all" ? trajectory.events.length : counts[kind]}</b></button>)}</div>
          <nav className="event-outline" aria-label="Event outline">{visible.map((event) => <button key={event.id} className={selected.id === event.id ? "active" : ""} onClick={() => setSelectedId(event.id)}><Kind kind={event.kind} /><span className="outline-text"><b>{title(event)}</b><small>{event.kind} · {duration(event.duration_ms)}</small></span><span className="outline-seq">{event.sequence}</span></button>)}</nav>
          {!visible.length && <div className="no-results">No matching events</div>}
        </aside>
        <main className="timeline" aria-label="Trajectory timeline">
          <div className="timeline-heading"><div><h1>{trajectory.name || trajectory.id}</h1><p>{trajectory.id} · {trajectory.started_at ? new Date(trajectory.started_at).toLocaleString() : "local trajectory"}</p></div><div className="legend"><span><i className="action"></i>action</span><span><i className="observation"></i>observation</span><span><i className="signal"></i>signal</span></div></div>
          <div className="timeline-events">{visible.map((event) => <TimelineCard key={event.id} event={event} selected={selected.id === event.id} expanded={expanded.has(event.id)} onSelect={() => setSelectedId(event.id)} onExpand={() => toggleExpand(event.id)} />)}</div>
        </main>
        <Inspector event={selected} raw={raw} />
      </div>
      <footer className="keybar"><span><kbd>j</kbd><kbd>k</kbd> navigate</span><span><kbd>↵</kbd>/<kbd>space</kbd> expand</span><span><kbd>e</kbd> error</span><span><kbd>r</kbd> reward</span><span><kbd>x</kbd> raw</span><span><kbd>?</kbd> shortcuts</span></footer>
      {help && <div className="modal-backdrop" onMouseDown={() => setHelp(false)}><div className="help-modal" role="dialog" aria-modal="true" aria-label="Keyboard shortcuts" onMouseDown={(e) => e.stopPropagation()}><header><div><span className="eyebrow">Navigation</span><h2>Keyboard shortcuts</h2></div><button onClick={() => setHelp(false)} aria-label="Close shortcuts">×</button></header><div className="shortcut-grid">{[["j / k", "Next / previous event"], ["Enter / Space", "Expand selected event"], ["/", "Search events"], ["e", "Jump to next error"], ["r", "Jump to next reward or grader"], ["x", "Toggle raw event record"], ["?", "Toggle this reference"], ["Esc", "Close search or dialog"]].map(([key, label]) => <div key={key}><kbd>{key}</kbd><span>{label}</span></div>)}</div></div></div>}
    </div>
  );
}
