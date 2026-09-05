import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ZoneCanvas } from "./ZoneCanvas";
import type { ZoneGrid } from "../types";

// A single row keeps the rect-index -> column mapping unambiguous: the
// mock's <Group> wraps each row (no onClick of its own) around a row of
// per-cell <Group>s (each with onClick), so only leaf cell Groups ever
// contain a <Rect> -- clicking a rect bubbles to its owning cell's
// onClick, and with one row, rect order equals column order exactly.
const GRID: ZoneGrid = {
  zone: {
    zoneId: "WH1-STOR-AMB",
    siteCode: "WH1",
    areaCode: "STOR",
    zoneCode: "AMB",
    temperatureClass: "Ambient",
    hazmat: false,
    status: "Active",
  },
  columns: [
    { aisleId: "WH1-STOR-AMB-A01", aisleCode: "A01", bay: "01", sequenceHint: 1 },
    { aisleId: "WH1-STOR-AMB-A01", aisleCode: "A01", bay: "02", sequenceHint: 1 },
  ],
  levels: ["01"],
  rows: [
    {
      level: "01",
      cells: [
        {
          positions: [
            { locationCode: "WH1-STOR-AMB-A01-01-01-A", position: "A", locationType: "PalletRack", status: "Active" },
          ],
        },
        null,
      ],
    },
  ],
};

describe("ZoneCanvas", () => {
  it("calls onCellClick with the correct column and level for a filled cell", async () => {
    const onCellClick = vi.fn();
    render(<ZoneCanvas grid={GRID} onCellClick={onCellClick} />);
    const rects = screen.getAllByTestId("konva-rect");
    await userEvent.click(rects[0]);
    expect(onCellClick).toHaveBeenCalledWith({ column: GRID.columns[0], level: "01" });
  });

  it("calls onCellClick for an empty gap cell too", async () => {
    const onCellClick = vi.fn();
    render(<ZoneCanvas grid={GRID} onCellClick={onCellClick} />);
    const rects = screen.getAllByTestId("konva-rect");
    await userEvent.click(rects[1]);
    expect(onCellClick).toHaveBeenCalledWith({ column: GRID.columns[1], level: "01" });
  });

  it("renders the filled cell's position letters as text", () => {
    render(<ZoneCanvas grid={GRID} />);
    expect(screen.getByText("A")).toBeInTheDocument();
  });

  it("renders a '+' placeholder for the empty gap cell", () => {
    render(<ZoneCanvas grid={GRID} />);
    expect(screen.getByText("+")).toBeInTheDocument();
  });

  it("does not throw when onCellClick is omitted", async () => {
    render(<ZoneCanvas grid={GRID} />);
    const rects = screen.getAllByTestId("konva-rect");
    await expect(userEvent.click(rects[0])).resolves.not.toThrow();
  });
});
