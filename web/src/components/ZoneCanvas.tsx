import { useMemo, useState } from "react";
import { Stage, Layer, Rect, Text, Group } from "react-konva";
import type { ZoneGrid, GridCell } from "../types";

/**
 * Renders one Zone's grid (from GET /zones/{zoneId}/grid) as an actual
 * drawn floor plan using Konva's <Stage>/<Layer>/<Rect> canvas primitives --
 * columns are (Aisle, Bay) pairs in aisle walk order, rows are Levels, one
 * rect per cell. This is the literal "draw the warehouse" surface CLAUDE.md
 * calls for: a UI iterating rows x columns and painting, not a table of
 * text.
 *
 * A cell with no slots renders as an empty dashed outline (a gap in the
 * rack, per the API contract's `null` cell semantics translated to "no
 * positions"). A cell with slots renders filled, colored by whether any
 * slot in it is Decommissioned/UnderMaintenance vs all Active, and shows
 * its Position letters. Clicking any cell (empty or not) invokes
 * onCellClick so a caller can open a "register a slot here" form pinned to
 * that exact (aisle, bay, level) coordinate -- the configuration part of
 * this screen, distinct from this pure rendering component.
 */
export function ZoneCanvas({
  grid,
  onCellClick,
  selected,
}: {
  grid: ZoneGrid;
  onCellClick?: (args: { column: ZoneGrid["columns"][number]; level: string }) => void;
  /** Currently selected (columnIndex, rowIndex) cell, if any -- drawn with
   *  an accent outline so the paired "register slot" form stays visually
   *  anchored to the cell it will fill. */
  selected?: { columnIndex: number; rowIndex: number } | null;
}) {
  const CELL_W = 92;
  const CELL_H = 56;
  const GUTTER = 6;
  const LABEL_COL_W = 64;
  const LABEL_ROW_H = 44;

  const width = LABEL_COL_W + grid.columns.length * (CELL_W + GUTTER) + GUTTER;
  const height = LABEL_ROW_H + grid.rows.length * (CELL_H + GUTTER) + GUTTER;

  // Group columns visually by aisle so a run of bays under the same aisle
  // reads as one corridor, matching the physical structure being drawn.
  const aisleSpans = useMemo(() => {
    const spans: { aisleCode: string; start: number; count: number }[] = [];
    grid.columns.forEach((col, i) => {
      const last = spans[spans.length - 1];
      if (last && last.aisleCode === col.aisleCode) {
        last.count += 1;
      } else {
        spans.push({ aisleCode: col.aisleCode, start: i, count: 1 });
      }
    });
    return spans;
  }, [grid.columns]);

  const [hovered, setHovered] = useState<{ c: number; r: number } | null>(null);

  function cellTone(cell: GridCell | null): { fill: string; stroke: string } {
    if (!cell || cell.positions.length === 0) {
      return { fill: "transparent", stroke: "var(--wh-color-border-subtle)" };
    }
    const anyDown = cell.positions.some((p) => p.status !== "Active");
    if (anyDown) {
      return { fill: "#3a2c10", stroke: "#f5b942" };
    }
    return { fill: "#123729", stroke: "#3dd68c" };
  }

  return (
    <div style={{ overflow: "auto", borderRadius: "var(--wh-radius-md)" }}>
      <Stage width={Math.max(width, 360)} height={Math.max(height, 160)}>
        <Layer>
          {/* Aisle group labels along the top */}
          {aisleSpans.map((span) => (
            <Text
              key={`aisle-${span.aisleCode}-${span.start}`}
              x={LABEL_COL_W + span.start * (CELL_W + GUTTER) + GUTTER}
              y={0}
              width={span.count * (CELL_W + GUTTER) - GUTTER}
              height={18}
              align="center"
              text={`Aisle ${span.aisleCode}`}
              fontSize={11}
              fontStyle="600"
              fill="#8f9bad"
            />
          ))}

          {/* Bay labels (column headers) */}
          {grid.columns.map((col, c) => (
            <Text
              key={`bay-${col.aisleId}-${col.bay}`}
              x={LABEL_COL_W + c * (CELL_W + GUTTER) + GUTTER}
              y={20}
              width={CELL_W}
              height={20}
              align="center"
              text={`Bay ${col.bay}`}
              fontSize={11}
              fill="#5c6779"
              fontFamily="monospace"
            />
          ))}

          {/* Level labels (row headers) + cells */}
          {grid.rows.map((row, r) => {
            const y = LABEL_ROW_H + r * (CELL_H + GUTTER) + GUTTER;
            return (
              <Group key={`row-${row.level}`}>
                <Text
                  x={0}
                  y={y + CELL_H / 2 - 7}
                  width={LABEL_COL_W - 8}
                  align="right"
                  text={`Lvl ${row.level}`}
                  fontSize={12}
                  fontFamily="monospace"
                  fill="#8f9bad"
                />
                {row.cells.map((cell, c) => {
                  const x = LABEL_COL_W + c * (CELL_W + GUTTER) + GUTTER;
                  const tone = cellTone(cell);
                  const isHovered = hovered?.c === c && hovered?.r === r;
                  const isSelected = selected?.columnIndex === c && selected?.rowIndex === r;
                  return (
                    <Group
                      key={`cell-${c}-${r}`}
                      onMouseEnter={() => setHovered({ c, r })}
                      onMouseLeave={() => setHovered(null)}
                      onClick={() => onCellClick?.({ column: grid.columns[c], level: row.level })}
                      onTap={() => onCellClick?.({ column: grid.columns[c], level: row.level })}
                    >
                      <Rect
                        x={x}
                        y={y}
                        width={CELL_W}
                        height={CELL_H}
                        cornerRadius={4}
                        fill={tone.fill}
                        stroke={isSelected ? "#4d8dff" : tone.stroke}
                        strokeWidth={isSelected ? 2.5 : 1}
                        dash={!cell || cell.positions.length === 0 ? [4, 3] : undefined}
                        shadowColor="#000"
                        shadowBlur={isHovered ? 6 : 0}
                        shadowOpacity={0.4}
                      />
                      {cell && cell.positions.length > 0 ? (
                        <Text
                          x={x}
                          y={y}
                          width={CELL_W}
                          height={CELL_H}
                          align="center"
                          verticalAlign="middle"
                          text={cell.positions.map((p) => p.position).join(" ")}
                          fontSize={12}
                          fontFamily="monospace"
                          fontStyle="600"
                          fill="#e6edf5"
                        />
                      ) : (
                        <Text
                          x={x}
                          y={y}
                          width={CELL_W}
                          height={CELL_H}
                          align="center"
                          verticalAlign="middle"
                          text="+"
                          fontSize={16}
                          fill="#5c6779"
                          opacity={isHovered ? 1 : 0.5}
                        />
                      )}
                    </Group>
                  );
                })}
              </Group>
            );
          })}
        </Layer>
      </Stage>
    </div>
  );
}
