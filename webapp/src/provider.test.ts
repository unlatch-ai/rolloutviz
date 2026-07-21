import { describe, expect, it, vi } from "vitest";
import gallery from "../../examples/gallery/coding-agent-bugfix.ndjson?raw";
import type { BrowserCollection } from "./provider";
import { createInMemoryProvider } from "./provider";

vi.mock("./wasm", () => ({
  analyze: vi.fn(async () => ({ analysis: { api_version: "test", provenance: { name: "test", version: "1", digest: "x", input_digest: "y" }, findings: [], signals: [] }, cached: false, analyzed_at: "test" })),
  compare: vi.fn(async () => ({})),
}));

function galleryCollection(): BrowserCollection {
  const records = gallery.trim().split("\n").map((line) => JSON.parse(line) as Record<string, unknown>);
  const trajectory = records.find((record) => record.record_type === "trajectory")!;
  const events = records.filter((record) => record.record_type === "event");
  const signals = records.filter((record) => record.record_type === "signal");
  const artifacts = records.filter((record) => record.record_type === "artifact");
  const id = String(trajectory.id);
  const source = { id: "gallery", name: "coding-agent-bugfix.ndjson", format: "canonical-ndjson", size: gallery.length, index_state: "complete" };
  return {
    source,
    browse: { sources: [source], count: 1, trajectories: [{ source_id: source.id, source_name: source.name, case_name: "Fix flaky cache invalidation", group_name: "Bugfix attempts", trajectory: trajectory as never, metrics: { event_count: events.length, trajectory: trajectory as never } }] },
    trajectories: { [id]: { trajectory: trajectory as never, events: events as never, signals: signals as never, artifacts: artifacts as never, source, page: { count: events.length, total: events.length, limit: events.length, has_more: false } } },
  };
}

describe("in-memory viewer provider", () => {
  it("serves Browse and Read from the bundled 300-event gallery fixture", async () => {
    const provider = createInMemoryProvider(JSON.stringify(galleryCollection()));
    const browse = await provider.loadBrowse();
    expect(browse.trajectories).toHaveLength(1);
    expect(browse.trajectories[0].source_name).toBe("coding-agent-bugfix.ndjson");
    const loaded = await provider.loadInitial();
    expect(loaded.trajectory.events).toHaveLength(300);
    expect(loaded.trajectory.events[0].id).toBe("coding-event-0000");
  });
});
