import { useState } from "react";

const MAX_NAME_LENGTH = 120;
const MAX_DESCRIPTION_LENGTH = 1000;

export default function CategoryForm({ onSubmit }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [token, setToken] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event) {
    event.preventDefault();
    setIsSubmitting(true);
    const succeeded = await onSubmit({ name, description }, token);
    setIsSubmitting(false);
    if (succeeded) {
      setName("");
      setDescription("");
    }
  }

  return (
    <form onSubmit={handleSubmit} className="category-form">
      <label>
        Name
        <input
          value={name}
          maxLength={MAX_NAME_LENGTH}
          onChange={(event) => setName(event.target.value)}
          required
        />
      </label>
      <label>
        Description
        <input
          value={description}
          maxLength={MAX_DESCRIPTION_LENGTH}
          onChange={(event) => setDescription(event.target.value)}
        />
      </label>
      <button type="submit" disabled={isSubmitting}>
        {isSubmitting ? "Creating..." : "Create"}
      </button>
      <label className="token-field">
        Write token (only if the server requires one)
        <input
          type="password"
          autoComplete="off"
          value={token}
          onChange={(event) => setToken(event.target.value)}
        />
      </label>
    </form>
  );
}
