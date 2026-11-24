import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import App from "../src/App.jsx";

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

const seed = { data: { categories: [{ id: 1, name: "Analgesics", description: "Pain relief drugs" }] } };

describe("App", () => {
  it("renders categories from the API", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse(seed));
    render(<App />);
    expect(await screen.findByText("Analgesics")).toBeInTheDocument();
  });

  it("shows an empty state", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse({ data: { categories: [] } }));
    render(<App />);
    expect(await screen.findByText("No categories yet.")).toBeInTheDocument();
  });

  it("shows an error when the backend is down", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new TypeError("network"));
    render(<App />);
    expect(await screen.findByRole("alert")).toHaveTextContent("not reachable");
  });

  it("creates a category and clears the form but keeps the token", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(jsonResponse(seed))
      .mockResolvedValueOnce(
        jsonResponse({ data: { createCategory: { id: 2, name: "Antivirals", description: "Viral" } } }),
      );
    render(<App />);
    await screen.findByText("Analgesics");

    await user.type(screen.getByLabelText("Name"), "Antivirals");
    await user.type(screen.getByLabelText("Description"), "Viral");
    await user.type(screen.getByLabelText(/Write token/), "test-token");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByText("Antivirals")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByLabelText("Name")).toHaveValue(""));
    expect(screen.getByLabelText(/Write token/)).toHaveValue("test-token");
  });

  it("keeps input and shows the error when creation fails", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(jsonResponse(seed))
      .mockResolvedValueOnce(jsonResponse({ errors: [{ message: "already exists" }] }));
    render(<App />);
    await screen.findByText("Analgesics");

    await user.type(screen.getByLabelText("Name"), "Analgesics");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("already exists");
    expect(screen.getByLabelText("Name")).toHaveValue("Analgesics");
  });

  it("renders stored markup as inert text", async () => {
    const payload = '<img src=x onerror="alert(1)">';
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse({ data: { categories: [{ id: 1, name: payload, description: "" }] } }),
    );
    const { container } = render(<App />);
    expect(await screen.findByText(payload)).toBeInTheDocument();
    expect(container.querySelector("img")).toBeNull();
  });

  it("never persists the token in browser storage", async () => {
    const user = userEvent.setup();
    const setItem = vi.spyOn(Storage.prototype, "setItem");
    vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse(seed));
    render(<App />);
    await screen.findByText("Analgesics");
    await user.type(screen.getByLabelText(/Write token/), "test-token");
    expect(setItem).not.toHaveBeenCalled();
  });
});
