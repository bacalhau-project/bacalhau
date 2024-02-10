import { v4 as uuidv4 } from "uuid"
import { TaskStatus } from "./taskstatus"

export class Task {
  id: string

  name: string

  createdOn: number

  completedOn?: number

  status: TaskStatus

  constructor(taskName: string) {
    this.id = uuidv4()
    this.name = taskName
    this.createdOn = Date.now()
    this.status = TaskStatus.INCOMPLETE
  }
}
