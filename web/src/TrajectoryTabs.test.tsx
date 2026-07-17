import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TrajectoryTabs } from "./TrajectoryTabs";

describe("trajectory view tabs", () => {
  it("exposes the active view and changes it explicitly", () => {
    const change = vi.fn();
    render(<TrajectoryTabs active="transcript" onChange={change} />);
    expect(screen.getByRole("tab", { name: /Transcript/ })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: /Events/ })).toHaveAttribute("aria-selected", "false");
    fireEvent.click(screen.getByRole("tab", { name: /Outcome/ }));
    expect(change).toHaveBeenCalledWith("outcome");
  });
});
