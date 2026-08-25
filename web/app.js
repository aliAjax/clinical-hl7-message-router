const endpoints = [
  ["health", "/healthz"],
  ["ready", "/readyz"],
  ["metrics", "/metrics"],
];

async function refresh() {
  const results = await Promise.all(endpoints.map(async ([name, path]) => {
    const cell = document.querySelector(`#${name}-status`);
    cell.className = "";
    cell.textContent = "Checking";
    try {
      const response = await fetch(path, { cache: "no-store" });
      cell.textContent = response.ok ? "Available" : `HTTP ${response.status}`;
      cell.className = response.ok ? "ok" : "failed";
      return response.ok;
    } catch {
      cell.textContent = "Unavailable";
      cell.className = "failed";
      return false;
    }
  }));

  const online = results[0];
  const health = document.querySelector(".health");
  health.className = online ? "health online" : "health offline";
  document.querySelector("#health-label").textContent = online ? "Gateway online" : "Gateway unavailable";
}

document.querySelector("#refresh").addEventListener("click", refresh);
refresh();
