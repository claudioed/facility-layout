import { useMemo, useState } from "react";
import { Stage, Layer, Rect, Text, Group } from "react-konva";

export interface PlannedCell {
  bay: string;
  level: string;
  position: string;
  locationType: string;
  /** Toggled off = this coordinate is a deliberate gap (skipped on deploy),
   *  not every planned cell has to become a real slot. */
  included: boolean;
}

/**
 * Draws a BLANK planned rack -- bays x levels, one or more positions per
 * cell -- before anything is persisted anywhere. This is the "sketch an
 * empty warehouse shape" surface RackPlannerScreen needs: unlike
 * ZoneCanvas (which draws slots that already exist server-side), every
 * cell here is pure client-side plan state until the user hits Deploy.
 *
 * Clicking a cell cycles it through the position's LocationType choices
 * assigned in the parent form, or toggles it out of the plan entirely --
 * see PlannedCell.included.
 */
export function RackPlanCanvas({
  bays,
  levels,
  cells,
  onCellClick,
}: {
  bays: string[];
  levels: string[];
  /** Keyed by `${bay}|${level}|${position}`. */
  cells: Map<string, PlannedCell>;
  onCellClick: (bay: string, level: string) => void;
}) {
  const CELL_W = 96;
  const CELL_H = 56;
  const GUTTER = 6;
  const LABEL_COL_W = 64;
  const LABEL_ROW_H = 30;

  const width = LABEL_COL_W + bays.length * (CELL_W + GUTTER) + GUTTER;
  const height = LABEL_ROW_H + levels.length * (CELL_H + GUTTER) + GUTTER;

  const [hovered, setHovered] = useState<{ b: number; l: number } | null>(null);

  const cellsByCoordinate = useMemo(() => {
    const grouped = new Map<string, PlannedCell[]>();
    for (const cell of cells.values()) {
      const key = `${cell.bay}|${cell.level}`;
      const list = grouped.get(key) ?? [];
      list.push(cell);
      grouped.set(key, list);
    }
    return grouped;
  }, [cells]);

  return (
    <div style={{ overflow: "auto", borderRadius: "var(--wh-radius-md)" }}>
      <Stage width={Math.max(width, 360)} height={Math.max(height, 160)}>
        <Layer>
          {bays.map((bay, b) => (
            <Text
              key={`bay-${bay}`}
              x={LABEL_COL_W + b * (CELL_W + GUTTER) + GUTTER}
              y={0}
              width={CELL_W}
              align="center"
              text={`Bay ${bay}`}
              fontSize={11}
              fill="#5c6779"
              fontFamily="monospace"
            />
          ))}

          {levels.map((level, l) => {
            const y = LABEL_ROW_H + l * (CELL_H + GUTTER) + GUTTER;
            return (
              <Group key={`row-${level}`}>
                <Text
                  x={0}
                  y={y + CELL_H / 2 - 7}
                  width={LABEL_COL_W - 8}
                  align="right"
                  text={`Lvl ${level}`}
                  fontSize={12}
                  fontFamily="monospace"
                  fill="#8f9bad"
                />
                {bays.map((bay, b) => {
                  const x = LABEL_COL_W + b * (CELL_W + GUTTER) + GUTTER;
                  const planned = (cellsByCoordinate.get(`${bay}|${level}`) ?? []).filter((c) => c.included);
                  const isHovered = hovered?.b === b && hovered?.l === l;
                  const hasPlan = planned.length > 0;
                  return (
                    <Group
                      key={`cell-${b}-${l}`}
                      onMouseEnter={() => setHovered({ b, l })}
                      onMouseLeave={() => setHovered(null)}
                      onClick={() => onCellClick(bay, level)}
                      onTap={() => onCellClick(bay, level)}
                    >
                      <Rect
                        x={x}
                        y={y}
                        width={CELL_W}
                        height={CELL_H}
                        cornerRadius={4}
                        fill={hasPlan ? "#16233d" : "transparent"}
                        stroke={hasPlan ? "#4d8dff" : "var(--wh-color-border-subtle)"}
                        strokeWidth={hasPlan ? 1.5 : 1}
                        dash={hasPlan ? undefined : [4, 3]}
                        shadowColor="#000"
                        shadowBlur={isHovered ? 6 : 0}
                        shadowOpacity={0.4}
                      />
                      <Text
                        x={x}
                        y={y}
                        width={CELL_W}
                        height={CELL_H}
                        align="center"
                        verticalAlign="middle"
                        text={hasPlan ? planned.map((c) => c.position).join(" ") : "+"}
                        fontSize={hasPlan ? 12 : 16}
                        fontFamily="monospace"
                        fontStyle={hasPlan ? "600" : "normal"}
                        fill={hasPlan ? "#e6edf5" : "#5c6779"}
                        opacity={hasPlan || isHovered ? 1 : 0.5}
                      />
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
