export type EventKind =
  | "message"
  | "generation"
  | "tool"
  | "environment_action"
  | "observation"
  | "state"
  | "reward"
  | "grader"
  | "artifact"
  | "error"
  | "log";

export interface SourceLocation {
  path?: string;
  line?: number;
  byte_offset?: number;
  byte_length?: number;
  byte_start?: number;
  byte_end?: number;
}

export interface TrajectoryEvent {
  id: string;
  sequence: number;
  kind: EventKind | string;
  timestamp?: string;
  title?: string;
  summary?: string;
  content?: unknown;
  data?: unknown;
  input?: unknown;
  output?: unknown;
  metadata?: Record<string, unknown>;
  source?: SourceLocation;
  duration_ms?: number;
  token_count?: number;
  reward?: number;
  parent_id?: string;
  alignment_key?: string;
  state_hash?: string;
  raw?: unknown;
}

export interface Trajectory {
  id: string;
  name?: string;
  run_id?: string;
  case_id?: string;
  group_id?: string;
  model?: string;
  status?: string;
  started_at?: string;
  duration_ms?: number;
  total_reward?: number;
  metadata?: Record<string, unknown>;
  events: TrajectoryEvent[];
}

export interface TrajectoryResponse {
  trajectory?: Omit<Trajectory, "events"> & { events?: TrajectoryEvent[] };
  events?: TrajectoryEvent[];
  run?: { id: string; name?: string; started_at?: string; metadata?: Record<string, unknown> };
  case?: { id: string; run_id: string; name?: string; metadata?: Record<string, unknown> };
  group?: { id: string; case_id: string; name?: string; metadata?: Record<string, unknown> };
  signals?: Array<{ trajectory_id: string; event_id?: string; name: string; value: unknown; unit?: string }>;
  id?: string;
  name?: string;
  [key: string]: unknown;
}
