package inspect

import "testing"

func TestDetectFramework(t *testing.T) {
	cases := []struct {
		exe, cmd, want string
	}{
		// node frameworks
		{"node", "node /home/x/app/node_modules/.bin/remix dev", "Remix"},
		{"node", "node /home/x/app/node_modules/.bin/svelte-kit dev", "SvelteKit"},
		{"node", "node /home/x/app/node_modules/.bin/solid-start", "SolidStart"},
		{"node", "node /home/x/app/node_modules/.bin/astro dev", "Astro"},
		{"node", "node /home/x/app/node_modules/.bin/hono", "Hono"},
		{"node", "node /home/x/app/node_modules/.bin/elysia", "Elysia"},
		{"node", "next-server (v15.1.0)", "Next.js"},
		{"node", "node /home/x/shop/node_modules/.bin/next dev", "Next.js"},
		{"node", "node /home/x/admin/node_modules/.bin/vite", "Vite"},
		{"node", "node /home/x/admin/node_modules/vite/bin/vite.js --port 5173", "Vite"},
		{"node", "node /home/x/admin/node_modules/.bin/vitest run", "Node.js"}, // not Vite
		{"node", "node server.js", "Node.js"},
		{"node", "node /home/x/app/node_modules/.bin/react-scripts start", "React (CRA)"},
		{"node", "node /home/x/app/node_modules/.bin/expo start", "Expo"},
		{"node", "node /home/x/app/node_modules/.bin/react-native start --port 8081", "React Native"},
		{"node", "node /home/x/app/node_modules/.bin/metro --port 8081", "React Native"},
		{"node", "node /home/x/app/node_modules/.bin/ng serve --port 4200", "Angular"},
		{"node", "node /home/x/app/node_modules/.bin/nest start", "NestJS"},
		{"node", "node /home/x/app/node_modules/.bin/strapi develop", "Strapi"},
		{"node", "node /home/x/app/node_modules/.bin/directus start", "Directus"},
		{"node", "node /home/x/app/node_modules/.bin/prisma studio", "Prisma"},
		{"node", "node /home/x/app/node_modules/.bin/drizzle-kit studio", "Drizzle Kit"},
		// python servers
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/uvicorn app.main:app --reload", "Uvicorn"},
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/fastapi dev app/main.py", "FastAPI"},
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/starlette", "Starlette"},
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/sanic", "Sanic"},
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/quart", "Quart"},
		{"python3.12", "python manage.py runserver 8000", "Django"},
		{"python3.12", "/home/x/.venv/bin/python /home/x/.venv/bin/flask run", "Flask"},
		{"python3.12", "gunicorn: master [app.wsgi:application]", "Gunicorn"},
		// php & ruby servers
		{"php", "php artisan serve --host=0.0.0.0 --port=8000", "Laravel"},
		{"puma", "/home/x/shop/bin/puma -C config/puma.rb", "Rails"},
		{"ruby", "/home/x/shop/bin/rails server -p 3000", "Rails"},
		// go frameworks (detected via cmdline import paths)
		{"main", "/home/x/go/bin/myapp", ""}, // generic go binary, no framework signal
		// databases & servers
		{"postgres", "/usr/lib/postgresql/16/bin/postgres -D /var/lib/postgresql/16/main", "PostgreSQL"},
		{"redis-server", "redis-server *:6379", "Redis"},
		{"surreal", "surreal start --bind 0.0.0.0:8000", "SurrealDB"},
		{"clickhouse", "clickhouse-server --config /etc/clickhouse-server/config.xml", "ClickHouse"},
		{"dragonfly", "dragonfly --port 6379", "Dragonfly"},
		// docker image names route through the same table
		{"postgres", "postgres:16-alpine", "PostgreSQL"},
		{"redis", "redis:7", "Redis"},
		{"mysql", "mysql:8.4", "MySQL"},
		{"mongo", "mongo:7", "MongoDB"},
		{"nginx", "nginx: master process /usr/sbin/nginx", "nginx"},
		{"main", "/tmp/go-build2384912/b001/exe/main", "Go (go run)"},
		{"docker-proxy", "/usr/bin/docker-proxy -proto tcp -host-port 5432", ""}, // M4's job
		{"python3.12", "python3 -m http.server 8123", ""},                        // no signal: fall back to exe
		// infrastructure, before their generic runtimes
		{"valkey-server", "valkey-server *:6379", "Valkey"},
		{"memcached", "memcached -p 11211 -u memcached", "Memcached"},
		{"etcd", "/usr/bin/etcd --listen-client-urls http://0.0.0.0:2379", "etcd"},
		{"nats-server", "nats-server -p 4222", "NATS"},
		{"minio", "/usr/bin/minio server /data --address :9000", "MinIO"},
		{"consul", "consul agent -dev -http-port 8500", "Consul"},
		{"beam.smp", "/usr/lib/erlang/erts-14.2/bin/beam.smp -- -root /usr/lib -progname erl -- -home /var --eval phx.server", "Phoenix"},
		{"beam.smp", "/usr/lib/rabbitmq/bin/erl -boot rabbit", "RabbitMQ"},
		{"java", "java -cp app.jar org.springframework.boot.loader.launch.JarLauncher", "Spring Boot"},
		{"dotnet", "/usr/share/dotnet/dotnet /app/shop.dll", ".NET"},
		// local kubernetes & dev conveniences
		{"kube-apiserver", "kube-apiserver --secure-port 6443 --advertise-address 10.0.0.1", "kube-apiserver"},
		{"mailpit", "/usr/bin/mailpit --smtp :1025 --listen 0.0.0.0:8025", "Mailpit"},
		{"node", "node /home/x/app/node_modules/.bin/wrangler dev", "Wrangler"},
		{"workerd", "workerd serve capnp-config", "Wrangler"},
		{"node", "node /home/x/app/node_modules/.bin/supabase start", "Supabase"},
		// runtime exes
		{"bun", "bun run dev", "Bun"},
		{"deno", "deno run --allow-net server.ts", "Deno"},
		{"java", "java -jar target/app-server.jar", "Java"},
		// the letter-suffixed binaries the table names explicitly
		{"mysqld", "mysqld --datadir /var/lib/mysql", "MySQL"},
		{"mariadbd", "mariadbd --datadir /var/lib/mysql", "MariaDB"},
		{"mongod", "/usr/bin/mongod --bind_ip 127.0.0.1", "MongoDB"},
		{"mongos", "mongos --configdb cfg/rs0", "MongoDB"},
		{"nodejs", "nodejs server.js", "Node.js"},
		{"javaw", "javaw -jar app.jar", "Java"},
		// hyphens and underscores end a match — these are different
		// programs, never Node.js, never stoppable as such
		{"node_exporter", "/usr/bin/node_exporter --web.listen-address=:9100", ""},
		{"node-exporter", "quay.io/prometheus/node-exporter", ""},
		{"redis_exporter", "redis_exporter --redis.addr redis:6379", ""},
		// versions and extensions still count
		{"node.exe", "node.exe server.js", "Node.js"},
		{"python3", "python3 -m http.server", ""}, // no Python rule: exe alone is not a signal
	}
	for _, c := range cases {
		if got := DetectFramework(c.exe, c.cmd); got != c.want {
			t.Errorf("DetectFramework(%q, %q) = %q, want %q", c.exe, c.cmd, got, c.want)
		}
	}
}
