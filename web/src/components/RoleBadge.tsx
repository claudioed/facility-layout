import { ROLE_META, ROLE_ORDER, roleMeta } from "../roles";
import type { LocationRole } from "../types";

/**
 * A compact glyph badge for one LocationSlot's role (ADR-0016). Rendered
 * next to a slot chip in FacilityScreen's aisle listing and in the slot
 * detail panel. Storage renders as a faint, near-invisible marker since it
 * is the pre-ADR-0016 default and the overwhelming majority of slots --
 * every OTHER role is the thing this badge exists to surface.
 */
export function RoleBadge({ role, dockFlow }: { role: string; dockFlow?: string }) {
  const meta = roleMeta(role);
  const title = dockFlow ? `${meta.label} (${dockFlow})` : meta.label;
  return (
    <span
      title={title}
      data-role={role}
      style={{
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        width: 16,
        height: 16,
        borderRadius: "var(--wh-radius-sm)",
        fontSize: "var(--wh-font-size-xs)",
        fontWeight: 700,
        fontFamily: "var(--wh-font-mono)",
        color: meta.fg,
        background: meta.bg,
        flexShrink: 0,
      }}
    >
      {meta.glyph}
    </span>
  );
}

/**
 * A row of toggleable role filter chips, used to narrow FacilityScreen's
 * layout tree to only the slots carrying a given role -- "show me just
 * this site's dock doors" without scrolling past hundreds of ordinary
 * storage slots. `null` (the default) means no filter, showing every role.
 */
export function RoleFilterChips({
  active,
  onChange,
}: {
  active: LocationRole | null;
  onChange: (role: LocationRole | null) => void;
}) {
  return (
    <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--wh-space-2)" }}>
      <button
        type="button"
        onClick={() => onChange(null)}
        aria-pressed={active === null}
        style={chipStyle(active === null, "var(--wh-color-text-muted)", "var(--wh-color-bg-sunken)")}
      >
        All roles
      </button>
      {ROLE_ORDER.map((role) => {
        const meta = ROLE_META[role];
        return (
          <button
            key={role}
            type="button"
            onClick={() => onChange(active === role ? null : role)}
            aria-pressed={active === role}
            style={chipStyle(active === role, meta.fg, meta.bg)}
          >
            {meta.glyph} {meta.label}
          </button>
        );
      })}
    </div>
  );
}

function chipStyle(isActive: boolean, fg: string, bg: string) {
  return {
    display: "inline-flex",
    alignItems: "center",
    gap: 4,
    padding: "4px 10px",
    borderRadius: "var(--wh-radius-pill)",
    border: isActive ? `1px solid ${fg}` : "1px solid var(--wh-color-border-subtle)",
    background: isActive ? bg : "transparent",
    color: isActive ? fg : "var(--wh-color-text-muted)",
    fontSize: "var(--wh-font-size-xs)",
    fontWeight: isActive ? 700 : 500,
    cursor: "pointer" as const,
  };
}
