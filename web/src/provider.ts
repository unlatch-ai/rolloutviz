import {
  loadAnalysis,
  loadBrowse,
  loadComparison,
  loadIndexedTrajectory,
  loadTrajectory,
} from "./api";
import type {
  AnalysisResponse,
  BrowseResponse,
  ComparisonResponse,
} from "./types";
import type { LoadResult } from "./api";

/** Data boundary shared by the loopback daemon and the static browser app. */
export interface ViewerProvider {
  loadInitial(signal?: AbortSignal): Promise<LoadResult>;
  loadBrowse(signal?: AbortSignal): Promise<BrowseResponse>;
  loadTrajectory(sourceId: string, trajectoryId: string, signal?: AbortSignal): Promise<LoadResult>;
  loadAnalysis(sourceId: string, trajectoryId: string, signal?: AbortSignal): Promise<AnalysisResponse>;
  loadComparison(sourceId: string, left: string, right: string, signal?: AbortSignal): Promise<ComparisonResponse>;
}

export const daemonProvider: ViewerProvider = {
  loadInitial: loadTrajectory,
  loadBrowse,
  loadTrajectory: loadIndexedTrajectory,
  loadAnalysis,
  loadComparison,
};
