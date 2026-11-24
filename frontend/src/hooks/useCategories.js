import { useCallback, useEffect, useState } from "react";
import { createCategory, listCategories } from "../api/client.js";

export function useCategories() {
  const [categories, setCategories] = useState([]);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    listCategories(controller.signal)
      .then(setCategories)
      .catch((cause) => {
        if (cause?.name !== "AbortError") {
          setError(cause.message);
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          setIsLoading(false);
        }
      });
    return () => controller.abort();
  }, []);

  const addCategory = useCallback(async (input, token) => {
    setError("");
    try {
      const created = await createCategory(input, token);
      setCategories((previous) => [...previous, created]);
      return true;
    } catch (cause) {
      setError(cause.message);
      return false;
    }
  }, []);

  return { categories, error, isLoading, addCategory };
}
