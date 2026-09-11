import { spawnSync } from "node:child_process";
import type { TaskID } from "./types.ts";

const PY = `
import json, sqlite3, sys
db = sqlite3.connect(sys.argv[1])
db.row_factory = sqlite3.Row
cur = db.execute(sys.argv[2], json.loads(sys.argv[3]))
print(json.dumps([dict(r) for r in cur.fetchall()]))
`;

export type OracleResult = {
  ok: boolean;
  reason: string;
  expected?: number;
};

export function checkOracle(
  dbPath: string,
  task: TaskID,
  answer: number | null,
  usedSearch: boolean,
): OracleResult {
  switch (task) {
    case "update-dan": {
      const rows = sqlAll(dbPath, "SELECT last_name AS last FROM contacts WHERE first_name = ? COLLATE NOCASE", [
        "Dan",
      ]);
      if (rows.length === 1 && String(rows[0]?.last) === "Hibiki") {
        return { ok: true, reason: "Dan last name is Hibiki" };
      }
      return { ok: false, reason: `Dan last name = ${JSON.stringify(rows)}` };
    }
    case "create-ken": {
      const rows = sqlAll(
        dbPath,
        "SELECT COUNT(*) AS n FROM contacts WHERE first_name = ? COLLATE NOCASE AND last_name = ? COLLATE NOCASE",
        ["Ken", "Masters"],
      );
      if (Number(rows[0]?.n) >= 1) {
        return { ok: true, reason: "Ken Masters exists" };
      }
      return { ok: false, reason: "Ken Masters missing" };
    }
    case "delete-chun-li": {
      const rows = sqlAll(
        dbPath,
        "SELECT COUNT(*) AS n FROM contacts WHERE first_name = ? COLLATE NOCASE OR (first_name = ? COLLATE NOCASE AND last_name = ? COLLATE NOCASE)",
        ["Chun-Li", "Chun", "Li"],
      );
      if (Number(rows[0]?.n) === 0) {
        return { ok: true, reason: "Chun-Li deleted" };
      }
      return { ok: false, reason: "Chun-Li still present" };
    }
    case "count-ryu": {
      const rows = sqlAll(
        dbPath,
        "SELECT COUNT(*) AS n FROM contacts WHERE first_name LIKE ? COLLATE NOCASE OR last_name LIKE ? COLLATE NOCASE",
        ["%Ryu%", "%Ryu%"],
      );
      const expected = Number(rows[0]?.n);
      if (answer !== expected) {
        return { ok: false, reason: `answer ${answer} != ${expected}`, expected };
      }
      if (!usedSearch) {
        return { ok: false, reason: "counted without using search on a paginated list", expected };
      }
      return { ok: true, reason: `Ryu count ${expected}`, expected };
    }
  }
}

export function sqlAll(dbPath: string, sql: string, params: unknown[] = []): Record<string, unknown>[] {
  const result = spawnSync("python3", ["-c", PY, dbPath, sql, JSON.stringify(params)], {
    encoding: "utf8",
  });
  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || "sqlite query failed");
  }
  return JSON.parse(result.stdout) as Record<string, unknown>[];
}
