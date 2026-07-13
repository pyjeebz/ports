package inspect

import "testing"

func TestDetectFramework(t *testing.T) {
	cases := []struct {
		exe, cmd, want string
	}{
		{"node", "next-server (v15.1.0)", "Next.js"},
		{"node", "node /home/x/shop/node_modules/.bin/next dev", "Next.js"},
		{"node", "node /home/x/admin/node_modules/.bin/vite", "Vite"},
		{"node", "node /home/x/admin/node_modules/vite/bin/vite.js --port 5173", "Vite"},
		{"node", "node /home/x/admin/node_modules/.bin/vitest run", "Node.js"}, // not Vite
		{"node", "node server.js", "Node.js"},
		{"node", "node /home/x/app/node_modules/.bin/react-scripts start", "React (CRA)"},
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/uvicorn app.main:app --reload", "Uvicorn"},
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/fastapi dev app/main.py", "FastAPI"},
		{"python3.12", "python manage.py runserver 8000", "Django"},
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/flask run", "Flask"},
		{"python3.12", "gunicorn: master [app.wsgi:application]", "Gunicorn"},
		{"postgres", "/usr/lib/postgresql/16/bin/postgres -D /var/lib/postgresql/16/main", "PostgreSQL"},
		{"redis-server", "redis-server *:6379", "Redis"},
		// docker image names route through the same table
		{"postgres", "postgres:16-alpine", "PostgreSQL"},
		{"redis", "redis:7", "Redis"},
		{"mysql", "mysql:8.4", "MySQL"},
		{"mongo", "mongo:7", "MongoDB"},
		{"nginx", "nginx: master process /usr/sbin/nginx", "nginx"},
		{"main", "/tmp/go-build2384912/b001/exe/main", "Go (go run)"},
		{"docker-proxy", "/usr/bin/docker-proxy -proto tcp -host-port 5432", ""}, // M4's job
		{"python3.12", "python3 -m http.server 8123", ""},                        // no signal: fall back to exe
	}
	for _, c := range cases {
		if got := DetectFramework(c.exe, c.cmd); got != c.want {
			t.Errorf("DetectFramework(%q, %q) = %q, want %q", c.exe, c.cmd, got, c.want)
		}
	}
}
