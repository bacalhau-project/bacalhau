import React, {
  useState,
  createContext,
  useContext,
  useEffect,
  ReactNode,
} from "react"

// Combined table settings interface
export interface TableSettings {
  // Jobs Table
  showJobName?: boolean
  showCreated?: boolean
  showProgram?: boolean
  showJobType?: boolean
  showLabel?: boolean
  showStatus?: boolean
  // Nodes Table
  showNodeId?: boolean
  showNodeType?: boolean
  showEnv?: boolean
  showInputs?: boolean
  showOutputs?: boolean
  showVersion?: boolean
  showAction?: boolean
}

interface TableSettingsContextType {
  settings: TableSettings
  toggleSetting: (key: keyof TableSettings) => void
}

const defaultState: TableSettings = {
  // Jobs Table
  showJobName: true,
  showCreated: true,
  showProgram: true,
  showJobType: true,
  showLabel: true,
  showStatus: true,
  // Nodes Table
  showNodeId: true,
  showNodeType: true,
  showEnv: true,
  showInputs: true,
  showOutputs: true,
  showVersion: true,
  showAction: true,
}

const TableSettingsContext = createContext<TableSettingsContextType>({
  settings: defaultState,
  toggleSetting: (key: keyof TableSettings) => {
    console.log("toggleSetting not implemented: %s", key)
  },
})

export const TableSettingsProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [settings, setSettings] = useState<TableSettings>(defaultState)

  useEffect(() => {
    const loadSettings = () => {
      const storedSettings = localStorage.getItem("tableSettings")
      if (storedSettings) {
        setSettings(JSON.parse(storedSettings))
      }
    }

    if (typeof window !== "undefined") {
      loadSettings()
    }
  }, [])

  const toggleSetting = (key: keyof TableSettings) => {
    setSettings((prev) => {
      const newSettings = { ...prev, [key]: !prev[key] }
      localStorage.setItem("tableSettings", JSON.stringify(newSettings))
      return newSettings
    })
  }

  return (
    <TableSettingsContext.Provider value={{ settings, toggleSetting }}>
      {children}
    </TableSettingsContext.Provider>
  )
}

export const useTableSettings = (): TableSettingsContextType => {
  const context = useContext(TableSettingsContext)
  if (!context) {
    throw new Error(
      "useTableSettings must be used within a TableSettingsProvider"
    )
  }
  return context
}

export default TableSettingsContext
