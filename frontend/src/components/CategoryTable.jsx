export default function CategoryTable({ categories, isLoading }) {
  if (isLoading) {
    return <p role="status">Loading categories...</p>;
  }
  if (categories.length === 0) {
    return <p role="status">No categories yet.</p>;
  }
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th scope="col">Name</th>
            <th scope="col">Description</th>
          </tr>
        </thead>
        <tbody>
          {categories.map((category) => (
            <tr key={category.id}>
              <td>{category.name}</td>
              <td>{category.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
