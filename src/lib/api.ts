export function getApiBase() {
  const base = import.meta.env.PUBLIC_API_BASE ?? "";
  return base.endsWith("/") ? base.slice(0, -1) : base;
}
