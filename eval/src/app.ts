import { spawn, type ChildProcess } from "node:child_process";
import { createServer } from "node:net";
import { setTimeout as delay } from "node:timers/promises";
import path from "node:path";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";

export type RunningApp = {
  origin: string;
  database: string;
  stop: () => Promise<void>;
};

export async function startApp(repoRoot: string, binary?: string): Promise<RunningApp> {
  const directory = await mkdtemp(path.join(tmpdir(), "agent-eval-"));
  const database = path.join(directory, "contacts.db");
  const port = await freePort();
  const child = spawn(binary ?? "go", binary ? [] : ["run", "./cmd/app"], {
    cwd: repoRoot,
    env: {
      ...process.env,
      APP_DATABASE: database,
      APP_SEED: "eval",
      APP_PORT: String(port),
      PORT: String(port),
    },
    stdio: ["ignore", "pipe", "pipe"],
  });
  child.stdout?.resume();
  let stderr = "";
  child.stderr?.on("data", (chunk: Buffer | string) => {
    stderr += String(chunk);
  });
  const origin = `http://127.0.0.1:${port}`;
  try {
    await waitFor(origin, child, () => stderr);
  } catch (error) {
    await stopProcess(child);
    await rm(directory, { recursive: true, force: true });
    throw error;
  }
  return {
    origin,
    database,
    stop: async () => {
      await stopProcess(child);
      await rm(directory, { recursive: true, force: true });
    },
  };
}

async function waitFor(origin: string, child: ChildProcess, stderr: () => string): Promise<void> {
  const deadline = Date.now() + 60_000;
  let last = "";
  while (Date.now() < deadline) {
    if (child.exitCode !== null) {
      throw new Error(`app exited ${child.exitCode}: ${stderr() || last}`);
    }
    try {
      const response = await fetch(`${origin}/contacts`);
      if (response.ok) {
        return;
      }
      last = `${response.status}`;
    } catch (error) {
      last = error instanceof Error ? error.message : String(error);
    }
    await delay(200);
  }
  throw new Error(`app did not start: ${stderr() || last}`);
}

async function freePort(): Promise<number> {
  return await new Promise((resolve, reject) => {
    const server = createServer();
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        server.close();
        reject(new Error("no port"));
        return;
      }
      const port = address.port;
      server.close((error) => {
        if (error) {
          reject(error);
          return;
        }
        resolve(port);
      });
    });
    server.on("error", reject);
  });
}

async function stopProcess(child: ChildProcess): Promise<void> {
  if (child.exitCode !== null) {
    return;
  }
  child.kill("SIGTERM");
  const deadline = Date.now() + 3000;
  while (child.exitCode === null && Date.now() < deadline) {
    await delay(50);
  }
  if (child.exitCode === null) {
    child.kill("SIGKILL");
  }
}
