import type { TrajectoryEvent } from "./types";

export const json = (value: unknown) => JSON.stringify(value, null, 2);

export function eventText(event: TrajectoryEvent): string {
  return [event.kind, event.title, event.summary, json(event.content), json(event.data), json(event.input), json(event.output), json(event.metadata)]
    .filter(Boolean).join(" ").toLowerCase();
}

export function payload(event: TrajectoryEvent): unknown {
  if (event.content !== undefined) return event.content;
  if (event.data !== undefined) return event.data;
  if (event.input !== undefined || event.output !== undefined) return { input: event.input, output: event.output };
  if (event.raw !== undefined) return event.raw;
  return event;
}

export function preview(value: unknown, limit = 700): string {
  const text = typeof value === "string" ? value : json(value);
  return text.length > limit ? `${text.slice(0, limit)}\n… ${text.length - limit} more characters` : text;
}

export function duration(ms?: number): string {
  if (ms === undefined) return "—";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(ms < 10000 ? 2 : 1)}s`;
}

export function time(timestamp?: string): string {
  if (!timestamp) return "";
  const parsed = new Date(timestamp);
  return Number.isNaN(parsed.valueOf()) ? timestamp : parsed.toLocaleTimeString([], { hour12: false, hour: "2-digit", minute: "2-digit", second: "2-digit", fractionalSecondDigits: 3 });
}

export function title(event: TrajectoryEvent): string {
  if (event.title || event.summary) return event.title || event.summary!;
  if (event.kind === "tool" && typeof event.input === "object" && event.input && "name" in event.input && typeof event.input.name === "string") return event.input.name;
  if (typeof event.data === "object" && event.data && "message" in event.data && typeof event.data.message === "string") return event.data.message;
  return event.kind.replaceAll("_", " ");
}
