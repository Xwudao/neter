# Database migrations (`nr migrate`)

`nr` embeds [golang-migrate](https://github.com/golang-migrate/migrate) so a
project can create paired migration files and apply them to a real database
without compiling the application first.

The new neter template uses PostgreSQL + pgx + sqlc with migrations under
`db/migrations`. Legacy Ent projects keep using `nr new` / `nr gen ent`; the
migration commands below only operate on file-based migrations.

## Commands

| Command | Description |
| --- | --- |
| `nr migrate new <name>` (alias `create`) | Create the next `000NNN_<name>.up.sql` / `.down.sql` pair. |
| `nr migrate up [--steps N]` | Apply all pending migrations, or at most N. |
| `nr migrate down [--all] [--steps N]` | Roll back one (default), N, or all migrations. |
| `nr migrate status` | Print the current version and dirty flag. |
| `nr migrate force <version>` | Pin the schema version to recover a dirty database. |

```sh
nr migrate new add_orders      # db/migrations/000002_add_orders.{up,down}.sql
$EDITOR db/migrations/000002_add_orders.up.sql
nr migrate up
nr migrate status              # version: 2 (clean)
nr migrate down                # back to version 1
```

## DSN resolution

The database DSN is resolved in this order:

1. `--dsn`
2. `NETER_DSN` or `DATABASE_URL`
3. the `db:` block of `config.yml`

The driver is selected from the DSN scheme (`postgres://`, `mysql://`) and
falls back to `db.dialect` in `config.yml`. Postgres is the default.

```sh
# explicit DSN (highest priority)
nr migrate up --dsn 'postgres://user:pass@127.0.0.1:5432/app?sslmode=disable'

# non-default migration directory
nr migrate up --path ./migrations
```

Flags shared by every `nr migrate` database subcommand:

| Flag | Default | Description |
| --- | --- | --- |
| `--dir, -d` | current directory | Project root. |
| `--path` | `db/migrations` | Migration directory, relative to the project root. |
| `--dsn` | – | Explicit DSN. |
| `--config` | `<root>/config.yml` | Config file used for the DSN fallback. |
| `--dialect` | from `config.yml` | `postgres` or `mysql`. |

## Dual-stack behaviour

`nr` detects the persistence layer from the project layout:

- `sqlc.yaml` or `db/query/` → **new** PostgreSQL + sqlc project.
- `internal/data/ent/schema/` → **legacy** Ent project.
- sqlc wins when both markers exist, so a mid-migration project is treated as
  the new stack.

Consequences:

- `nr gen biz --with-crud` emits `sqlc.<Model>` code on new projects
  (`--model Order`) and `ent.<Name>` code on legacy projects (`--ent-name`).
- `nr new`, `nr gen ent`, `nr show ent` fail with guidance on new projects.
- `nr route-info` prints a notice on new projects; it is not maintained for the
  sqlc template.
