import type { LoadResult } from "../../web/src/api";
import type { ViewerProvider } from "../../web/src/provider";
import type { AnalysisResponse, BrowseResponse, ComparisonResponse, TrajectoryResponse } from "../../web/src/types";
import { analyze, compare } from "./wasm";

export interface BrowserCollection {
  source: { id: string; name: string; format: string; size: number; index_state: string };
  browse: BrowseResponse;
  trajectories: Record<string, TrajectoryResponse & { page: { count: number; total: number; limit: number; has_more: boolean } }>;
}

export function createInMemoryProvider(encoded: string): ViewerProvider {
  const collection = JSON.parse(encoded) as BrowserCollection;
  const first = collection.browse.trajectories[0];
  if (!first) throw new Error("Trace contains no trajectories");
  const load = (id: string): LoadResult => {
    const value = collection.trajectories[id];
    if (!value) throw new Error(`Unknown trajectory ${id}`);
    if (!value.trajectory) throw new Error(`Trajectory ${id} has no metadata`);
    return {
      trajectory: { ...value.trajectory, events: value.events ?? [], signals: value.signals ?? [], artifacts: value.artifacts ?? [] },
      isSample: false,
      page: value.page,
      source: value.source,
    };
  };
  return {
    async loadInitial() { return load(first.trajectory.id); },
    async loadBrowse() { return collection.browse; },
    async loadTrajectory(_sourceId, trajectoryId) { return load(trajectoryId); },
    async loadAnalysis(_sourceId, trajectoryId) { return analyze(encoded, trajectoryId) as Promise<AnalysisResponse>; },
    async loadComparison(_sourceId, left, right) { return compare(encoded, left, right) as Promise<ComparisonResponse>; },
  };
}
