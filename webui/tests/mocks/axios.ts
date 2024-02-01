/* eslint-disable @typescript-eslint/no-unsafe-member-access */
/* eslint-disable @typescript-eslint/no-unsafe-return */
/* eslint-disable @typescript-eslint/no-unsafe-assignment */
import { jest } from "@jest/globals"

const mockAxios: any = jest.createMockFromModule("axios")

const mockData = {
  data: {
    Jobs: [{ id: "job1" }, { id: "job2" }],
  },
}

mockAxios.create = jest.fn(() => mockAxios)
mockAxios.get = jest.fn(() => Promise.resolve(mockData))

export default mockAxios
