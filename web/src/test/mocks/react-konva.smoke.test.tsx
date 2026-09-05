import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { Stage, Layer, Rect } from "react-konva";

describe("react-konva mock", () => {
  it("resolves to the test double, not the real package", () => {
    render(
      <Stage width={100} height={100}>
        <Layer>
          <Rect x={0} y={0} width={10} height={10} />
        </Layer>
      </Stage>,
    );
    expect(screen.getByTestId("konva-stage")).toBeInTheDocument();
    expect(screen.getByTestId("konva-rect")).toBeInTheDocument();
  });
});
