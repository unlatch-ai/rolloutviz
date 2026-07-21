import { describe, expect, it } from "vitest";
import { attentionScore, stagesFor } from "./instrument";
import type { BrowseTrajectory, ComparisonSide } from "./types";

const row = (id: string, values: { errors?: number; pass?: boolean; reward?: number }): BrowseTrajectory => ({
  source_id: "source",
  source_name: "gallery.ndjson",
  trajectory: { id },
  metrics: {
    trajectory: { id },
    event_count: 10,
    error_count: values.errors ?? 0,
    pass: values.pass,
    reward: values.reward,
  },
});

describe("instrument projections", () => {
  it("orders Browse attention above marginal and nominal rollouts", () => {
    const ordered = [
      row("nominal", { pass: true, reward: 1 }),
      row("marginal", { pass: true, reward: 0.4 }),
      row("policy-failure", { pass: false, reward: -1 }),
      row("tool-errors", { errors: 2, pass: false, reward: -1 }),
    ].sort((left, right) => attentionScore(right) - attentionScore(left));
    expect(ordered.map(({ trajectory }) => trajectory.id)).toEqual(["tool-errors", "policy-failure", "marginal", "nominal"]);
  });

  it("chooses adapter stage anchors instead of outcome-only alignment when present", () => {
    const anchored: ComparisonSide = {
      trajectory: { id: "anchored" },
      events: [
        { id: "start", sequence: 0, kind: "message" },
        { id: "work", sequence: 1, kind: "tool", alignment_key: "stage:work" },
        { id: "outcome", sequence: 2, kind: "grader", alignment_key: "stage:outcome" },
      ],
    };
    expect(stagesFor(anchored)).toMatchObject({
      tier: "adapter episode boundaries",
      stages: [{ key: "opening" }, { key: "stage:work" }, { key: "stage:outcome" }],
    });
    expect(stagesFor({ trajectory: { id: "plain" }, events: anchored.events.map(({ alignment_key: _alignment, ...event }) => event) })).toMatchObject({
      tier: "outcome only",
      stages: [{ key: "outcome" }],
    });
  });
});
