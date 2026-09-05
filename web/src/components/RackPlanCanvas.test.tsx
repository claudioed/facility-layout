import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RackPlanCanvas, type PlannedCell } from "./RackPlanCanvas";

function buildCells(): Map<string, PlannedCell> {
  const cells = new Map<string, PlannedCell>();
  cells.set("01|01|A", { bay: "01", level: "01", position: "A", locationType: "PalletRack", included: true });
  cells.set("01|01|B", { bay: "01", level: "01", position: "B", locationType: "PalletRack", included: true });
  return cells;
}

describe("RackPlanCanvas", () => {
  it("calls onCellClick with the bay and level of the clicked cell", async () => {
    const onCellClick = vi.fn();
    render(
      <RackPlanCanvas bays={["01"]} levels={["01"]} cells={buildCells()} onCellClick={onCellClick} />,
    );
    const rect = screen.getByTestId("konva-rect");
    await userEvent.click(rect);
    expect(onCellClick).toHaveBeenCalledWith("01", "01");
  });

  it("renders included positions as text", () => {
    render(
      <RackPlanCanvas bays={["01"]} levels={["01"]} cells={buildCells()} onCellClick={() => {}} />,
    );
    expect(screen.getByText("A B")).toBeInTheDocument();
  });

  it("renders a '+' placeholder for a cell with no planned positions", () => {
    render(
      <RackPlanCanvas bays={["01"]} levels={["01"]} cells={new Map()} onCellClick={() => {}} />,
    );
    expect(screen.getByText("+")).toBeInTheDocument();
  });

  it("excludes a cell whose positions are all marked included=false", () => {
    const cells = new Map<string, PlannedCell>();
    cells.set("01|01|A", { bay: "01", level: "01", position: "A", locationType: "PalletRack", included: false });
    render(<RackPlanCanvas bays={["01"]} levels={["01"]} cells={cells} onCellClick={() => {}} />);
    // With every position excluded, the cell has no planned positions left,
    // so it renders the empty-gap placeholder, not "A".
    expect(screen.getByText("+")).toBeInTheDocument();
    expect(screen.queryByText("A")).not.toBeInTheDocument();
  });

  it("resolves multi-bay/multi-level coordinates to the right cell", async () => {
    const onCellClick = vi.fn();
    render(
      <RackPlanCanvas
        bays={["01", "02"]}
        levels={["01", "02"]}
        cells={buildCells()}
        onCellClick={onCellClick}
      />,
    );
    // Row-major order: (bay01,lvl01), (bay02,lvl01), (bay01,lvl02), (bay02,lvl02)
    const rects = screen.getAllByTestId("konva-rect");
    expect(rects).toHaveLength(4);
    await userEvent.click(rects[3]);
    expect(onCellClick).toHaveBeenCalledWith("02", "02");
  });
});
