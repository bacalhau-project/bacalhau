import { randomUUID } from "crypto"
import { TaskStatus } from "./taskstatus"

export class Task {
  id: string

  name: string

  createdOn: number

  completedOn?: number

  status: TaskStatus

  constructor(taskName: string) {
    this.id = randomUUID()
    this.name = taskName
    this.createdOn = Date.now()
    this.status = TaskStatus.INCOMPLETE
  }
}
