import type { ChangeEvent, ReactNode } from "react";

/**
 * Minimal styled form primitives shared by every Configure panel. Not
 * promoted to @warehouse/ui-kit (out of scope -- this task only touches
 * facility-mfe's own web/ directory) but intentionally styled purely off
 * the shared design tokens so it looks native inside the console shell.
 */

const fieldWrapStyle = {
  display: "flex",
  flexDirection: "column" as const,
  gap: 4,
};

const labelStyle = {
  fontSize: "var(--wh-font-size-xs)",
  color: "var(--wh-color-text-muted)",
  fontWeight: 600,
};

const inputStyle = {
  padding: "8px 10px",
  borderRadius: "var(--wh-radius-md)",
  border: "1px solid var(--wh-color-border)",
  background: "var(--wh-color-bg-sunken)",
  color: "var(--wh-color-text)",
  fontFamily: "var(--wh-font-mono)",
  fontSize: "var(--wh-font-size-sm)",
};

export function TextField({
  label,
  value,
  onChange,
  placeholder,
  type = "text",
  required,
  list,
}: {
  label: string;
  value: string | number;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: "text" | "number";
  required?: boolean;
  /** Wires up an HTML5 <datalist> id for autocomplete suggestions without
   *  forcing a hard-select dropdown -- e.g. "type WH2 or pick an existing
   *  site code" in the Rack Planner. */
  list?: string;
}) {
  return (
    <label style={fieldWrapStyle}>
      <span style={labelStyle}>
        {label}
        {required && " *"}
      </span>
      <input
        type={type}
        value={value}
        placeholder={placeholder}
        onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(e.target.value)}
        style={inputStyle}
        list={list}
      />
    </label>
  );
}

export function SelectField({
  label,
  value,
  onChange,
  options,
  placeholder = "Select…",
  required,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
  placeholder?: string;
  required?: boolean;
}) {
  return (
    <label style={fieldWrapStyle}>
      <span style={labelStyle}>
        {label}
        {required && " *"}
      </span>
      <select
        value={value}
        onChange={(e: ChangeEvent<HTMLSelectElement>) => onChange(e.target.value)}
        style={{ ...inputStyle, fontFamily: "var(--wh-font-sans)" }}
      >
        <option value="" disabled>
          {placeholder}
        </option>
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </label>
  );
}

export function CheckboxField({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <label
      style={{
        display: "flex",
        alignItems: "center",
        gap: 8,
        fontSize: "var(--wh-font-size-sm)",
        color: "var(--wh-color-text)",
        cursor: "pointer",
      }}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
      />
      {label}
    </label>
  );
}

export function FormRow({ children }: { children: ReactNode }) {
  return (
    <div style={{ display: "flex", gap: "var(--wh-space-3)", flexWrap: "wrap", alignItems: "end" }}>
      {children}
    </div>
  );
}

export function SubmitButton({
  children,
  disabled,
  onClick,
  type = "submit",
}: {
  children: ReactNode;
  disabled?: boolean;
  /** Set together with type="button" to use this as a plain action button
   *  outside a <form> submit flow (e.g. a second "Deploy" step gated on
   *  reviewing a generated plan, not on filling the first form again). */
  onClick?: () => void;
  type?: "submit" | "button";
}) {
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      style={{
        padding: "9px 16px",
        borderRadius: "var(--wh-radius-md)",
        border: "none",
        background: disabled ? "var(--wh-color-border)" : "var(--wh-color-accent)",
        color: "#fff",
        fontWeight: 600,
        fontSize: "var(--wh-font-size-sm)",
        cursor: disabled ? "not-allowed" : "pointer",
        height: 36,
      }}
    >
      {children}
    </button>
  );
}

export function InlineError({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <div
      style={{
        color: "var(--wh-color-status-danger)",
        background: "var(--wh-color-status-danger-bg)",
        borderRadius: "var(--wh-radius-md)",
        padding: "8px 12px",
        fontSize: "var(--wh-font-size-sm)",
      }}
    >
      {message}
    </div>
  );
}

export function InlineSuccess({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <div
      style={{
        color: "var(--wh-color-status-success)",
        background: "var(--wh-color-status-success-bg)",
        borderRadius: "var(--wh-radius-md)",
        padding: "8px 12px",
        fontSize: "var(--wh-font-size-sm)",
      }}
    >
      {message}
    </div>
  );
}

export interface TabItem {
  id: string;
  label: string;
}

export function Tabs({
  tabs,
  active,
  onChange,
}: {
  tabs: TabItem[];
  active: string;
  onChange: (id: string) => void;
}) {
  return (
    <div
      style={{
        display: "flex",
        gap: "var(--wh-space-1)",
        borderBottom: "1px solid var(--wh-color-border)",
      }}
    >
      {tabs.map((tab) => {
        const isActive = tab.id === active;
        return (
          <button
            key={tab.id}
            type="button"
            onClick={() => onChange(tab.id)}
            style={{
              padding: "10px 14px",
              background: "transparent",
              border: "none",
              borderBottom: isActive
                ? "2px solid var(--wh-color-accent)"
                : "2px solid transparent",
              color: isActive ? "var(--wh-color-text)" : "var(--wh-color-text-muted)",
              fontWeight: isActive ? 600 : 500,
              fontSize: "var(--wh-font-size-sm)",
              cursor: "pointer",
            }}
          >
            {tab.label}
          </button>
        );
      })}
    </div>
  );
}
