import { describe, expect, it, vi } from "vitest";
import { ApiError, createCategory, listCategories } from "../src/api/client.js";

function jsonResponse(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("api client", () => {
  it("lists categories through the same-origin endpoint", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(jsonResponse({ data: { categories: [{ id: 1, name: "A", description: "" }] } }));

    await expect(listCategories()).resolves.toHaveLength(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe("/graphql");
    expect(options.credentials).toBe("omit");
    expect(options.headers.Authorization).toBeUndefined();
  });

  it("sends user input only as GraphQL variables", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(jsonResponse({ data: { createCategory: { id: 2, name: "x", description: "" } } }));
    const payload = '") { id } } #';

    await createCategory({ name: payload, description: "" });
    const body = JSON.parse(fetchMock.mock.calls[0][1].body);
    expect(body.query).not.toContain(payload);
    expect(body.variables.name).toBe(payload);
  });

  it("attaches the write token as a bearer header", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(jsonResponse({ data: { createCategory: { id: 2, name: "x", description: "" } } }));

    await createCategory({ name: "x", description: "" }, "test-token");
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe("Bearer test-token");
  });

  it("surfaces GraphQL errors", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse({ errors: [{ message: "denied" }] }, 401));
    await expect(createCategory({ name: "x" })).rejects.toMatchObject({ message: "denied", status: 401 });
  });

  it("reports an unreachable backend", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new TypeError("network"));
    await expect(listCategories()).rejects.toBeInstanceOf(ApiError);
  });

  it("reports an unreadable response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("not json", { status: 502 }));
    await expect(listCategories()).rejects.toMatchObject({ status: 502 });
  });

  it("rethrows aborts unchanged", async () => {
    const abort = new DOMException("aborted", "AbortError");
    vi.spyOn(globalThis, "fetch").mockRejectedValue(abort);
    await expect(listCategories()).rejects.toBe(abort);
  });
});
