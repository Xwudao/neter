# Agent skills (`nr skills`)

`nr skills` writes a generated skill file that teaches an agent how to use the
`nr` CLI in the current project:

```
.agents/skills/neter/SKILL.md
```

The content is tailored to the project's persistence stack, which `nr` detects
the same way the rest of the CLI does:

- `sqlc.yaml` or `db/query/` → the new **PostgreSQL + pgx + sqlc** stack.
- `internal/data/ent/schema/` → the legacy **Ent + MySQL** stack.
- otherwise → unknown, and the skill explains both `--model` and `--ent-name`.

```sh
nr skills                       # detect the stack and write the skill
nr skills --kind ent            # force the legacy section
nr skills --name neter          # choose the directory name
nr skills --dir ./apps/api      # explicit project root
nr skills --print               # print to stdout, write nothing
```

The file is **generated**: `nr skills` always overwrites it, so do not edit it
by hand. To change the wording, edit
`internal/skills/templates/neter.md.tmpl` in the neter repository.

Skills are discovered by pi and other harnesses from `.agents/skills/` in the
project (and its ancestors), so committing the generated file makes the
workflow available to every agent working in the repo.
