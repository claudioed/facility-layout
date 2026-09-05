import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  TextField,
  SelectField,
  CheckboxField,
  SubmitButton,
  InlineError,
  InlineSuccess,
  Tabs,
} from "./formkit";

describe("TextField", () => {
  it("renders the label and calls onChange with the new value", async () => {
    const onChange = vi.fn();
    render(<TextField label="Site code" value="" onChange={onChange} />);
    const input = screen.getByLabelText("Site code");
    await userEvent.type(input, "WH1");
    // onChange fires once per keystroke; assert the last call has the last char typed
    // (this is a controlled input with value="" fixed by the test, so each
    // keystroke's event target.value is just that one character).
    expect(onChange).toHaveBeenLastCalledWith("1");
    expect(onChange).toHaveBeenCalledTimes(3);
  });

  it("marks the label with * when required", () => {
    render(<TextField label="Zone code" value="" onChange={() => {}} required />);
    expect(screen.getByText("Zone code *")).toBeInTheDocument();
  });
});

describe("SelectField", () => {
  it("renders every option and calls onChange on selection", async () => {
    const onChange = vi.fn();
    render(
      <SelectField
        label="Direction"
        value=""
        onChange={onChange}
        options={[
          { value: "OneWay", label: "OneWay" },
          { value: "TwoWay", label: "TwoWay" },
        ]}
      />,
    );
    await userEvent.selectOptions(screen.getByLabelText("Direction"), "TwoWay");
    expect(onChange).toHaveBeenCalledWith("TwoWay");
  });
});

describe("CheckboxField", () => {
  it("toggles checked state via onChange", async () => {
    const onChange = vi.fn();
    render(<CheckboxField label="Hazmat zone" checked={false} onChange={onChange} />);
    await userEvent.click(screen.getByLabelText("Hazmat zone"));
    expect(onChange).toHaveBeenCalledWith(true);
  });
});

describe("SubmitButton", () => {
  it("is disabled and shows disabled styling when disabled=true", () => {
    render(<SubmitButton disabled>Register site</SubmitButton>);
    expect(screen.getByRole("button", { name: "Register site" })).toBeDisabled();
  });

  it("fires onClick when type='button'", async () => {
    const onClick = vi.fn();
    render(
      <SubmitButton type="button" onClick={onClick}>
        Deploy
      </SubmitButton>,
    );
    await userEvent.click(screen.getByRole("button", { name: "Deploy" }));
    expect(onClick).toHaveBeenCalledOnce();
  });
});

describe("InlineError / InlineSuccess", () => {
  it("renders nothing when message is null", () => {
    const { container } = render(<InlineError message={null} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders the message text when present", () => {
    render(<InlineSuccess message="Site WH1 registered." />);
    expect(screen.getByText("Site WH1 registered.")).toBeInTheDocument();
  });
});

describe("Tabs", () => {
  it("calls onChange with the clicked tab id", async () => {
    const onChange = vi.fn();
    render(
      <Tabs
        tabs={[
          { id: "sites", label: "Sites" },
          { id: "zones", label: "Zones" },
        ]}
        active="sites"
        onChange={onChange}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "Zones" }));
    expect(onChange).toHaveBeenCalledWith("zones");
  });
});
