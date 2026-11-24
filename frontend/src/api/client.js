const GRAPHQL_ENDPOINT = "/graphql";

const LIST_CATEGORIES = "query ListCategories { categories { id name description } }";
const CREATE_CATEGORY = `mutation CreateCategory($name: String!, $description: String) {
  createCategory(name: $name, description: $description) { id name description }
}`;

export class ApiError extends Error {
  constructor(message, status) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request(query, { variables, token, signal } = {}) {
  const headers = { "Content-Type": "application/json" };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  let response;
  try {
    response = await fetch(GRAPHQL_ENDPOINT, {
      method: "POST",
      headers,
      body: JSON.stringify({ query, variables }),
      credentials: "omit",
      signal,
    });
  } catch (cause) {
    if (cause?.name === "AbortError") {
      throw cause;
    }
    throw new ApiError("The backend is not reachable.", 0);
  }

  let payload;
  try {
    payload = await response.json();
  } catch {
    throw new ApiError("The backend returned an unreadable response.", response.status);
  }

  if (payload?.errors?.length) {
    throw new ApiError(String(payload.errors[0].message), response.status);
  }
  if (!response.ok || !payload?.data) {
    throw new ApiError("The request failed.", response.status);
  }
  return payload.data;
}

export async function listCategories(signal) {
  const data = await request(LIST_CATEGORIES, { signal });
  return Array.isArray(data.categories) ? data.categories : [];
}

export async function createCategory({ name, description }, token) {
  const data = await request(CREATE_CATEGORY, {
    variables: { name, description },
    token,
  });
  return data.createCategory;
}
