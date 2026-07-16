import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { App } from "./App";
import { sampleTrajectory } from "./sample";

describe("trajectory navigation", () => {
  it("navigates between events with j and k", () => {
    render(<App initialTrajectory={sampleTrajectory} />);
    expect(screen.getByText("Task prompt", { selector: ".selected-heading h3" })).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "j" });
    expect(screen.getByText("Assistant reasoning", { selector: ".selected-heading h3" })).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "k" });
    expect(screen.getByText("Task prompt", { selector: ".selected-heading h3" })).toBeInTheDocument();
  });

  it("jumps to errors and toggles raw JSON", () => {
    render(<App initialTrajectory={sampleTrajectory} />);
    fireEvent.keyDown(window, { key: "e" });
    expect(screen.getByText("Stale confirmation token", { selector: ".selected-heading h3" })).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "x" });
    expect(screen.getByText("Raw normalized record")).toBeInTheDocument();
    expect(screen.getByText(/STALE_CONFIRMATION/, { selector: ".raw-json" })).toBeInTheDocument();
  });

  it("opens search with slash and filters events", () => {
    render(<App initialTrajectory={sampleTrajectory} />);
    fireEvent.keyDown(window, { key: "/" });
    const search = screen.getByLabelText("Search events");
    expect(search).toHaveFocus();
    fireEvent.change(search, { target: { value: "submit_order" } });
    expect(screen.getAllByText("submit_order").length).toBeGreaterThan(0);
    expect(screen.queryByText("Task prompt", { selector: ".event-card h2" })).not.toBeInTheDocument();
  });
});
