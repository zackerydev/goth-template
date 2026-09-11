import { TASK_IDS, type Task, type TaskID } from "./types.ts";

const prompts: Record<TaskID, string> = {
  "update-dan": "Update contact Dan's last name to Hibiki",
  "create-ken": "Create a contact for Ken Masters",
  "delete-chun-li": "Delete Chun-Li's contact",
  "count-ryu": "How many contacts named Ryu do I have?",
};

export function allTasks(): Task[] {
  return TASK_IDS.map((id) => ({ id, prompt: prompts[id] }));
}

export function getTask(id: TaskID): Task {
  return { id, prompt: prompts[id] };
}
