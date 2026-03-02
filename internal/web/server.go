package web

import (
	"encoding/json"
	"net/http"

	"netscope/internal/store"
)

func NewHandler(s *store.MemoryStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.ListLatest())
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(dashboardHTML))
	})

	return mux
}

const dashboardHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>NetScope Dashboard</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 2rem; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
    th { background: #f2f2f2; }
    .up { color: green; font-weight: bold; }
    .down { color: red; font-weight: bold; }
  </style>
</head>
<body>
  <h1>NetScope</h1>
  <p>Read-only network visibility and health view.</p>
  <table>
    <thead>
      <tr><th>Name</th><th>Address</th><th>Type</th><th>Status</th><th>Latency (ms)</th><th>Packet Loss (%)</th><th>Updated</th></tr>
    </thead>
    <tbody id="rows"></tbody>
  </table>
  <script>
    async function refresh() {
      const resp = await fetch('/api/status');
      const data = await resp.json();
      const rows = document.getElementById('rows');
      rows.innerHTML = '';
      data.forEach(item => {
        const tr = document.createElement('tr');
        tr.innerHTML =
          '<td>' + item.name + '</td>' +
          '<td>' + item.address + '</td>' +
          '<td>' + (item.type || '') + '</td>' +
          '<td class="' + (item.online ? 'up' : 'down') + '">' + (item.online ? 'UP' : 'DOWN') + '</td>' +
          '<td>' + item.latency_ms.toFixed(2) + '</td>' +
          '<td>' + item.packet_loss_percent.toFixed(1) + '</td>' +
          '<td>' + new Date(item.updated_at).toLocaleTimeString() + '</td>';
        rows.appendChild(tr);
      });
    }
    refresh();
    setInterval(refresh, 5000);
  </script>
</body>
</html>`
