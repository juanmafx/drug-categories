import CategoryForm from "./components/CategoryForm.jsx";
import CategoryTable from "./components/CategoryTable.jsx";
import { useCategories } from "./hooks/useCategories.js";

export default function App() {
  const { categories, error, isLoading, addCategory } = useCategories();

  return (
    <main className="page-shell">
      <section className="app-card hero-card">
        <p className="eyebrow">React + Go + GraphQL</p>
        <h1>Drug Categories</h1>
        <p className="subtitle">Browse and add drug categories through a GraphQL API.</p>
      </section>

      <section className="app-card">
        <h2>Add Category</h2>
        {error ? (
          <p className="error-box" role="alert">
            {error}
          </p>
        ) : null}
        <CategoryForm onSubmit={addCategory} />
      </section>

      <section className="app-card">
        <h2>Category List</h2>
        <CategoryTable categories={categories} isLoading={isLoading} />
      </section>
    </main>
  );
}
