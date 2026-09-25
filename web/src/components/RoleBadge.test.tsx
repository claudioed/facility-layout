import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RoleBadge, RoleFilterChips } from "./RoleBadge";

describe("RoleBadge", () => {
  it("renders the role's glyph and includes the role in its title", () => {
    render(<RoleBadge role="Dock" />);
    const badge = screen.getByText("D");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("title", "Dock");
  });

  it("appends dockFlow to the title when present", () => {
    render(<RoleBadge role="Dock" dockFlow="Outbound" />);
    expect(screen.getByText("D")).toHaveAttribute("title", "Dock (Outbound)");
  });

  it("falls back to Storage's glyph for an unknown role", () => {
    render(<RoleBadge role="SomethingNew" />);
    expect(screen.getByText("S")).toBeInTheDocument();
  });
});

describe("RoleFilterChips", () => {
  it("renders an 'All roles' chip plus one chip per role", () => {
    render(<RoleFilterChips active={null} onChange={vi.fn()} />);
    expect(screen.getByRole("button", { name: "All roles" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Dock/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Work Center/ })).toBeInTheDocument();
  });

  it("marks the active role's chip as pressed", () => {
    render(<RoleFilterChips active="Dock" onChange={vi.fn()} />);
    expect(screen.getByRole("button", { name: /Dock/ })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "All roles" })).toHaveAttribute("aria-pressed", "false");
  });

  it("calls onChange with the clicked role", async () => {
    const onChange = vi.fn();
    render(<RoleFilterChips active={null} onChange={onChange} />);
    await userEvent.click(screen.getByRole("button", { name: /Yard/ }));
    expect(onChange).toHaveBeenCalledWith("Yard");
  });

  it("clicking the already-active role clears the filter", async () => {
    const onChange = vi.fn();
    render(<RoleFilterChips active="Yard" onChange={onChange} />);
    await userEvent.click(screen.getByRole("button", { name: /Yard/ }));
    expect(onChange).toHaveBeenCalledWith(null);
  });

  it("clicking 'All roles' calls onChange with null", async () => {
    const onChange = vi.fn();
    render(<RoleFilterChips active="Dock" onChange={onChange} />);
    await userEvent.click(screen.getByRole("button", { name: "All roles" }));
    expect(onChange).toHaveBeenCalledWith(null);
  });
});
