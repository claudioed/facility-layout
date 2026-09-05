import type { ReactNode } from "react";

// Minimal test double for react-konva: renders each primitive as a plain
// <div>/<span> carrying its props as data-* attributes and forwarding
// onClick/onTap, so component tests can query by role/text/data-testid
// and fire click events without a real canvas 2D context (jsdom has
// none). This intentionally does NOT try to visually replicate Konva --
// it exists only to make ZoneCanvas/RackPlanCanvas's OWN logic
// (which cell maps to which callback args) testable.

export function Stage({
  children,
  width,
  height,
}: {
  children: ReactNode;
  width: number;
  height: number;
}) {
  return (
    <div data-testid="konva-stage" data-width={width} data-height={height}>
      {children}
    </div>
  );
}

export function Layer({ children }: { children: ReactNode }) {
  return <div data-testid="konva-layer">{children}</div>;
}

export function Group({
  children,
  onClick,
  onTap,
  onMouseEnter,
  onMouseLeave,
  ...rest
}: {
  children?: ReactNode;
  onClick?: () => void;
  onTap?: () => void;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
  [key: string]: unknown;
}) {
  return (
    <div
      data-testid="konva-group"
      onClick={onClick}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      {...toDataAttrs(rest)}
    >
      {children}
    </div>
  );
}

export function Rect(props: Record<string, unknown>) {
  return <div data-testid="konva-rect" {...toDataAttrs(props)} />;
}

export function Text({ text, ...rest }: { text: string; [key: string]: unknown }) {
  return (
    <span data-testid="konva-text" {...toDataAttrs(rest)}>
      {text}
    </span>
  );
}

function toDataAttrs(props: Record<string, unknown>) {
  const out: Record<string, string> = {};
  for (const [key, value] of Object.entries(props)) {
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      out[`data-${key.toLowerCase()}`] = String(value);
    }
  }
  return out;
}
