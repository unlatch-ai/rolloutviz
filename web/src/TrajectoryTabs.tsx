import { bindingLabel, commandIds, useKeymapRevision } from "./commands";

export type TrajectorySurface = "transcript" | "timeline" | "outcome";

const surfaces: Array<{ id: TrajectorySurface; label: string; command: typeof commandIds.trajectory.openTranscript | typeof commandIds.trajectory.openTimeline | typeof commandIds.trajectory.openOutcome }> = [
  { id: "transcript", label: "Transcript", command: commandIds.trajectory.openTranscript },
  { id: "timeline", label: "Events", command: commandIds.trajectory.openTimeline },
  { id: "outcome", label: "Outcome", command: commandIds.trajectory.openOutcome },
];

export function TrajectoryTabs({ active, onChange }: { active: TrajectorySurface; onChange: (surface: TrajectorySurface) => void }) {
  useKeymapRevision();
  return <nav className="trajectory-tabs" role="tablist" aria-label="Trajectory view">
    {surfaces.map((surface) => <button key={surface.id} type="button" role="tab" aria-selected={active === surface.id} className={active === surface.id ? "active" : ""} onClick={() => onChange(surface.id)}>
      <span>{surface.label}</span><kbd>{bindingLabel(surface.command)}</kbd>
    </button>)}
  </nav>;
}
